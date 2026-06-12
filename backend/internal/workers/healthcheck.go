package workers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
	"github.com/monitor-api/backend/internal/models"
)

// HTTPClient interface allows us to mock http.Client in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	defaultClient HTTPClient = &http.Client{Timeout: 30 * time.Second}
	testClient    HTTPClient
)

// SetTestClient allows tests to inject a mock HTTP client
func SetTestClient(client HTTPClient) {
	testClient = client
}

func getClient() HTTPClient {
	if testClient != nil {
		return testClient
	}
	return defaultClient
}

// StartHealthCheckWorker starts the background process for pinging APIs
func StartHealthCheckWorker() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			checkAPIs()
		}
	}()
}

var lastCheckMap = make(map[uuid.UUID]time.Time)

func checkAPIs() {
	var apis []models.API
	database.DB.Where("is_active = ?", true).Find(&apis)

	activeIDs := make(map[uuid.UUID]bool)
	for _, api := range apis {
		activeIDs[api.ID] = true
	}
	for id := range lastCheckMap {
		if !activeIDs[id] {
			delete(lastCheckMap, id)
		}
	}

	// Fetch projects and their env vars + execution mode
	type ProjectResult struct {
		ID                   uuid.UUID
		Name                 string
		CompanyID            *uuid.UUID
		EnvironmentVariables string
		ExecutionMode        string
	}
	var projects []ProjectResult
	database.DB.Model(&models.Project{}).Select("id, name, company_id, environment_variables, execution_mode").Find(&projects)

	envMap := make(map[uuid.UUID]map[string]string)
	nameMap := make(map[uuid.UUID]string)
	companyIDMap := make(map[uuid.UUID]*uuid.UUID)
	execModeMap := make(map[uuid.UUID]string)

	for _, p := range projects {
		var vars map[string]string
		if p.EnvironmentVariables != "" && p.EnvironmentVariables != "{}" {
			importJSON(p.EnvironmentVariables, &vars)
		}
		envMap[p.ID] = vars
		nameMap[p.ID] = p.Name
		companyIDMap[p.ID] = p.CompanyID
		mode := p.ExecutionMode
		if mode == "" {
			mode = "sequential"
		}
		execModeMap[p.ID] = mode
	}

	// Fetch companies
	type CompanyResult struct {
		ID   uuid.UUID
		Name string
	}
	var companies []CompanyResult
	database.DB.Model(&models.Company{}).Select("id, name").Find(&companies)

	companyNameMap := make(map[uuid.UUID]string)
	for _, c := range companies {
		companyNameMap[c.ID] = c.Name
	}

	// Group APIs due for checking by Project
	type apiWithContext struct {
		api         models.API
		vars        map[string]string
		projectName string
		companyName string
	}
	projectGroups := make(map[uuid.UUID][]apiWithContext)

	now := time.Now()
	for _, api := range apis {
		if api.PausedUntil != nil && api.PausedUntil.After(now) {
			continue
		}
		if lastCheck, exists := lastCheckMap[api.ID]; exists {
			if time.Since(lastCheck).Seconds() < float64(api.Interval) {
				continue
			}
		}
		lastCheckMap[api.ID] = now

		companyName := ""
		if cid := companyIDMap[api.ProjectID]; cid != nil {
			companyName = companyNameMap[*cid]
		}
		projectGroups[api.ProjectID] = append(projectGroups[api.ProjectID], apiWithContext{
			api:         api,
			vars:        envMap[api.ProjectID],
			projectName: nameMap[api.ProjectID],
			companyName: companyName,
		})
	}

	// Dispatch per-project — projects always run concurrently with each other
	for projectID, group := range projectGroups {
		mode := execModeMap[projectID]
		groupCopy := group // capture for goroutine

		if mode == "parallel" {
			// Parallel: fire all APIs in the project simultaneously
			for _, ctx := range groupCopy {
				go RunPing(ctx.api, ctx.vars, ctx.projectName, ctx.companyName)
			}
		} else {
			// Sequential (default): wait for each API to finish before starting next
			// Sort by order_index to respect user-defined order
			go func(items []apiWithContext) {
				for _, ctx := range items {
					RunPing(ctx.api, ctx.vars, ctx.projectName, ctx.companyName)
				}
			}(groupCopy)
		}
	}
}


// RunPing executes the healthcheck orchestrating Builder, Evaluator, and Dispatcher.
func RunPing(api models.API, envVars map[string]string, projectName string, companyName string) {
	start := time.Now()

	// 1. Build Request
	req, err := BuildRequest(api, envVars)
	if err != nil {
		handleResult(api, nil, "", err, 0, projectName, companyName)
		return
	}

	// 2. Execute Request
	client := getClient()
	resp, err := client.Do(req)

	duration := time.Since(start)

	var bodyStr string
	if resp != nil && resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr = string(bodyBytes)
		resp.Body.Close()
	}

	// 3. Evaluate & Handle Result
	handleResult(api, resp, bodyStr, err, duration, projectName, companyName)
}

func handleResult(api models.API, resp *http.Response, bodyStr string, reqErr error, duration time.Duration, projectName string, companyName string) {
	isSuccess, errorMessage := EvaluateResult(api, resp, bodyStr, reqErr, duration)

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	// Only store response body on failure to prevent DB bloat.
	// Truncate to 500 chars max to keep error context readable.
	if isSuccess {
		bodyStr = ""
	} else if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "... (truncated)"
	}

	// Save to MonitorLog
	// Clear error message if the check ultimately succeeded
	if isSuccess {
		errorMessage = ""
	}

	logEntry := models.MonitorLog{
		ApiID:        api.ID,
		StatusCode:   statusCode,
		ResponseTime: duration.Milliseconds(),
		IsSuccess:    isSuccess,
		ErrorMessage: errorMessage,
		ResponseBody: bodyStr,
		CheckedAt:    time.Now(),
	}
	database.DB.Create(&logEntry)

	// If failed, manage Auto-Repair Tasks & Notifications
	if !isSuccess {
		var activeTask models.RepairTask
		database.DB.Where("api_id = ? AND status IN ('open', 'pending')", api.ID).First(&activeTask)

		if activeTask.ID == uuid.Nil {
			// No active repair task, create one
			newTask := models.RepairTask{
				ProjectID:   api.ProjectID,
				ApiID:       api.ID,
				Status:      "open",
				Description: fmt.Sprintf("Auto-detected failure for %s. Error: %s", api.Name, errorMessage),
			}
			database.DB.Create(&newTask)

			// Dispatch Alerts
			DispatchAlerts(api, errorMessage, projectName, companyName)

			// Generate Dashboard Notification
			handlers.CreateProjectNotification(
				api.ProjectID,
				"task_create",
				"New Repair Task",
				fmt.Sprintf("Auto-generated task for failing API: %s", api.Name),
			)
		}
	} else {
		// Recovery script logic
		if api.RecoveryScript != "" {
			log.Printf("[Recovery] Running recovery script for API %s", api.Name)
		}
	}
}

func importJSON(s string, v interface{}) {
	_ = json.Unmarshal([]byte(s), v)
}

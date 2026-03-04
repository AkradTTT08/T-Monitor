package workers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

// StartHealthCheckWorker starts the background process for pinging APIs
func StartHealthCheckWorker() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds for APIs due
	go func() {
		for range ticker.C {
			checkAPIs()
		}
	}()
}

func replaceEnvVariables(input string, envVars map[string]string) string {
	if input == "" {
		return ""
	}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return re.ReplaceAllStringFunc(input, func(m string) string {
		key := m[2 : len(m)-2] // strip {{ and }}
		if val, ok := envVars[key]; ok {
			return val
		}
		return m
	})
}

var lastCheckMap = make(map[uint]time.Time)

func checkAPIs() {
	var apis []models.API
	database.DB.Where("is_active = ?", true).Find(&apis)

	// Fetch all projects to get their environment variables
	var projects []models.Project
	database.DB.Find(&projects)
	envMap := make(map[uint]map[string]string)
	for _, p := range projects {
		var vars map[string]string
		if p.EnvironmentVariables != "" && p.EnvironmentVariables != "{}" {
			json.Unmarshal([]byte(p.EnvironmentVariables), &vars)
		}
		envMap[p.ID] = vars
	}

	now := time.Now()
	for _, api := range apis {
		// Check if this specific API is due based on its interval
		if lastCheck, exists := lastCheckMap[api.ID]; exists {
			// api.Interval is in seconds
			if time.Since(lastCheck).Seconds() < float64(api.Interval) {
				continue // not due yet
			}
		}

		// Update last check time before running
		lastCheckMap[api.ID] = now

		vars := envMap[api.ProjectID]
		go runPing(api, vars)
	}
}

func runPing(api models.API, envVars map[string]string) {
	start := time.Now()

	// Replace variables
	if len(envVars) > 0 {
		api.URL = replaceEnvVariables(api.URL, envVars)
		api.Headers = replaceEnvVariables(api.Headers, envVars)
		api.Body = replaceEnvVariables(api.Body, envVars)
		api.Parameters = replaceEnvVariables(api.Parameters, envVars)
	}

	// Process URL Parameters
	if api.Parameters != "" && api.Parameters != "[]" && api.Parameters != "{}" && api.Parameters != "{\n}" {
		u, err := url.Parse(api.URL)
		if err == nil {
			q := u.Query()
			// Try to parse as array of objects [{"key": "foo", "value": "bar"}]
			var paramsArray []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.Unmarshal([]byte(api.Parameters), &paramsArray); err == nil {
				for _, p := range paramsArray {
					if p.Key != "" {
						q.Add(strings.TrimSpace(p.Key), p.Value)
					}
				}
			} else {
				// Try to parse as map {"foo": "bar"}
				var paramsMap map[string]interface{}
				if err := json.Unmarshal([]byte(api.Parameters), &paramsMap); err == nil {
					for k, v := range paramsMap {
						if k != "" {
							q.Add(strings.TrimSpace(k), fmt.Sprintf("%v", v))
						}
					}
				}
			}
			u.RawQuery = q.Encode()
			api.URL = u.String()
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var req *http.Request
	var err error

	if api.Body != "" {
		req, err = http.NewRequest(api.Method, api.URL, bytes.NewBuffer([]byte(api.Body)))
	} else {
		req, err = http.NewRequest(api.Method, api.URL, nil)
	}

	if err != nil {
		handleResult(api, 0, 0, false, err.Error(), "")
		return
	}

	// Parse Headers
	if api.Headers != "" && api.Headers != "[]" && api.Headers != "{}" && api.Headers != "{\n}" {
		// Try to parse as array of objects [{"key": "foo", "value": "bar"}]
		var headersArray []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(api.Headers), &headersArray); err == nil {
			for _, h := range headersArray {
				if h.Key != "" {
					req.Header.Add(strings.TrimSpace(h.Key), h.Value)
				}
			}
		} else {
			// Try to parse as map {"foo": "bar"}
			var headersMap map[string]interface{}
			if err := json.Unmarshal([]byte(api.Headers), &headersMap); err == nil {
				for k, v := range headersMap {
					if k != "" {
						req.Header.Add(strings.TrimSpace(k), fmt.Sprintf("%v", v))
					}
				}
			}
		}
	}

	res, err := client.Do(req)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		handleResult(api, 0, duration, false, err.Error(), "")
		return
	}
	defer res.Body.Close()

	bodyBytes, readerr := io.ReadAll(res.Body)
	bodyString := string(bodyBytes)

	log.Printf("[HealthCheck DEBUG] API ID: %d, Read Err: %v, Body Length: %d, Parsed Body: %s", api.ID, readerr, len(bodyBytes), bodyString)

	if res.StatusCode != api.ExpectedStatusCode {
		errMsg := fmt.Sprintf("Expected %d, got %d. Body: %s", api.ExpectedStatusCode, res.StatusCode, bodyString)
		handleResult(api, res.StatusCode, duration, false, errMsg, bodyString)
	} else {
		handleResult(api, res.StatusCode, duration, true, "", bodyString)
	}
}

func handleResult(api models.API, statusCode int, duration int64, isSuccess bool, errorMsg string, responseBody string) {
	logEntry := models.MonitorLog{
		ApiID:        api.ID,
		StatusCode:   statusCode,
		ResponseTime: duration,
		IsSuccess:    isSuccess,
		ErrorMessage: errorMsg,
		ResponseBody: responseBody,
		CheckedAt:    time.Now(),
	}

	database.DB.Create(&logEntry)

	// If failed, trigger webhook to n8n
	if !isSuccess {
		var config models.NotificationConfig
		err := database.DB.Where("project_id = ?", api.ProjectID).First(&config).Error

		// If config is genuinely missing, initialize an empty default to suppress error logs
		if err != nil {
			config = models.NotificationConfig{
				ProjectID: api.ProjectID,
			}
			database.DB.Create(&config)
		}

		triggerN8nWebhook(api, logEntry, &config)
	}
}

func triggerN8nWebhook(api models.API, entry models.MonitorLog, config *models.NotificationConfig) {
	webhookURL := os.Getenv("N8N_WEBHOOK_URL")
	if webhookURL == "" {
		log.Println("N8N_WEBHOOK_URL is not configured")
		return
	}

	payload := map[string]interface{}{
		"api_id":        api.ID,
		"project_id":    api.ProjectID,
		"api_name":      api.Name,
		"url":           api.URL,
		"status_code":   entry.StatusCode,
		"error_message": entry.ErrorMessage,
		"config":        config,
	}

	jsonPayload, _ := json.Marshal(payload)

	_, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("Error triggering n8n webhook: %v\n", err)
	}
}

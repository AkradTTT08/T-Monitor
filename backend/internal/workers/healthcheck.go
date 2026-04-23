package workers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
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

var lastCheckMap = make(map[uuid.UUID]time.Time)

func checkAPIs() {
	var apis []models.API
	// GORM soft-delete: deleted_at IS NULL is auto-applied, so deleted APIs won't appear
	database.DB.Where("is_active = ?", true).Find(&apis)

	// Clean up lastCheckMap for APIs that no longer exist (soft-deleted)
	activeIDs := make(map[uuid.UUID]bool)
	for _, api := range apis {
		activeIDs[api.ID] = true
	}
	for id := range lastCheckMap {
		if !activeIDs[id] {
			delete(lastCheckMap, id)
		}
	}

	// Fetch all projects to get their environment variables
	var projects []models.Project
	database.DB.Find(&projects)
	envMap := make(map[uuid.UUID]map[string]string)
	nameMap := make(map[uuid.UUID]string)
	for _, p := range projects {
		var vars map[string]string
		if p.EnvironmentVariables != "" && p.EnvironmentVariables != "{}" {
			json.Unmarshal([]byte(p.EnvironmentVariables), &vars)
		}
		envMap[p.ID] = vars
		nameMap[p.ID] = p.Name
	}

	now := time.Now()
	for _, api := range apis {
		// Skip if explicitly paused
		if api.PausedUntil != nil && api.PausedUntil.After(now) {
			continue
		}

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
		projectName := nameMap[api.ProjectID]
		go runPing(api, vars, projectName)
	}
}

func runPing(api models.API, envVars map[string]string, projectName string) {
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
		handleResult(api, 0, 0, false, err.Error(), "", projectName, "", "{}")
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

	// ---- SELF HEALING AUTO-RETRY LOGIC ----
	isFailed := err != nil || (res != nil && res.StatusCode != api.ExpectedStatusCode)
	if isFailed && api.RecoveryScript != "" {
		errMsg := "Unknown"
		if err != nil { errMsg = err.Error() } else if res != nil { errMsg = fmt.Sprintf("Expected %d, got %d", api.ExpectedStatusCode, res.StatusCode) }
		
		log.Printf("[Self-Healing] Attempting recovery for API %s: %v", api.Name, errMsg)
		if res != nil { res.Body.Close() }
		
		executeRecoveryScript(api, errMsg)
		time.Sleep(5 * time.Second)
		
		start = time.Now()
		// Re-prepare request for retry
		if api.Body != "" {
			req, _ = http.NewRequest(api.Method, api.URL, bytes.NewBuffer([]byte(api.Body)))
		} else {
			req, _ = http.NewRequest(api.Method, api.URL, nil)
		}
		if req != nil {
			req.Header = req.Header.Clone() // Try to keep headers simple or just let them be omitted for retry if logic is too complex to copy, wait no, original req.Header was populated.
			// Let's just re-parse headers for safety
			if api.Headers != "" && api.Headers != "[]" && api.Headers != "{}" && api.Headers != "{\n}" {
				var headersArray []struct { Key string `json:"key"`; Value string `json:"value"` }
				if err := json.Unmarshal([]byte(api.Headers), &headersArray); err == nil {
					for _, h := range headersArray { if h.Key != "" { req.Header.Add(strings.TrimSpace(h.Key), h.Value) } }
				} else {
					var headersMap map[string]interface{}
					if err := json.Unmarshal([]byte(api.Headers), &headersMap); err == nil {
						for k, v := range headersMap { if k != "" { req.Header.Add(strings.TrimSpace(k), fmt.Sprintf("%v", v)) } }
					}
				}
			}
			res, err = client.Do(req)
			duration = time.Since(start).Milliseconds()
		}
	}
	// ---------------------------------------

	if err != nil {
		handleResult(api, 0, duration, false, err.Error(), "", projectName, "", "{}")
		return
	}
	defer res.Body.Close()

	var tlsStatusStr string
	if res.TLS != nil && len(res.TLS.PeerCertificates) > 0 {
		cert := res.TLS.PeerCertificates[0]
		tlsData := map[string]interface{}{
			"valid":      time.Now().Before(cert.NotAfter),
			"expires_at": cert.NotAfter,
			"issuer":     cert.Issuer.CommonName,
		}
		if b, e := json.Marshal(tlsData); e == nil {
			tlsStatusStr = string(b)
		}
	}

	securityHeadersStr := "{}"
	secHeaders := map[string]string{}
	headersToCheck := []string{"Strict-Transport-Security", "Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options"}
	for _, h := range headersToCheck {
		if val := res.Header.Get(h); val != "" {
			secHeaders[h] = "Present"
		} else {
			secHeaders[h] = "Missing"
		}
	}
	if b, e := json.Marshal(secHeaders); e == nil {
		securityHeadersStr = string(b)
	}

	bodyBytes, readerr := io.ReadAll(res.Body)
	bodyString := string(bodyBytes)

	log.Printf("[HealthCheck DEBUG] API ID: %s, Read Err: %v, Body Length: %d, Parsed Body: %s", api.ID, readerr, len(bodyBytes), bodyString)

	// Anomaly Detection: Compare latency with last 5 successful runs
	var recentLogs []models.MonitorLog
	database.DB.Where("api_id = ? AND is_success = true", api.ID).
		Order("checked_at DESC").Limit(5).Find(&recentLogs)
	
	if len(recentLogs) >= 3 {
		var totalDuration int64
		for _, l := range recentLogs {
			totalDuration += l.ResponseTime
		}
		avgDuration := totalDuration / int64(len(recentLogs))
		
		if avgDuration > 0 && duration > (avgDuration * 3) && duration > 500 {
			// Fetch Project Owner to send Dashboard Notification
			var project models.Project
			if err := database.DB.Select("user_id").First(&project, "id = ?", api.ProjectID).Error; err == nil {
				anomalyNotif := models.DashboardNotification{
					UserID:    project.UserID,
					ProjectID: api.ProjectID,
					Type:      "api_fail",
					Title:     "⚠️ Performance Anomaly: " + api.Name,
					Message:   fmt.Sprintf("API response time jumped to %dms (Average is %dms).", duration, avgDuration),
				}
				database.DB.Create(&anomalyNotif)
			}
		}
	}

	if res.StatusCode != api.ExpectedStatusCode {
		errMsg := fmt.Sprintf("Expected %d, got %d. Body: %s", api.ExpectedStatusCode, res.StatusCode, bodyString)
		handleResult(api, res.StatusCode, duration, false, errMsg, bodyString, projectName, tlsStatusStr, securityHeadersStr)
	} else {
		// Success! Execute extraction script if present
		if api.ResponseScript != "" {
			executeResponseScript(api, bodyString, res.StatusCode, res.Header)
		}
		handleResult(api, res.StatusCode, duration, true, "", bodyString, projectName, tlsStatusStr, securityHeadersStr)
	}
}

func executeRecoveryScript(api models.API, errorReason string) {
	vm := goja.New()

	vm.Set("errorReason", errorReason)

	// Set setEnv function for modifying env (e.g. storing new token)
	vm.Set("setEnv", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 { return goja.Undefined() }
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		updateProjectEnv(api.ProjectID, key, value)
		return goja.Undefined()
	})

	_, err := vm.RunString(api.RecoveryScript)
	if err != nil {
		log.Printf("[Recovery Script Error] API %s: %v", api.Name, err)
	}
}

func executeResponseScript(api models.API, responseBody string, statusCode int, headers http.Header) {
	vm := goja.New()

	// 1. Set response object
	respObj := vm.NewObject()
	respObj.Set("body", responseBody)
	respObj.Set("status", statusCode)
	
	// Convert headers to a simple map[string]string for the script
	hdrMap := make(map[string]string)
	for k, v := range headers {
		if len(v) > 0 {
			hdrMap[k] = v[0]
		}
	}
	respObj.Set("headers", hdrMap)
	vm.Set("response", respObj)

	// 2. Set setEnv function
	vm.Set("setEnv", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		updateProjectEnv(api.ProjectID, key, value)
		return goja.Undefined()
	})

	// 3. Set pm object for Postman familiarity
	pm := vm.NewObject()
	env := vm.NewObject()
	env.Set("set", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		updateProjectEnv(api.ProjectID, key, value)
		return goja.Undefined()
	})
	pm.Set("environment", env)
	vm.Set("pm", pm)

	// Execute with local timeout handling could be added here
	// For now, execute directly
	_, err := vm.RunString(api.ResponseScript)
	if err != nil {
		log.Printf("[Script Error] API %s (%s): %v", api.Name, api.ID, err)
	}
}

func updateProjectEnv(projectID uuid.UUID, key string, value string) {
	var project models.Project
	if err := database.DB.First(&project, "id = ?", projectID).Error; err != nil {
		return
	}

	var envVars map[string]string
	if project.EnvironmentVariables != "" && project.EnvironmentVariables != "{}" {
		json.Unmarshal([]byte(project.EnvironmentVariables), &envVars)
	} else {
		envVars = make(map[string]string)
	}

	// Update or Add
	envVars[key] = value
	envBytes, _ := json.Marshal(envVars)
	
	database.DB.Model(&models.Project{}).Where("id = ?", projectID).Update("environment_variables", string(envBytes))
}

func handleResult(api models.API, statusCode int, duration int64, isSuccess bool, errorMsg string, responseBody string, projectName string, tlsStatus string, secHeaders string) {
	logEntry := models.MonitorLog{
		ApiID:           api.ID,
		StatusCode:      statusCode,
		ResponseTime:    duration,
		IsSuccess:       isSuccess,
		ErrorMessage:    errorMsg,
		ResponseBody:    responseBody,
		Schedule:        formatScheduleString(api),
		TlsStatus:       tlsStatus,
		SecurityHeaders: secHeaders,
		CheckedAt:       time.Now(),
	}

	database.DB.Create(&logEntry)

	// If failed, send notifications
	if !isSuccess {
		var config models.NotificationConfig
		// Use Last() to get the most recently updated config
		err := database.DB.Where("project_id = ?", api.ProjectID).Last(&config).Error

		// If no config is set for this project, skip notification
		if err != nil {
			log.Printf("[Notify] No notification config found for project %d, skipping", api.ProjectID)
			return
		}

		// Send Telegram notification directly
		if config.EnableTelegram && config.TelegramBotToken != "" && config.TelegramChatID != "" {
			go sendTelegramMessage(api, logEntry, &config, projectName)
		}

		// Send Email notification directly
		if config.EnableEmail && config.EmailAddress != "" {
			go sendEmailNotification(api, logEntry, &config, projectName)
		}

		// Send Webhook notification directly
		if config.EnableWebhook && config.WebhookURL != "" {
			go sendWebhookNotification(api, logEntry, &config, projectName)
		}

		// Send LINE notification directly
		if config.EnableLINE && config.LINEUserID != "" {
			go sendLineNotification(api, logEntry, &config, projectName)
		}

		// INTELLIGENT REPAIR TASK CREATION (Rule 1, 2, 3)
		// Rule 3: Check if the previous check was a success (A new incident after recovery)
		var lastLog models.MonitorLog
		err = database.DB.Where("api_id = ?", api.ID).Order("checked_at desc").Offset(1).First(&lastLog).Error

		shouldCreateNew := false
		if err == nil && lastLog.IsSuccess {
			// Previous check was success, so this is a new incident
			shouldCreateNew = true
		} else if err != nil {
			// No previous logs, first failure
			shouldCreateNew = true
		}

		if !shouldCreateNew {
			// Rule 1 & 2: Check for identical existing tasks in the current failure streak
			var existingTask models.RepairTask
			err = database.DB.Where("api_id = ? AND status IN ? AND error_message = ? AND schedule = ?",
				api.ID, []string{"open", "pending"}, errorMsg, logEntry.Schedule).First(&existingTask).Error

			if err != nil {
				// No matching task found for this error type/schedule in the current streak
				shouldCreateNew = true
			}
		}

		if shouldCreateNew {
			// Create a new RepairTask
			newTask := models.RepairTask{
				ProjectID:    api.ProjectID,
				ApiID:        api.ID,
				Status:       "open",
				ErrorMessage: errorMsg,
				Schedule:     logEntry.Schedule,
			}
			database.DB.Create(&newTask)

			// Create Dashboard Notification
			handlers.CreateProjectNotification(
				api.ProjectID,
				"api_fail",
				"API Failure Detected",
				"API '" + api.Name + "' has failed: " + errorMsg,
			)
		}
	}
}

func formatScheduleString(api models.API) string {
	if api.ScheduleConfig == "" || api.ScheduleConfig == "{}" || api.ScheduleConfig == "{\n}" {
		// Fallback to interval based
		if api.Interval < 60 {
			return fmt.Sprintf("EVERY %d SEC", api.Interval)
		} else if api.Interval < 3600 {
			return fmt.Sprintf("EVERY %d MIN", api.Interval/60)
		} else {
			return fmt.Sprintf("EVERY %d HR", api.Interval/3600)
		}
	}

	var config struct {
		Mode  string `json:"mode"`
		Value int    `json:"value"`
		Day   string `json:"day"`
		Time  string `json:"time"`
	}

	if err := json.Unmarshal([]byte(api.ScheduleConfig), &config); err != nil {
		if api.Interval < 60 {
			return fmt.Sprintf("EVERY %d SEC", api.Interval)
		} else if api.Interval < 3600 {
			return fmt.Sprintf("EVERY %d MIN", api.Interval/60)
		} else {
			return fmt.Sprintf("EVERY %d HR", api.Interval/3600)
		}
	}

	switch config.Mode {
	case "Minute timer":
		return fmt.Sprintf("EVERY %d MIN", config.Value)
	case "Hour timer":
		return fmt.Sprintf("EVERY %d HR", config.Value)
	case "Week timer":
		day := strings.ToUpper(config.Day)
		return fmt.Sprintf("%s AT %s", day, config.Time)
	default:
		if api.Interval < 60 {
			return fmt.Sprintf("EVERY %d SEC", api.Interval)
		} else if api.Interval < 3600 {
			return fmt.Sprintf("EVERY %d MIN", api.Interval/60)
		} else {
			return fmt.Sprintf("EVERY %d HR", api.Interval/3600)
		}
	}
}

func sendTelegramMessage(api models.API, entry models.MonitorLog, config *models.NotificationConfig, projectName string) {
	message := fmt.Sprintf(
		"🚨 *API Alert - Project %s*\n\n"+
			"*API:* %s\n"+
			"*URL:* `%s`\n"+
			"*Status Code:* `%d`\n"+
			"*Error:* %s\n"+
			"*Time:* %s",
		projectName,
		api.Name,
		api.URL,
		entry.StatusCode,
		entry.ErrorMessage,
		entry.CheckedAt.Format("2006-01-02 15:04:05"),
	)

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramBotToken)

	dashboardURL := fmt.Sprintf("http://localhost:5173/dashboard/projects/%s", api.ProjectID.String())
	payload := map[string]interface{}{
		"chat_id":    config.TelegramChatID,
		"text":       message,
		"parse_mode": "Markdown",
		"reply_markup": map[string]interface{}{
			"inline_keyboard": [][]map[string]interface{}{
				{
					{"text": "🔇 Mute Alert (1h)", "callback_data": "mute_1h_" + api.ID.String()},
					{"text": "🛑 Pause Checker", "callback_data": "pause_inf_" + api.ID.String()},
				},
				{
					{"text": "🛠️ View Error in Dashboard", "url": dashboardURL},
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(telegramURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("[Notify] Failed to send Telegram message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Printf("[Notify] ✅ Telegram alert sent for API '%s' (project %d)", api.Name, api.ProjectID)
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Notify] ❌ Telegram API error %d: %s", resp.StatusCode, string(body))
	}
}

func sendEmailNotification(api models.API, entry models.MonitorLog, config *models.NotificationConfig, projectName string) {
	if config.SmtpHost == "" || config.SmtpUser == "" || config.SmtpPass == "" {
		log.Printf("[Notify] SMTP settings missing for project %d, skipping email", api.ProjectID)
		return
	}

	// Split multiple emails
	recipients := strings.Split(config.EmailAddress, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	subject := fmt.Sprintf("Subject: 🚨 API Alert: %s is DOWN\n", api.Name)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	// HTML Template Data
	data := struct {
		ApiName     string
		Url         string
		StatusCode  int
		ErrorMsg    string
		Time        string
		ProjectId   uuid.UUID
		ProjectName string
	}{
		ApiName:    api.Name,
		Url:        api.URL,
		StatusCode: entry.StatusCode,
		ErrorMsg:   entry.ErrorMessage,
		Time:       entry.CheckedAt.Format("2006-01-02 15:04:05"),
		ProjectId:  api.ProjectID,
		ProjectName: projectName,
	}

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		log.Printf("[Notify] Error parsing email template: %v", err)
		return
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		log.Printf("[Notify] Error executing email template: %v", err)
		return
	}

	msg := []byte(subject + mime + body.String())
	auth := smtp.PlainAuth("", config.SmtpUser, config.SmtpPass, config.SmtpHost)

	addr := fmt.Sprintf("%s:%d", config.SmtpHost, config.SmtpPort)
	err = smtp.SendMail(addr, auth, config.SmtpUser, recipients, msg)

	if err != nil {
		log.Printf("[Notify] ❌ Failed to send email alert: %v", err)
	} else {
		log.Printf("[Notify] ✅ Email alert sent to %d recipients for API '%s'", len(recipients), api.Name)
	}
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #0f172a; color: #f8fafc; margin: 0; padding: 0; }
        .container { max-width: 600px; margin: 40px auto; background-color: #1e293b; border-radius: 16px; border: 1px solid #334155; overflow: hidden; box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.4); }
        .header { background: linear-gradient(135deg, #ef4444 0%, #7f1d1d 100%); padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; letter-spacing: 2px; text-transform: uppercase; color: #fff; }
        .content { padding: 40px; }
        .alert-box { background-color: #0f172a; border-left: 4px solid #ef4444; padding: 20px; border-radius: 8px; margin-bottom: 30px; }
        .label { color: #94a3b8; font-size: 12px; font-weight: bold; text-transform: uppercase; margin-bottom: 4px; }
        .value { color: #f1f5f9; font-size: 16px; margin-bottom: 16px; font-family: 'Courier New', Courier, monospace; }
        .footer { padding: 20px; text-align: center; font-size: 12px; color: #64748b; border-top: 1px solid #334155; }
        .btn { display: inline-block; padding: 12px 24px; background-color: #ef4444; color: #fff; text-decoration: none; border-radius: 8px; font-weight: bold; margin-top: 20px; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚨 API CRITICAL ALERT</h1>
        </div>
        <div class="content">
            <p style="font-size: 16px; color: #cbd5e1; margin-top: 0;">An endpoint health check has failed. Action may be required.</p>
            
            <div class="alert-box">
                <div class="label">API NAME</div>
                <div class="value">{{.ApiName}}</div>
                
                <div class="label">ENDPOINT URL</div>
                <div class="value">{{.Url}}</div>
                
                <div class="label">STATUS CODE</div>
                <div class="value" style="color: #ef4444; font-weight: bold;">{{.StatusCode}}</div>
                
                <div class="label">ERROR MESSAGE</div>
                <div class="value">{{.ErrorMsg}}</div>
                
                <div class="label">CHECKED AT</div>
                <div class="value">{{.Time}}</div>
            </div>
            
            <div style="text-align: center;">
                <a href="http://localhost:5173/dashboard/projects/{{.ProjectId}}" class="btn">VIEW IN DASHBOARD</a>
            </div>
        </div>
        <div class="footer">
            <div style="margin-bottom: 6px;">© 2024 TTT BROTHER CO., LTD.</div>
            <div style="font-size: 11px;">
                Facebook: <a href="https://www.facebook.com/TTTBrother/" style="color: #64748b; text-decoration: none;">facebook.com/TTTBrother</a> | 
                Website: <a href="https://tttbrother.com/" style="color: #64748b; text-decoration: none;">tttbrother.com</a><br>
                Tell: 085 818 8910
            </div>
        </div>
    </div>
</body>
</html>
`

// kept for reference; no longer used after switching to native Telegram
func triggerN8nWebhook_legacy(api models.API, entry models.MonitorLog, config *models.NotificationConfig) {
	webhookURL := os.Getenv("N8N_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}
	payload := map[string]interface{}{
		"api_id": api.ID, "api_name": api.Name,
		"status_code": entry.StatusCode, "error_message": entry.ErrorMessage,
		"config": config,
	}
	jsonPayload, _ := json.Marshal(payload)
	http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
}

func sendWebhookNotification(api models.API, entry models.MonitorLog, config *models.NotificationConfig, projectName string) {
	payload := map[string]interface{}{
		"event":        "api_failure",
		"project_id":   api.ProjectID,
		"project_name": projectName,
		"api_id":       api.ID,
		"api_name":     api.Name,
		"url":          api.URL,
		"status_code":  entry.StatusCode,
		"error":        entry.ErrorMessage,
		"timestamp":    entry.CheckedAt.Format(time.RFC3339),
		"dashboard_url": fmt.Sprintf("http://localhost:5173/dashboard/projects/%s", api.ProjectID.String()),
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("[Webhook] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "T-Monitor/1.0")

	// Add HMAC signature if secret is provided
	if config.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(config.WebhookSecret))
		mac.Write(jsonPayload)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-TMonitor-Signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Webhook] Failed to send: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[Webhook] ✅ Alert sent to %s", config.WebhookURL)
	} else {
		log.Printf("[Webhook] ❌ Target returned status %d", resp.StatusCode)
	}
}

func sendLineNotification(api models.API, entry models.MonitorLog, config *models.NotificationConfig, projectName string) {
	message := fmt.Sprintf(
		"\n🚨 API Alert: %s\nProject: %s\nURL: %s\nStatus: %d\nError: %s\nTime: %s",
		api.Name,
		projectName,
		api.URL,
		entry.StatusCode,
		entry.ErrorMessage,
		entry.CheckedAt.Format("15:04:05"),
	)

	apiURL := "https://notify-api.line.me/api/notify"
	data := url.Values{}
	data.Set("message", message)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+config.LINEUserID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Printf("[LINE] ✅ Alert sent for %s", api.Name)
	} else {
		log.Printf("[LINE] ❌ Failed to send alert: %d", resp.StatusCode)
	}
}

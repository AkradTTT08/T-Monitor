package workers

import (
	"bytes"
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
		Schedule:     formatScheduleString(api),
		CheckedAt:    time.Now(),
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
			go sendTelegramMessage(api, logEntry, &config)
		}

		// Send Email notification directly
		if config.EnableEmail && config.EmailAddress != "" {
			go sendEmailNotification(api, logEntry, &config)
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

func sendTelegramMessage(api models.API, entry models.MonitorLog, config *models.NotificationConfig) {
	message := fmt.Sprintf(
		"🚨 *API Alert - Project %d*\n\n"+
			"*API:* %s\n"+
			"*URL:* `%s`\n"+
			"*Status Code:* `%d`\n"+
			"*Error:* %s\n"+
			"*Time:* %s",
		api.ProjectID,
		api.Name,
		api.URL,
		entry.StatusCode,
		entry.ErrorMessage,
		entry.CheckedAt.Format("2006-01-02 15:04:05"),
	)

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramBotToken)

	payload := map[string]interface{}{
		"chat_id":    config.TelegramChatID,
		"text":       message,
		"parse_mode": "Markdown",
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

func sendEmailNotification(api models.API, entry models.MonitorLog, config *models.NotificationConfig) {
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
		ApiName    string
		Url        string
		StatusCode int
		ErrorMsg   string
		Time       string
		ProjectId  uint
	}{
		ApiName:    api.Name,
		Url:        api.URL,
		StatusCode: entry.StatusCode,
		ErrorMsg:   entry.ErrorMessage,
		Time:       entry.CheckedAt.Format("2006-01-02 15:04:05"),
		ProjectId:  api.ProjectID,
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

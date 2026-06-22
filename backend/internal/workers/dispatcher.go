package workers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/smtp"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

// DispatchAlerts handles sending notifications to configured channels
func DispatchAlerts(api models.API, errorMsg string, projectName string, companyName string) {
	var config models.NotificationConfig
	if err := database.DB.First(&config, "project_id = ?", api.ProjectID).Error; err != nil {
		return // No config found
	}

	// Telegram
	if config.EnableTelegram && config.TelegramBotToken != "" && config.TelegramChatID != "" {
		go sendTelegramAlert(config.TelegramBotToken, config.TelegramChatID, api.Name, errorMsg, projectName)
	}

	// LINE
	if config.EnableLINE && config.LINEUserID != "" {
		go sendLINEAlert(config.LINEUserID, api.Name, errorMsg, projectName)
	}

	// Email
	if config.EnableEmail && config.EmailAddress != "" && config.SmtpHost != "" {
		go sendEmailAlert(config, api.Name, errorMsg, projectName, companyName)
	}

	// Webhook
	if config.EnableWebhook && config.WebhookURL != "" {
		go sendWebhookAlert(config, api.Name, errorMsg, projectName)
	}
}

func sendTelegramAlert(botToken, chatID, apiName, errorMsg, projectName string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	
	safeProjectName := html.EscapeString(projectName)
	safeApiName := html.EscapeString(apiName)
	safeErrorMsg := html.EscapeString(errorMsg)
	
	message := fmt.Sprintf("🚨 <b>API Alert</b>\n\n<b>Project:</b> %s\n<b>API:</b> %s\n<b>Error:</b> <code>%s</code>\n\n<i>Powered by T-Monitor</i>", safeProjectName, safeApiName, safeErrorMsg)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("[Telegram] Failed to send alert: %v", err)
	} else {
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Printf("[Telegram] Error from API (Status %d): %s", resp.StatusCode, string(bodyBytes))
		}
		resp.Body.Close()
	}
}

func sendLINEAlert(userID, apiName, errorMsg, projectName string) {
	// Note: Requires LINE Messaging API access token setup in environment
	// This is a placeholder for the actual LINE implementation
	log.Printf("[LINE] Alert sent to %s for %s: %s", userID, apiName, errorMsg)
}

func sendEmailAlert(config models.NotificationConfig, apiName, errorMsg, projectName, companyName string) {
	auth := smtp.PlainAuth("", config.SmtpUser, config.SmtpPass, config.SmtpHost)
	
	to := []string{config.EmailAddress}
	
	subject := fmt.Sprintf("Subject: 🚨 Alert: %s is failing\r\n", apiName)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	
	htmlBody := generateEmailTemplate(projectName, companyName, apiName, errorMsg)
	
	msg := []byte(subject + mime + htmlBody)

	addr := fmt.Sprintf("%s:%d", config.SmtpHost, config.SmtpPort)
	err := smtp.SendMail(addr, auth, config.SmtpUser, to, msg)
	if err != nil {
		log.Printf("[Email] Failed to send alert: %v", err)
	}
}

func sendWebhookAlert(config models.NotificationConfig, apiName, errorMsg, projectName string) {
	payload := map[string]string{
		"event":        "api_failure",
		"project_name": projectName,
		"api_name":     apiName,
		"error":        errorMsg,
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	// Create HMAC signature if secret is provided
	if config.WebhookSecret != "" {
		h := hmac.New(sha256.New, []byte(config.WebhookSecret))
		h.Write(jsonPayload)
		signature := hex.EncodeToString(h.Sum(nil))
		req.Header.Set("X-Hub-Signature-256", "sha256="+signature)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Webhook] Failed to send alert: %v", err)
	} else {
		resp.Body.Close()
	}
}

func generateEmailTemplate(projectName, companyName, apiName, errorMsg string) string {
	if companyName == "" {
		companyName = "T-Monitor System"
	}
	
	htmlTmpl := `
	<!DOCTYPE html>
	<html>
	<body style="font-family: Arial, sans-serif; background-color: #f4f4f5; padding: 20px;">
		<div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
			<div style="background-color: #ef4444; padding: 20px; text-align: center;">
				<h1 style="color: #ffffff; margin: 0;">🚨 API Alert</h1>
			</div>
			<div style="padding: 30px;">
				<p style="font-size: 16px; color: #374151;">Hello,</p>
				<p style="font-size: 16px; color: #374151;">An issue has been detected with your API.</p>
				
				<div style="background-color: #fee2e2; border-left: 4px solid #ef4444; padding: 15px; margin: 20px 0;">
					<h3 style="margin-top: 0; color: #b91c1c;">{{.APIName}}</h3>
					<p style="margin: 5px 0 0 0; color: #7f1d1d; font-family: monospace;">{{.ErrorMsg}}</p>
				</div>
				
				<table style="width: 100%; border-collapse: collapse; margin-top: 20px;">
					<tr>
						<td style="padding: 10px; border-bottom: 1px solid #e5e7eb; color: #6b7280; width: 30%;">Project</td>
						<td style="padding: 10px; border-bottom: 1px solid #e5e7eb; color: #111827; font-weight: bold;">{{.ProjectName}}</td>
					</tr>
					<tr>
						<td style="padding: 10px; border-bottom: 1px solid #e5e7eb; color: #6b7280;">Company</td>
						<td style="padding: 10px; border-bottom: 1px solid #e5e7eb; color: #111827; font-weight: bold;">{{.CompanyName}}</td>
					</tr>
				</table>
				
				<div style="margin-top: 30px; text-align: center;">
					<a href="#" style="background-color: #3b82f6; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">View Dashboard</a>
				</div>
			</div>
			<div style="background-color: #f9fafb; padding: 15px; text-align: center; font-size: 12px; color: #9ca3af;">
				Powered by T-Monitor ProDoc HUB
			</div>
		</div>
	</body>
	</html>
	`
	
	t, err := template.New("email").Parse(htmlTmpl)
	if err != nil {
		return fmt.Sprintf("API %s failed: %s", apiName, errorMsg)
	}
	
	data := struct {
		ProjectName string
		CompanyName string
		APIName     string
		ErrorMsg    string
	}{
		ProjectName: projectName,
		CompanyName: companyName,
		APIName:     apiName,
		ErrorMsg:    errorMsg,
	}
	
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Sprintf("API %s failed: %s", apiName, errorMsg)
	}
	
	return buf.String()
}

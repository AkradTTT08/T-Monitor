package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

type ChatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type AIQueryRequest struct {
	Query   string        `json:"query"`
	History []ChatMessage `json:"history"`
}

type AIQueryResponse struct {
	Answer string `json:"answer"`
}

// getOllamaHost returns the Ollama base URL from env or default docker service
func getOllamaHost() string {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		// Default to localhost for easier local development on Windows
		host = "http://localhost:11434"
	}
	return strings.TrimRight(host, "/")
}

// getOllamaModel returns the model name to use
func getOllamaModel() string {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}
	return model
}

// ollamaGenerate calls Ollama /api/generate and returns the response text
func ollamaGenerate(prompt string) (string, error) {
	host := getOllamaHost()
	model := getOllamaModel()

	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", host+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("ollama error: %s", result.Error)
	}

	return strings.TrimSpace(result.Response), nil
}

// getDatabaseSchema returns a read-only representation of the database schema for the AI to understand.
// We only expose non-sensitive fields.
func getDatabaseSchema() string {
	return `
You are an expert SQL assistant for an API Monitoring platform.

Relevant Tables:
1. users (id, email, name, role)
2. projects (id, name, description, user_id, created_at, deleted_at)
3. project_members (id, project_id, user_id, role)
4. apis (id, project_id, name, method, url, is_active, created_at, deleted_at)
5. monitor_logs (id, api_id, status_code, is_success, checked_at)

CORE QUERY RULES:
- Return ONLY a valid PostgreSQL SELECT statement. No markdown, no backticks.
- IMPORTANT: When searching for project names, ignore prefixes like "Project" or "โปรเจกต์" unless it's part of the actual name. For "Project Allkon", search for name ILIKE '%Allkon%'.
- ALWAYS use ILIKE for string matches for maximum flexibility.
- ALWAYS include "deleted_at IS NULL" for tables that have it.
- To find APIs for a project, JOIN apis and projects on apis.project_id = projects.id.
- If asking "กี่เส้น" (how many), use COUNT(*).
- If asking "เส้นไหน" (which ones), SELECT apis.name, apis.url, apis.method.
- Limit results to 50 items.
- If the query is not data-related, return: SELECT 'general' as type;
`
}

func ChatWithAI(c *fiber.Ctx) error {
	var req AIQueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Build conversation context
	historyContext := ""
	if len(req.History) > 0 {
		historyContext = "Recent Conversation Context:\n"
		for _, msg := range req.History {
			historyContext += fmt.Sprintf("%s: %s\n", msg.Role, msg.Text)
		}
	}

	// 1. Ask Ollama to generate SQL
	sqlPrompt := fmt.Sprintf("%s\n\n%s\nUser Question: %s\n\nSQL Query:", getDatabaseSchema(), historyContext, req.Query)

	sqlQuery, err := ollamaGenerate(sqlPrompt)
	if err != nil {
		log.Printf("[AI] Failed to generate SQL from Ollama: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "AI service is unavailable. Please ensure Ollama is running.",
		})
	}

	// Clean any markdown that might slip through
	sqlQuery = strings.TrimSpace(sqlQuery)
	sqlQuery = strings.TrimPrefix(sqlQuery, "```sql")
	sqlQuery = strings.TrimPrefix(sqlQuery, "```postgres")
	sqlQuery = strings.TrimPrefix(sqlQuery, "```")
	sqlQuery = strings.TrimSuffix(sqlQuery, "```")
	sqlQuery = strings.TrimSpace(sqlQuery)

	log.Printf("[AI] Generated SQL: %s", sqlQuery)

	// 2. Execute SQL Query
	var results []map[string]interface{}
	dbRes := database.DB.Raw(sqlQuery).Scan(&results)
	if dbRes.Error != nil {
		log.Printf("[AI] Failed to execute SQL: %v", dbRes.Error)
		return c.JSON(fiber.Map{
			"answer": "ขออภัยครับ ไม่สามารถสร้าง Query ที่ถูกต้องได้ กรุณาลองถามใหม่อีกครั้งครับ",
		})
	}

	// 3. Convert results to JSON
	resultsJSON, _ := json.Marshal(results)

	// 4. Ask Ollama to summarize results in Thai
	summaryPrompt := fmt.Sprintf(`You are a helpful AI assistant for an API Monitoring platform.
A user asked: "%s"

I queried the database and got this JSON result:
%s

Please provide a concise, friendly, and complete answer to the user based ONLY on the JSON result above.
If the JSON result is empty, tell the user no data was found for their query.
Use Thai language.`, req.Query, string(resultsJSON))

	finalAnswer, err := ollamaGenerate(summaryPrompt)
	if err != nil {
		log.Printf("[AI] Failed to summarize with Ollama: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to summarize the data"})
	}

	return c.JSON(fiber.Map{"answer": finalAnswer})
}

// AnalyzeIncident uses Ollama to perform Root Cause Analysis on a failed API request
func AnalyzeIncident(c *fiber.Ctx) error {
	type AnalyzeRequest struct {
		LogID string `json:"log_id"`
	}

	var req AnalyzeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Fetch the log with its associated API
	var logEntry models.MonitorLog
	if err := database.DB.Preload("API").First(&logEntry, "id = ?", req.LogID).Error; err != nil {
		log.Printf("[AI] Log not found: %v", req.LogID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Monitor log not found"})
	}

	if logEntry.API == nil {
		log.Printf("[AI] API record missing for log: %v", req.LogID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Associated API record not found for this log"})
	}

	// If log is success, early return
	if logEntry.IsSuccess {
		return c.JSON(fiber.Map{"reason": "ระบบทำงานปกติ 200 OK ไม่พบข้อผิดพลาดที่ต้องวิเคราะห์ครับ"})
	}

	prompt := fmt.Sprintf(`You are an expert Backend & DevOps Engineer. I need your help to perform a Root Cause Analysis (RCA) on an API failure.
Here are the details of the failed request:

API Endpoint: [%s] %s
Expected Status Code: %d
Actual Status Code: %d
Error Message: %s
Response Body:
%s

Please analyze this failure. Explain what likely went wrong and suggest 1-2 actionable steps to fix it.
Format your response in Thai language, make it concise, easy to read, and polite.
Use Markdown lists for readability.`,
		logEntry.API.Method, logEntry.API.URL,
		logEntry.API.ExpectedStatusCode, logEntry.StatusCode,
		logEntry.ErrorMessage, logEntry.ResponseBody,
	)

	rcaAnswer, err := ollamaGenerate(prompt)
	if err != nil {
		log.Printf("[AI] Ollama Error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "AI service error: " + err.Error(),
		})
	}

	if rcaAnswer == "" {
		rcaAnswer = "AI ไม่สามารถวิเคราะห์สาเหตุได้ในขณะนี้ โปรดตรวจสอบ Error Message ด้วยตนเองครับ"
	}

	return c.JSON(fiber.Map{"reason": rcaAnswer})
}

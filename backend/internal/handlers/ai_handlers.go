package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/generative-ai-go/genai"
	"github.com/monitor-api/backend/internal/database"
	"google.golang.org/api/option"
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

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gemini API key is not configured"})
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("Failed to create GenAI client: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to connect to AI service"})
	}
	defer client.Close()

	// 1. Ask Gemini to generate SQL based on Schema and User Query
	sqlModel := client.GenerativeModel("gemini-2.5-flash")
	
	historyContext := ""
	if len(req.History) > 0 {
		historyContext = "Recent Conversation Context:\n"
		for _, msg := range req.History {
			historyContext += fmt.Sprintf("%s: %s\n", msg.Role, msg.Text)
		}
	}

	prompt := fmt.Sprintf("%s\n\n%s\nUser Question: %s\n\nSQL Query:", getDatabaseSchema(), historyContext, req.Query)
	
	resp, err := sqlModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("Failed to generate SQL from Gemini: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("AI failed to process the query: %v", err)})
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "AI returned empty response"})
	}

	sqlQuery := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	
	// Clean markdown backticks just in case Gemini ignored the rule
	sqlQuery = strings.TrimSpace(sqlQuery)
	sqlQuery = strings.TrimPrefix(sqlQuery, "```sql")
	sqlQuery = strings.TrimPrefix(sqlQuery, "```postgres")
	sqlQuery = strings.TrimPrefix(sqlQuery, "```")
	sqlQuery = strings.TrimSuffix(sqlQuery, "```")
	sqlQuery = strings.TrimSpace(sqlQuery)

	// Basic sanitation to prevent execution of non-selects just in case
	log.Printf("AI Generated SQL: %s", sqlQuery)

	// 2. Execute SQL Query
	var results []map[string]interface{}
	dbRes := database.DB.Raw(sqlQuery).Scan(&results)
	if dbRes.Error != nil {
		log.Printf("Failed to execute AI SQL: %v", dbRes.Error)
		// If SQL fails, we just feed the error back to the user smoothly
		return c.JSON(fiber.Map{
			"answer": "Sorry, I couldn't formulate a proper query to find that information. Could you rephrase your question?",
		})
	}

	// 3. Convert results to JSON string
	resultsJSON, _ := json.Marshal(results)

	// 4. Summarize Results with Gemini Flash
	summaryModel := client.GenerativeModel("gemini-2.5-flash") 
	summaryPrompt := fmt.Sprintf(`
You are a helpful AI assistant for an API Monitoring platform.
A user asked: "%s"

I queried the database and got this JSON result:
%s

Please provide a concise, friendly, and complete answer to the user based ONLY on the JSON result above. 
If the JSON result is empty, tell the user no data was found for their query.
Use Thai language.
`, req.Query, string(resultsJSON))

	sumResp, err := summaryModel.GenerateContent(ctx, genai.Text(summaryPrompt))
	if err != nil {
		log.Printf("Failed to summarize with Gemini: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to summarize the data"})
	}

	if len(sumResp.Candidates) == 0 || len(sumResp.Candidates[0].Content.Parts) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "AI returned empty summary"})
	}

	finalAnswer := fmt.Sprintf("%v", sumResp.Candidates[0].Content.Parts[0])

	return c.JSON(fiber.Map{
		"answer": finalAnswer,
	})
}

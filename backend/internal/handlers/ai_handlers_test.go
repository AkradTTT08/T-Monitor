package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func setupAIRouter() *testutils.FiberApp {
	db := testutils.SetupTestDB()
	app := testutils.SetupTestApp()

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		role := c.Get("Test-Role")
		if userID != "" {
			uid, _ := uuid.Parse(userID)
			c.Locals("user_id", uid)
			c.Locals("role", role)
		}
		return c.Next()
	})

	app.Get("/ai/analyze/:id", handlers.AnalyzeIncident)
	app.Post("/ai/query", handlers.QueryData)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestAnalyzeIncident_Mocked(t *testing.T) {
	env := setupAIRouter()

	user := models.User{Email: "ai_incident@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "AI Project", UserID: user.ID}
	database.DB.Create(&project)

	api := models.API{Name: "Fail API", ProjectID: project.ID, URL: "http://example.com/fail"}
	database.DB.Create(&api)

	log := models.MonitorLog{
		ApiID:        api.ID,
		IsSuccess:    false,
		ErrorMessage: "Connection refused",
		ResponseBody: "",
	}
	database.DB.Create(&log)

	req := httptest.NewRequest("GET", "/ai/analyze/"+log.ID.String(), nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")
	// Trigger the mock mode in handler
	req.Header.Set("Test-Mock-AI", "true")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "Mocked AI Response", result["analysis"])
}

func TestQueryData_Mocked(t *testing.T) {
	env := setupAIRouter()

	user := models.User{Email: "ai_query@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Query Project", UserID: user.ID}
	database.DB.Create(&project)

	body := map[string]interface{}{
		"query": "What is my uptime?",
		"history": []map[string]string{},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/ai/query?project_id="+project.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")
	req.Header.Set("Test-Mock-AI", "true")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "Mocked AI Response", result["answer"])
}

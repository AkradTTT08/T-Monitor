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

func setupRepairRouter() *testutils.FiberApp {
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

	app.Get("/projects/:id/repair-tasks", handlers.GetRepairTasks)
	app.Put("/repair-tasks/:id/approve", handlers.ApproveRepairTask)
	app.Put("/repair-tasks/:id/close", handlers.CloseRepairTask)
	app.Put("/repair-tasks/:id/fail", handlers.FailRepairTask)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestApproveRepairTask_Success(t *testing.T) {
	env := setupRepairRouter()

	user := models.User{Email: "repair_approver@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Repair Project", UserID: user.ID}
	database.DB.Create(&project)

	task := models.RepairTask{
		ProjectID:   project.ID,
		Status:      "open",
		Description: "API is down",
	}
	database.DB.Create(&task)

	req := httptest.NewRequest("PUT", "/repair-tasks/"+task.ID.String()+"/approve", nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var checkTask models.RepairTask
	database.DB.First(&checkTask, "id = ?", task.ID)
	assert.Equal(t, "pending", checkTask.Status)
	assert.Equal(t, user.ID, *checkTask.ApprovedBy)
	assert.NotNil(t, checkTask.ApprovedAt)
}

func TestCloseRepairTask_Success(t *testing.T) {
	env := setupRepairRouter()

	user := models.User{Email: "repair_closer@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Close Project", UserID: user.ID}
	database.DB.Create(&project)

	task := models.RepairTask{
		ProjectID:   project.ID,
		Status:      "pending",
		Description: "Fixing DB connection",
	}
	database.DB.Create(&task)

	body := map[string]interface{}{
		"reason":       "Database credentials expired",
		"fixer_name":   "Dev Team",
		"document_url": "http://jira.com/ticket-123",
		"documents":    []string{"Screenshot.png"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/repair-tasks/"+task.ID.String()+"/close", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var checkTask models.RepairTask
	database.DB.First(&checkTask, "id = ?", task.ID)
	assert.Equal(t, "closed", checkTask.Status)
	assert.Equal(t, "Database credentials expired", checkTask.Reason)
	assert.Equal(t, `["Screenshot.png"]`, checkTask.Documents)
	assert.NotNil(t, checkTask.ClosedAt)
}

func TestFailRepairTask_Success(t *testing.T) {
	env := setupRepairRouter()

	user := models.User{Email: "repair_failer@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Fail Project", UserID: user.ID}
	database.DB.Create(&project)

	task := models.RepairTask{
		ProjectID:   project.ID,
		Status:      "pending",
		Description: "Investigating 500 error",
	}
	database.DB.Create(&task)

	body := map[string]interface{}{
		"description": "Could not reproduce locally, moving to failed.",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/repair-tasks/"+task.ID.String()+"/fail", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var checkTask models.RepairTask
	database.DB.First(&checkTask, "id = ?", task.ID)
	assert.Equal(t, "failed", checkTask.Status)
	assert.Equal(t, "Could not reproduce locally, moving to failed.", checkTask.Description)
}

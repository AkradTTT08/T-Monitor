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

func setupProjectRouter() *testutils.FiberApp {
	db := testutils.SetupTestDB()
	app := testutils.SetupTestApp()

	// Mocking auth middleware
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

	app.Post("/projects", handlers.CreateProject)
	app.Get("/projects", handlers.GetProjects)
	app.Delete("/projects/:id", handlers.DeleteProject)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestCreateProject_Success(t *testing.T) {
	env := setupProjectRouter()

	user := models.User{Email: "projectowner@example.com"}
	database.DB.Create(&user)

	body := map[string]interface{}{
		"name":        "Test Project",
		"description": "A project for testing",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/projects", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var project models.Project
	database.DB.First(&project, "name = ?", "Test Project")
	assert.Equal(t, "A project for testing", project.Description)
	assert.Equal(t, user.ID, project.UserID)

	// Verify default notification config was created
	var config models.NotificationConfig
	database.DB.First(&config, "project_id = ?", project.ID)
	assert.Equal(t, project.ID, config.ProjectID)
}

func TestGetProjects_Success(t *testing.T) {
	env := setupProjectRouter()

	admin := models.User{Email: "admin@example.com"}
	database.DB.Create(&admin)

	user1 := models.User{Email: "user1@example.com"}
	database.DB.Create(&user1)

	user2 := models.User{Email: "user2@example.com"}
	database.DB.Create(&user2)

	// User1's project
	proj1 := models.Project{Name: "P1", UserID: user1.ID}
	database.DB.Create(&proj1)

	// User2's project
	proj2 := models.Project{Name: "P2", UserID: user2.ID}
	database.DB.Create(&proj2)

	// User2's project where User1 is a member
	proj3 := models.Project{Name: "P3", UserID: user2.ID}
	database.DB.Create(&proj3)
	database.DB.Create(&models.ProjectMember{ProjectID: proj3.ID, UserID: user1.ID, Role: "member"})

	// Admin test
	reqAdmin := httptest.NewRequest("GET", "/projects", nil)
	reqAdmin.Header.Set("Test-User-ID", admin.ID.String())
	reqAdmin.Header.Set("Test-Role", "admin")

	respAdmin, _ := env.App.Test(reqAdmin, -1)
	var adminResult []models.Project
	json.NewDecoder(respAdmin.Body).Decode(&adminResult)
	assert.GreaterOrEqual(t, len(adminResult), 3) // Admin sees all projects in the shared test DB

	// User1 test
	reqUser1 := httptest.NewRequest("GET", "/projects", nil)
	reqUser1.Header.Set("Test-User-ID", user1.ID.String())
	reqUser1.Header.Set("Test-Role", "user")

	respUser1, _ := env.App.Test(reqUser1, -1)
	var user1Result []models.Project
	json.NewDecoder(respUser1.Body).Decode(&user1Result)
	assert.Len(t, user1Result, 2) // User1 sees P1 (owner) and P3 (member)
}

func TestDeleteProject_Success(t *testing.T) {
	env := setupProjectRouter()

	user := models.User{Email: "deluser@example.com"}
	database.DB.Create(&user)

	proj := models.Project{Name: "Project to Delete", UserID: user.ID}
	database.DB.Create(&proj)

	api := models.API{Name: "Test API", ProjectID: proj.ID}
	database.DB.Create(&api)

	req := httptest.NewRequest("DELETE", "/projects/"+proj.ID.String(), nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify project is soft deleted
	var deletedProj models.Project
	err = database.DB.Unscoped().First(&deletedProj, "id = ?", proj.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deletedProj.DeletedAt.Time)

	// Verify API is soft deleted
	var deletedApi models.API
	err = database.DB.Unscoped().First(&deletedApi, "id = ?", api.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deletedApi.DeletedAt.Time)
}

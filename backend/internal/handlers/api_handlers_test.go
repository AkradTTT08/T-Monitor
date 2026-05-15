package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func setupAPIRouter() *testutils.FiberApp {
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

	app.Post("/apis", handlers.CreateAPI)
	app.Put("/apis/:id", handlers.UpdateAPI)
	app.Post("/apis/:id/pause", handlers.PauseAPI)
	app.Post("/apis/import/postman", handlers.UploadPostmanCollection)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestCreateAPI_Success(t *testing.T) {
	env := setupAPIRouter()

	user := models.User{Email: "apiowner@test.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "API Project", UserID: user.ID}
	database.DB.Create(&project)

	body := map[string]interface{}{
		"project_id":           project.ID.String(),
		"name":                 "My Test API",
		"method":               "GET",
		"url":                  "https://example.com",
		"expected_status_code": 200,
		"interval":             60,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/apis", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var api models.API
	database.DB.First(&api, "name = ?", "My Test API")
	assert.Equal(t, project.ID, api.ProjectID)
	assert.Equal(t, "GET", api.Method)
	assert.Equal(t, 200, api.ExpectedStatusCode)
}

func TestPauseAPI_Success(t *testing.T) {
	env := setupAPIRouter()

	user := models.User{Email: "pauser@test.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Pause Project", UserID: user.ID}
	database.DB.Create(&project)

	indefinite := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	api := models.API{Name: "To Pause", ProjectID: project.ID, PausedUntil: &indefinite}
	database.DB.Create(&api)

	body := map[string]interface{}{
		"duration_minutes": 60,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/apis/"+api.ID.String()+"/pause", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedAPI models.API
	database.DB.First(&updatedAPI, "id = ?", api.ID)
	
	// Should not be year 9999 anymore
	assert.NotEqual(t, 9999, updatedAPI.PausedUntil.Year())
}

func TestPostmanImport_Success(t *testing.T) {
	env := setupAPIRouter()

	user := models.User{Email: "importer@test.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Import Project", UserID: user.ID}
	database.DB.Create(&project)

	postmanJSON := `{
		"item": [
			{
				"name": "Folder 1",
				"item": [
					{
						"name": "Get User",
						"request": {
							"method": "GET",
							"url": {
								"raw": "https://api.test.com/users/1"
							}
						}
					}
				]
			}
		],
		"variable": [
			{
				"key": "base_url",
				"value": "https://api.test.com"
			}
		]
	}`

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("collection", "postman_collection.json")
	part.Write([]byte(postmanJSON))
	writer.Close()

	req := httptest.NewRequest("POST", "/apis/import/postman?project_id="+project.ID.String()+"&mode=append", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apis []models.API
	database.DB.Where("project_id = ?", project.ID).Find(&apis)
	assert.Len(t, apis, 1)
	assert.Equal(t, "Get User", apis[0].Name)
	assert.Equal(t, "Folder 1", apis[0].Folder)

	// Check environment variables updated on project
	var updatedProject models.Project
	database.DB.First(&updatedProject, "id = ?", project.ID)
	assert.Contains(t, updatedProject.EnvironmentVariables, "base_url")
}

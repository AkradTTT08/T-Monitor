package handlers_test

import (
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

func setupUserRouter() *testutils.FiberApp {
	db := testutils.SetupTestDB()
	app := testutils.SetupTestApp()

	// Mocking auth middleware via a custom middleware just for tests
	// It extracts the "Authorization" header and sets locals
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

	app.Get("/users/profile", handlers.GetProfile)
	
	return &testutils.FiberApp{App: app, DB: db}
}

func TestGetProfile_Success(t *testing.T) {
	env := setupUserRouter()

	// Seed user
	user := models.User{
		Email: "profile@example.com",
		Name:  "Profile User",
	}
	database.DB.Create(&user)

	req := httptest.NewRequest("GET", "/users/profile", nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.User
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "profile@example.com", result.Email)
	assert.Equal(t, "Profile User", result.Name)
}

func TestGetProfile_Unauthorized(t *testing.T) {
	env := setupUserRouter()

	req := httptest.NewRequest("GET", "/users/profile", nil)
	// No Test-User-ID header provided

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/services"
	"github.com/monitor-api/backend/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func setupAuthRouter() *testutils.FiberApp {
	db := testutils.SetupTestDB()
	app := testutils.SetupTestApp()

	// Register Routes
	app.Post("/auth/register", handlers.Register)
	app.Post("/auth/login", handlers.Login)

	return &testutils.FiberApp{App: app, DB: db}
}

// Wrapper for convenience
type TestEnv struct {
	App *testutils.FiberApp
}

func TestRegister_Success(t *testing.T) {
	env := setupAuthRouter()

	// Clear users to ensure this is the first user (who becomes admin)
	database.DB.Exec("DELETE FROM users")

	body := map[string]string{
		"email":      "test@example.com",
		"password":   "password123",
		"name":       "Test User",
		"phone":      "0123456789",
		"department": "IT",
		"position":   "Dev",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify it was saved in the DB
	var user models.User
	database.DB.First(&user, "email = ?", "test@example.com")
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "admin", user.Role) // The first user registered should be admin
	assert.True(t, user.IsApproved)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	env := setupAuthRouter()

	// Create existing user
	database.DB.Create(&models.User{Email: "exist@example.com", Password: "hashed", Name: "Existing"})

	body := map[string]string{
		"email":    "exist@example.com",
		"password": "password123",
		"name":     "Duplicate User",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestLogin_Success(t *testing.T) {
	env := setupAuthRouter()

	// Seed an approved user in the DB
	authSvc := services.NewAuthService()
	hashed, _ := authSvc.HashPassword("mypassword")
	database.DB.Create(&models.User{
		Email:      "login@example.com",
		Password:   hashed,
		Name:       "Login User",
		IsApproved: true,
	})

	// Now try to login
	loginBody := map[string]string{
		"email":    "login@example.com",
		"password": "mypassword",
	}
	lb, _ := json.Marshal(loginBody)
	reqLogin := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(lb))
	reqLogin.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(reqLogin, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	assert.NotEmpty(t, result["token"])
	userMap := result["user"].(map[string]interface{})
	assert.Equal(t, "login@example.com", userMap["email"])
}

func TestLogin_InvalidPassword(t *testing.T) {
	env := setupAuthRouter()

	// Seed an approved user in the DB
	authSvc := services.NewAuthService()
	hashed, _ := authSvc.HashPassword("mypassword")
	database.DB.Create(&models.User{
		Email:      "login2@example.com",
		Password:   hashed,
		Name:       "Login User",
		IsApproved: true,
	})

	// Now try to login with WRONG password
	loginBody := map[string]string{
		"email":    "login2@example.com",
		"password": "wrongpassword",
	}
	lb, _ := json.Marshal(loginBody)
	reqLogin := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(lb))
	reqLogin.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(reqLogin, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

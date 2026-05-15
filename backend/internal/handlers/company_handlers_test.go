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

func setupCompanyRouter() *testutils.FiberApp {
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

	app.Post("/companies", handlers.CreateCompany)
	app.Get("/companies", handlers.GetCompanies)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestCreateCompany_Success(t *testing.T) {
	env := setupCompanyRouter()

	user := models.User{
		Email: "owner@example.com",
		Name:  "Company Owner",
	}
	database.DB.Create(&user)

	body := map[string]string{
		"name":        "Test Company",
		"description": "A company for testing",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/companies", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var company models.Company
	database.DB.First(&company, "name = ?", "Test Company")
	assert.Equal(t, "A company for testing", company.Description)
	assert.Equal(t, user.ID, company.UserID)
}

func TestGetCompanies_Success(t *testing.T) {
	env := setupCompanyRouter()

	user1 := models.User{Email: "u1@test.com"}
	database.DB.Create(&user1)

	user2 := models.User{Email: "u2@test.com"}
	database.DB.Create(&user2)

	// User1's company
	comp1 := models.Company{Name: "C1", UserID: user1.ID}
	database.DB.Create(&comp1)

	// User2's company where User1 is a member
	comp2 := models.Company{Name: "C2", UserID: user2.ID}
	database.DB.Create(&comp2)
	database.DB.Create(&models.CompanyMember{CompanyID: comp2.ID, UserID: user1.ID, Role: "member"})

	// User2's private company
	comp3 := models.Company{Name: "C3", UserID: user2.ID}
	database.DB.Create(&comp3)

	req := httptest.NewRequest("GET", "/companies", nil)
	req.Header.Set("Test-User-ID", user1.ID.String())

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []models.Company
	json.NewDecoder(resp.Body).Decode(&result)

	// User1 should see C1 (owner) and C2 (member), but NOT C3
	assert.Len(t, result, 2)
	names := []string{result[0].Name, result[1].Name}
	assert.Contains(t, names, "C1")
	assert.Contains(t, names, "C2")
	assert.NotContains(t, names, "C3")
}

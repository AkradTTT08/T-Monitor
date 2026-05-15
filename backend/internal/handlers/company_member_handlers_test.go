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

func setupCompanyMemberRouter() *testutils.FiberApp {
	db := testutils.SetupTestDB()
	app := testutils.SetupTestApp()

	// Mocking auth middleware
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			uid, _ := uuid.Parse(userID)
			c.Locals("user_id", uid)
		}
		return c.Next()
	})

	app.Post("/companies/:id/invite", handlers.InviteMemberByEmail)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestInviteMember_Success(t *testing.T) {
	env := setupCompanyMemberRouter()

	inviter := models.User{Email: "inviter@example.com"}
	database.DB.Create(&inviter)

	invitee := models.User{Email: "invitee@example.com"}
	database.DB.Create(&invitee)

	company := models.Company{Name: "Inviter Company", UserID: inviter.ID}
	database.DB.Create(&company)

	body := map[string]string{
		"email": "invitee@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/companies/"+company.ID.String()+"/invite", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", inviter.ID.String())

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var invitation models.CompanyInvitation
	database.DB.First(&invitation)
	assert.Equal(t, company.ID, invitation.CompanyID)
	assert.Equal(t, invitee.ID, invitation.InviteeID)
	assert.Equal(t, "pending", invitation.Status)
}

func TestInviteMember_UserNotFound(t *testing.T) {
	env := setupCompanyMemberRouter()

	inviter := models.User{Email: "inviter2@example.com"}
	database.DB.Create(&inviter)

	company := models.Company{Name: "Inviter 2 Company", UserID: inviter.ID}
	database.DB.Create(&company)

	body := map[string]string{
		"email": "doesntexist@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/companies/"+company.ID.String()+"/invite", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", inviter.ID.String())

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

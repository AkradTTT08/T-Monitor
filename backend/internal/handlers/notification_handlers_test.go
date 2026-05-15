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

func setupNotificationRouter() *testutils.FiberApp {
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

	app.Get("/notifications", handlers.GetNotifications)
	app.Put("/notifications/:id/read", handlers.MarkNotificationRead)
	app.Put("/notifications/read-all", handlers.MarkAllNotificationsRead)
	app.Get("/notifications/config/:projectId", handlers.GetNotificationConfig)
	app.Post("/notifications/config", handlers.UpsertNotificationConfig)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestGetNotificationConfig_Success(t *testing.T) {
	env := setupNotificationRouter()

	user := models.User{Email: "notifowner@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Notif Project", UserID: user.ID}
	database.DB.Create(&project)

	// Test getting config when it doesn't exist (should return default object with project ID)
	req := httptest.NewRequest("GET", "/notifications/config/"+project.ID.String(), nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var config models.NotificationConfig
	json.NewDecoder(resp.Body).Decode(&config)
	assert.Equal(t, project.ID, config.ProjectID)
	assert.False(t, config.EnableTelegram)
}

func TestUpsertNotificationConfig_Success(t *testing.T) {
	env := setupNotificationRouter()

	user := models.User{Email: "notifupdater@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Upsert Project", UserID: user.ID}
	database.DB.Create(&project)

	body := map[string]interface{}{
		"project_id":         project.ID.String(),
		"enable_telegram":    true,
		"telegram_bot_token": "token123",
		"telegram_chat_id":   "chat123",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/notifications/config", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var config models.NotificationConfig
	database.DB.First(&config, "project_id = ?", project.ID)
	assert.True(t, config.EnableTelegram)
	assert.Equal(t, "token123", config.TelegramBotToken)
	assert.Equal(t, "chat123", config.TelegramChatID)
	
	// Test updating (setting it to false)
	updateBody := map[string]interface{}{
		"project_id":         project.ID.String(),
		"enable_telegram":    false,
		"telegram_bot_token": "token123",
		"telegram_chat_id":   "chat123",
	}
	updateBytes, _ := json.Marshal(updateBody)
	
	reqUpdate := httptest.NewRequest("POST", "/notifications/config", bytes.NewReader(updateBytes))
	reqUpdate.Header.Set("Content-Type", "application/json")
	reqUpdate.Header.Set("Test-User-ID", user.ID.String())
	reqUpdate.Header.Set("Test-Role", "user")
	
	respUpdate, _ := env.App.Test(reqUpdate, -1)
	assert.Equal(t, http.StatusOK, respUpdate.StatusCode)
	
	var updatedConfig models.NotificationConfig
	database.DB.First(&updatedConfig, "project_id = ?", project.ID)
	assert.False(t, updatedConfig.EnableTelegram)
}

func TestDashboardNotifications_Success(t *testing.T) {
	env := setupNotificationRouter()

	user := models.User{Email: "dashuser@example.com"}
	database.DB.Create(&user)

	// Create a few notifications
	notif1 := models.DashboardNotification{UserID: user.ID, Title: "Test 1", Message: "Msg 1", IsRead: false}
	notif2 := models.DashboardNotification{UserID: user.ID, Title: "Test 2", Message: "Msg 2", IsRead: false}
	notifSystem := models.DashboardNotification{UserID: uuid.Nil, Title: "System", Message: "Maintenance", IsRead: false}
	database.DB.Create(&notif1)
	database.DB.Create(&notif2)
	database.DB.Create(&notifSystem)

	// 1. Get Notifications
	reqGet := httptest.NewRequest("GET", "/notifications", nil)
	reqGet.Header.Set("Test-User-ID", user.ID.String())
	reqGet.Header.Set("Test-Role", "user")

	respGet, err := env.App.Test(reqGet, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respGet.StatusCode)

	var fetchedNotifs []models.DashboardNotification
	json.NewDecoder(respGet.Body).Decode(&fetchedNotifs)
	assert.GreaterOrEqual(t, len(fetchedNotifs), 3)

	// 2. Mark one as read
	reqMark := httptest.NewRequest("PUT", "/notifications/"+notif1.ID.String()+"/read", nil)
	reqMark.Header.Set("Test-User-ID", user.ID.String())
	reqMark.Header.Set("Test-Role", "user")

	respMark, _ := env.App.Test(reqMark, -1)
	assert.Equal(t, http.StatusOK, respMark.StatusCode)

	var checkNotif1 models.DashboardNotification
	database.DB.First(&checkNotif1, "id = ?", notif1.ID)
	assert.True(t, checkNotif1.IsRead)

	// 3. Mark all as read
	reqMarkAll := httptest.NewRequest("PUT", "/notifications/read-all", nil)
	reqMarkAll.Header.Set("Test-User-ID", user.ID.String())
	reqMarkAll.Header.Set("Test-Role", "user")

	respMarkAll, _ := env.App.Test(reqMarkAll, -1)
	assert.Equal(t, http.StatusOK, respMarkAll.StatusCode)

	var checkNotif2 models.DashboardNotification
	database.DB.First(&checkNotif2, "id = ?", notif2.ID)
	assert.True(t, checkNotif2.IsRead)

	var checkSystemNotif models.DashboardNotification
	database.DB.First(&checkSystemNotif, "id = ?", notifSystem.ID)
	assert.True(t, checkSystemNotif.IsRead)
}

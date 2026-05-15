package handlers_test

import (
	"encoding/json"
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

func setupAnalyticsRouter() *testutils.FiberApp {
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

	app.Get("/analytics/uptime", handlers.GetUptimeStats)
	app.Get("/analytics/latency", handlers.GetLatencyTrend)
	app.Get("/analytics/incidents", handlers.GetIncidentTimeline)
	app.Get("/analytics/global", handlers.GetGlobalPulse)

	return &testutils.FiberApp{App: app, DB: db}
}

func TestGetUptimeStats_Success(t *testing.T) {
	env := setupAnalyticsRouter()

	user := models.User{Email: "analytics@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Analytics Project", UserID: user.ID}
	database.DB.Create(&project)

	api := models.API{Name: "Test API", ProjectID: project.ID}
	database.DB.Create(&api)

	// Create some monitor logs (2 success, 1 fail)
	now := time.Now()
	log1 := models.MonitorLog{ApiID: api.ID, IsSuccess: true, ResponseTime: 100, CheckedAt: now.Add(-1 * time.Hour)}
	log2 := models.MonitorLog{ApiID: api.ID, IsSuccess: true, ResponseTime: 150, CheckedAt: now.Add(-2 * time.Hour)}
	log3 := models.MonitorLog{ApiID: api.ID, IsSuccess: false, ResponseTime: 500, CheckedAt: now.Add(-3 * time.Hour)}
	database.DB.Create(&log1)
	database.DB.Create(&log2)
	database.DB.Create(&log3)

	req := httptest.NewRequest("GET", "/analytics/uptime?project_id="+project.ID.String(), nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "24h", result["period"])
	assert.Equal(t, float64(3), result["total_checks"])
	assert.Equal(t, float64(1), result["total_failures"]) // 1 fail
	assert.Equal(t, 66.67, result["overall_uptime"])      // 2/3 = 66.67%
}

func TestGetLatencyTrend_Success(t *testing.T) {
	env := setupAnalyticsRouter()

	user := models.User{Email: "trend@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Trend Project", UserID: user.ID}
	database.DB.Create(&project)

	api := models.API{Name: "Trend API", ProjectID: project.ID}
	database.DB.Create(&api)

	// Create some monitor logs
	now := time.Now()
	database.DB.Create(&models.MonitorLog{ApiID: api.ID, IsSuccess: true, ResponseTime: 100, CheckedAt: now.Add(-1 * time.Hour)})

	req := httptest.NewRequest("GET", "/analytics/latency?project_id="+project.ID.String(), nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "24h", result["period"])
	dataPoints, ok := result["data_points"].([]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(dataPoints), 1)
}

func TestGetGlobalPulse_Success(t *testing.T) {
	env := setupAnalyticsRouter()

	user := models.User{Email: "pulse@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Pulse Project", UserID: user.ID}
	database.DB.Create(&project)

	req := httptest.NewRequest("GET", "/analytics/global", nil)
	req.Header.Set("Test-User-ID", user.ID.String())
	req.Header.Set("Test-Role", "user")

	resp, err := env.App.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["global_uptime"])
	assert.NotNil(t, result["avg_latency"])
}

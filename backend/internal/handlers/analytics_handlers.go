package handlers

import (
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

// GetUptimeStats returns uptime statistics for all APIs in a project
func GetUptimeStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	projectID := c.Query("project_id")
	period := c.Query("period", "24h")

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	// Verify ownership unless admin
	if role != "admin" {
		var project models.Project
		if err := database.DB.First(&project, "id = ? AND user_id = ?", projectID, userID).Error; err != nil {
			// Check if user is a member
			var member models.ProjectMember
			if err := database.DB.First(&member, "project_id = ? AND user_id = ?", projectID, userID).Error; err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
			}
		}
	}

	// Calculate time range
	now := time.Now()
	var since time.Time
	switch period {
	case "7d":
		since = now.AddDate(0, 0, -7)
	case "30d":
		since = now.AddDate(0, 0, -30)
	default: // 24h
		since = now.Add(-24 * time.Hour)
	}

	// Get all APIs for this project
	var apis []models.API
	database.DB.Where("project_id = ?", projectID).Find(&apis)

	type APIUptime struct {
		ID            uuid.UUID `json:"id"`
		Name          string    `json:"name"`
		Method        string    `json:"method"`
		URL           string    `json:"url"`
		UptimePercent float64   `json:"uptime_percent"`
		AvgLatency    float64   `json:"avg_latency"`
		MaxLatency    int64     `json:"max_latency"`
		MinLatency    int64     `json:"min_latency"`
		TotalChecks   int64     `json:"total_checks"`
		FailCount     int64     `json:"fail_count"`
		LastChecked   *time.Time `json:"last_checked"`
	}

	results := make([]APIUptime, 0, len(apis))
	var overallTotal int64
	var overallSuccess int64

	for _, api := range apis {
		var totalChecks int64
		var successCount int64
		var failCount int64

		// Count totals
		database.DB.Model(&models.MonitorLog{}).
			Where("api_id = ? AND checked_at >= ?", api.ID, since).
			Count(&totalChecks)

		database.DB.Model(&models.MonitorLog{}).
			Where("api_id = ? AND checked_at >= ? AND is_success = true", api.ID, since).
			Count(&successCount)

		failCount = totalChecks - successCount

		// Calculate uptime
		var uptimePercent float64
		if totalChecks > 0 {
			uptimePercent = math.Round((float64(successCount)/float64(totalChecks))*10000) / 100
		}

		// Get avg/max/min latency from successful checks
		type LatencyStats struct {
			AvgLatency float64
			MaxLatency int64
			MinLatency int64
		}
		var stats LatencyStats
		database.DB.Model(&models.MonitorLog{}).
			Select("COALESCE(AVG(response_time), 0) as avg_latency, COALESCE(MAX(response_time), 0) as max_latency, COALESCE(MIN(response_time), 0) as min_latency").
			Where("api_id = ? AND checked_at >= ? AND is_success = true", api.ID, since).
			Scan(&stats)

		// Get last checked time
		var lastLog models.MonitorLog
		var lastChecked *time.Time
		if err := database.DB.Where("api_id = ?", api.ID).Order("checked_at DESC").First(&lastLog).Error; err == nil {
			lastChecked = &lastLog.CheckedAt
		}

		results = append(results, APIUptime{
			ID:            api.ID,
			Name:          api.Name,
			Method:        api.Method,
			URL:           api.URL,
			UptimePercent: uptimePercent,
			AvgLatency:    math.Round(stats.AvgLatency*100) / 100,
			MaxLatency:    stats.MaxLatency,
			MinLatency:    stats.MinLatency,
			TotalChecks:   totalChecks,
			FailCount:     failCount,
			LastChecked:   lastChecked,
		})

		overallTotal += totalChecks
		overallSuccess += successCount
	}

	var overallUptime float64
	if overallTotal > 0 {
		overallUptime = math.Round((float64(overallSuccess)/float64(overallTotal))*10000) / 100
	}

	return c.JSON(fiber.Map{
		"apis":           results,
		"overall_uptime": overallUptime,
		"total_checks":   overallTotal,
		"total_failures": overallTotal - overallSuccess,
		"period":         period,
	})
}

// GetLatencyTrend returns time-series latency data for charts
func GetLatencyTrend(c *fiber.Ctx) error {
	projectID := c.Query("project_id")
	period := c.Query("period", "24h")

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	now := time.Now()
	var since time.Time
	var groupFormat string

	switch period {
	case "7d":
		since = now.AddDate(0, 0, -7)
		groupFormat = "YYYY-MM-DD HH24" // hourly for 7 days
	case "30d":
		since = now.AddDate(0, 0, -30)
		groupFormat = "YYYY-MM-DD" // daily for 30 days
	default: // 24h
		since = now.Add(-24 * time.Hour)
		groupFormat = "YYYY-MM-DD HH24" // hourly for 24h
	}

	type DataPoint struct {
		Timestamp   string  `json:"timestamp"`
		AvgLatency  float64 `json:"avg_latency"`
		MaxLatency  float64 `json:"max_latency"`
		MinLatency  float64 `json:"min_latency"`
		TotalChecks int64   `json:"total_checks"`
		FailCount   int64   `json:"fail_count"`
		SuccessRate float64 `json:"success_rate"`
	}

	var dataPoints []DataPoint

	database.DB.Model(&models.MonitorLog{}).
		Select(`
			TO_CHAR(checked_at, '`+groupFormat+`') as timestamp,
			ROUND(AVG(response_time)::numeric, 2) as avg_latency,
			MAX(response_time) as max_latency,
			MIN(response_time) as min_latency,
			COUNT(*) as total_checks,
			COUNT(*) FILTER (WHERE is_success = false) as fail_count,
			ROUND((COUNT(*) FILTER (WHERE is_success = true)::numeric / NULLIF(COUNT(*)::numeric, 0)) * 100, 2) as success_rate
		`).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id = ? AND checked_at >= ?", projectID, since).
		Group("timestamp").
		Order("timestamp ASC").
		Scan(&dataPoints)

	return c.JSON(fiber.Map{
		"data_points": dataPoints,
		"period":      period,
	})
}

// GetIncidentTimeline returns recent incidents for timeline display
func GetIncidentTimeline(c *fiber.Ctx) error {
	projectID := c.Query("project_id")
	limit := c.QueryInt("limit", 20)

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	if limit > 100 {
		limit = 100
	}

	type Incident struct {
		ID           uuid.UUID `json:"id"`
		APIName      string    `json:"api_name"`
		APIURL       string    `json:"api_url"`
		APIMethod    string    `json:"api_method"`
		ErrorMessage string    `json:"error_message"`
		StatusCode   int       `json:"status_code"`
		ResponseTime int64     `json:"response_time"`
		CheckedAt    time.Time `json:"checked_at"`
	}

	var incidents []Incident

	database.DB.Model(&models.MonitorLog{}).
		Select("monitor_logs.id, apis.name as api_name, apis.url as api_url, apis.method as api_method, monitor_logs.error_message, monitor_logs.status_code, monitor_logs.response_time, monitor_logs.checked_at").
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id = ? AND monitor_logs.is_success = false", projectID).
		Order("monitor_logs.checked_at DESC").
		Limit(limit).
		Scan(&incidents)

	return c.JSON(fiber.Map{
		"incidents": incidents,
		"total":     len(incidents),
	})
}

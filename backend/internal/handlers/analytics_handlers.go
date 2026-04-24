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

	if len(apis) == 0 {
		return c.JSON(fiber.Map{
			"apis":           []interface{}{},
			"overall_uptime": 0,
			"total_checks":   0,
			"total_failures": 0,
			"period":         period,
		})
	}

	apiIDs := make([]uuid.UUID, len(apis))
	for i, api := range apis {
		apiIDs[i] = api.ID
	}

	// 1. Get Aggregated Stats (Total, Success, Latency) for all APIs in one go
	type APIAggregatedStats struct {
		ApiID         uuid.UUID
		TotalChecks   int64
		SuccessChecks int64
		AvgLatency    float64
		MaxLatency    int64
		MinLatency    int64
	}
	var aggStats []APIAggregatedStats
	database.DB.Model(&models.MonitorLog{}).
		Select("api_id, COUNT(*) as total_checks, COUNT(*) FILTER (WHERE is_success = true) as success_checks, AVG(response_time) as avg_latency, MAX(response_time) as max_latency, MIN(response_time) as min_latency").
		Where("api_id IN ? AND checked_at >= ?", apiIDs, since).
		Group("api_id").
		Scan(&aggStats)

	// Create a map for quick lookup
	statsMap := make(map[uuid.UUID]APIAggregatedStats)
	for _, s := range aggStats {
		statsMap[s.ApiID] = s
	}

	// 2. Get Last Checked Time for all APIs
	type LastLog struct {
		ApiID     uuid.UUID
		CheckedAt time.Time
	}
	var lastLogs []LastLog
	// Using a subquery/distinct on to get the latest log for each API
	database.DB.Model(&models.MonitorLog{}).
		Select("DISTINCT ON (api_id) api_id, checked_at").
		Where("api_id IN ?", apiIDs).
		Order("api_id, checked_at DESC").
		Scan(&lastLogs)

	lastLogMap := make(map[uuid.UUID]time.Time)
	for _, l := range lastLogs {
		lastLogMap[l.ApiID] = l.CheckedAt
	}

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
		stats, ok := statsMap[api.ID]
		if !ok {
			// No logs for this period
			results = append(results, APIUptime{
				ID:            api.ID,
				Name:          api.Name,
				Method:        api.Method,
				URL:           api.URL,
				UptimePercent: 0,
				AvgLatency:    0,
				MaxLatency:    0,
				MinLatency:    0,
				TotalChecks:   0,
				FailCount:     0,
				LastChecked:   nil,
			})
			continue
		}

		uptimePercent := 0.0
		if stats.TotalChecks > 0 {
			uptimePercent = math.Round((float64(stats.SuccessChecks)/float64(stats.TotalChecks))*10000) / 100
		}

		var lastChecked *time.Time
		if t, exists := lastLogMap[api.ID]; exists {
			lastChecked = &t
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
			TotalChecks:   stats.TotalChecks,
			FailCount:     stats.TotalChecks - stats.SuccessChecks,
			LastChecked:   lastChecked,
		})

		overallTotal += stats.TotalChecks
		overallSuccess += stats.SuccessChecks
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
		Timestamp   string  `json:"timestamp" gorm:"column:bucket_time"`
		AvgLatency  float64 `json:"avg_latency"`
		MaxLatency  float64 `json:"max_latency"`
		MinLatency  float64 `json:"min_latency"`
		TotalChecks int64   `json:"total_checks"`
		FailCount   int64   `json:"fail_count"`
		SuccessRate float64 `json:"success_rate"`
	}

	var dataPoints []DataPoint

	// Using explicit table names and avoiding 'timestamp' keyword
	err := database.DB.Model(&models.MonitorLog{}).
		Select(`
			TO_CHAR(monitor_logs.checked_at, '`+groupFormat+`') as bucket_time,
			ROUND(AVG(monitor_logs.response_time)::numeric, 2) as avg_latency,
			MAX(monitor_logs.response_time) as max_latency,
			MIN(monitor_logs.response_time) as min_latency,
			COUNT(*) as total_checks,
			COUNT(*) FILTER (WHERE monitor_logs.is_success = false) as fail_count,
			ROUND((COUNT(*) FILTER (WHERE monitor_logs.is_success = true)::numeric / NULLIF(COUNT(*)::numeric, 0)) * 100, 2) as success_rate
		`).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id = ? AND monitor_logs.checked_at >= ? AND apis.deleted_at IS NULL", projectID, since).
		Group("bucket_time").
		Order("bucket_time ASC").
		Scan(&dataPoints).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch trend data: " + err.Error()})
	}

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

// GetGlobalPulse returns high-level metrics and recent pings across all accessible projects
func GetGlobalPulse(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	
	companyID := c.Query("company_id")
	projectID := c.Query("project_id")

	now := time.Now()
	since24h := now.Add(-24 * time.Hour)

	var accessibleProjectIDs []uuid.UUID

	if role == "admin" {
		query := database.DB.Model(&models.Project{})
		if companyID != "" {
			query = query.Where("company_id = ?", companyID)
		}
		if projectID != "" && projectID != "all" {
			query = query.Where("id = ?", projectID)
		}
		query.Pluck("id", &accessibleProjectIDs)
	} else {
		// Projects owned by user
		var owned []uuid.UUID
		q1 := database.DB.Model(&models.Project{}).Where("user_id = ?", userID)
		if companyID != "" {
			q1 = q1.Where("company_id = ?", companyID)
		}
		if projectID != "" && projectID != "all" {
			q1 = q1.Where("id = ?", projectID)
		}
		q1.Pluck("id", &owned)

		// Projects user is member of
		var memberOf []uuid.UUID
		q2 := database.DB.Model(&models.ProjectMember{}).
			Select("project_members.project_id").
			Joins("JOIN projects ON projects.id = project_members.project_id").
			Where("project_members.user_id = ?", userID)
		
		if companyID != "" {
			q2 = q2.Where("projects.company_id = ?", companyID)
		}
		if projectID != "" && projectID != "all" {
			q2 = q2.Where("project_members.project_id = ?", projectID)
		}
		q2.Pluck("project_id", &memberOf)
		
		// Uniquify IDs
		idMap := make(map[uuid.UUID]bool)
		for _, id := range owned {
			if !idMap[id] {
				accessibleProjectIDs = append(accessibleProjectIDs, id)
				idMap[id] = true
			}
		}
		for _, id := range memberOf {
			if !idMap[id] {
				accessibleProjectIDs = append(accessibleProjectIDs, id)
				idMap[id] = true
			}
		}
	}

	if len(accessibleProjectIDs) == 0 {
		return c.JSON(fiber.Map{
			"active_apis":    0,
			"global_uptime":  100,
			"avg_latency":    0,
			"recent_pings":   []interface{}{},
		})
	}

	// 1. Active APIs count
	var activeAPIsCount int64
	database.DB.Model(&models.API{}).Where("project_id IN ? AND is_active = true AND deleted_at IS NULL", accessibleProjectIDs).Count(&activeAPIsCount)

	// 2. Global Uptime (Last 24h)
	var totalChecks, successChecks int64
	database.DB.Model(&models.MonitorLog{}).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id IN ? AND monitor_logs.checked_at >= ?", accessibleProjectIDs, since24h).
		Count(&totalChecks)

	database.DB.Model(&models.MonitorLog{}).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id IN ? AND monitor_logs.checked_at >= ? AND monitor_logs.is_success = true", accessibleProjectIDs, since24h).
		Count(&successChecks)

	var globalUptime float64 = 100.0
	if totalChecks > 0 {
		globalUptime = math.Round((float64(successChecks)/float64(totalChecks))*10000) / 100
	}

	// 3. Average Global Latency (Last 24h, successful only)
	var avgLatency float64
	database.DB.Model(&models.MonitorLog{}).
		Select("COALESCE(AVG(monitor_logs.response_time), 0)").
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id IN ? AND monitor_logs.checked_at >= ? AND monitor_logs.is_success = true", accessibleProjectIDs, since24h).
		Scan(&avgLatency)
	avgLatency = math.Round(avgLatency*100) / 100

	// 4. Live Pings (Last 50)
	type Ping struct {
		ID           uuid.UUID `json:"id"`
		APIName      string    `json:"api_name"`
		ProjectName  string    `json:"project_name"`
		URL          string    `json:"url"`
		Method       string    `json:"method"`
		IsSuccess    bool      `json:"is_success"`
		ResponseTime int64     `json:"response_time"`
		StatusCode   int       `json:"status_code"`
		CheckedAt    time.Time `json:"checked_at"`
	}

	var recentPings []Ping
	database.DB.Model(&models.MonitorLog{}).
		Select(`
			monitor_logs.id, apis.name as api_name, projects.name as project_name, 
			apis.url, apis.method, monitor_logs.is_success, 
			monitor_logs.response_time, monitor_logs.status_code, monitor_logs.checked_at
		`).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Joins("JOIN projects ON projects.id = apis.project_id").
		Where("apis.project_id IN ? AND apis.deleted_at IS NULL", accessibleProjectIDs).
		Order("monitor_logs.checked_at DESC").
		Limit(50).
		Scan(&recentPings)

	return c.JSON(fiber.Map{
		"active_apis":    activeAPIsCount,
		"global_uptime":  globalUptime,
		"avg_latency":    avgLatency,
		"recent_pings":   recentPings,
	})
}

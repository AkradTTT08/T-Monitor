package services

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

type AnalyticsService interface {
	GetUptimeStats(projectID string, period string, userID uuid.UUID, role string) (map[string]interface{}, error)
	GetLatencyTrend(projectID string, period string, userID uuid.UUID, role string) (map[string]interface{}, error)
	GetIncidentTimeline(projectID string, limit int) (map[string]interface{}, error)
	GetGlobalPulse(userID uuid.UUID, role string, companyID string, projectID string) (map[string]interface{}, error)
}

type analyticsService struct {
	db *gorm.DB
}

func NewAnalyticsService(db *gorm.DB) AnalyticsService {
	if db == nil {
		db = database.DB
	}
	return &analyticsService{db: db}
}

// verifyProjectAccess checks if the user has access to the project
func (s *analyticsService) verifyProjectAccess(projectID string, userID uuid.UUID, role string) error {
	var project models.Project
	if role == "admin" {
		if err := s.db.First(&project, "id = ?", projectID).Error; err != nil {
			return errors.New("Project not found")
		}
	} else {
		if err := s.db.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return errors.New("Access denied")
		}
	}
	return nil
}

func (s *analyticsService) GetUptimeStats(projectID string, period string, userID uuid.UUID, role string) (map[string]interface{}, error) {
	if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
		return nil, err
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
	s.db.Where("project_id = ?", projectID).Find(&apis)

	if len(apis) == 0 {
		return map[string]interface{}{
			"apis":           []interface{}{},
			"overall_uptime": float64(0),
			"total_checks":   int64(0),
			"total_failures": int64(0),
			"period":         period,
		}, nil
	}

	apiIDs := make([]uuid.UUID, len(apis))
	for i, api := range apis {
		apiIDs[i] = api.ID
	}

	// 1. Get Aggregated Stats
	type APIAggregatedStats struct {
		ApiID         uuid.UUID
		TotalChecks   int64
		SuccessChecks int64
		AvgLatency    float64
		MaxLatency    int64
		MinLatency    int64
	}
	var aggStats []APIAggregatedStats
	s.db.Model(&models.MonitorLog{}).
		Select("api_id, COUNT(*) as total_checks, COUNT(*) FILTER (WHERE is_success = true) as success_checks, COALESCE(AVG(response_time), 0) as avg_latency, COALESCE(MAX(response_time), 0) as max_latency, COALESCE(MIN(response_time), 0) as min_latency").
		Where("api_id IN ? AND checked_at >= ?", apiIDs, since).
		Group("api_id").
		Scan(&aggStats)

	statsMap := make(map[uuid.UUID]APIAggregatedStats)
	for _, st := range aggStats {
		statsMap[st.ApiID] = st
	}

	// 2. Get Last Checked Time
	type LastLog struct {
		ApiID     uuid.UUID
		CheckedAt time.Time
	}
	var lastLogs []LastLog
	// Note: For SQLite compatibility in tests, we cannot use DISTINCT ON.
	// We'll just fetch the MAX(checked_at) grouped by api_id
	s.db.Model(&models.MonitorLog{}).
		Select("api_id, MAX(checked_at) as checked_at").
		Where("api_id IN ?", apiIDs).
		Group("api_id").
		Scan(&lastLogs)

	lastLogMap := make(map[uuid.UUID]time.Time)
	for _, l := range lastLogs {
		lastLogMap[l.ApiID] = l.CheckedAt
	}

	type APIUptime struct {
		ID            uuid.UUID  `json:"id"`
		Name          string     `json:"name"`
		Method        string     `json:"method"`
		URL           string     `json:"url"`
		UptimePercent float64    `json:"uptime_percent"`
		AvgLatency    float64    `json:"avg_latency"`
		MaxLatency    int64      `json:"max_latency"`
		MinLatency    int64      `json:"min_latency"`
		TotalChecks   int64      `json:"total_checks"`
		FailCount     int64      `json:"fail_count"`
		LastChecked   *time.Time `json:"last_checked"`
	}

	results := make([]APIUptime, 0, len(apis))
	var overallTotal int64
	var overallSuccess int64

	for _, api := range apis {
		st, ok := statsMap[api.ID]
		if !ok {
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
		if st.TotalChecks > 0 {
			uptimePercent = math.Round((float64(st.SuccessChecks)/float64(st.TotalChecks))*10000) / 100
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
			AvgLatency:    math.Round(st.AvgLatency*100) / 100,
			MaxLatency:    st.MaxLatency,
			MinLatency:    st.MinLatency,
			TotalChecks:   st.TotalChecks,
			FailCount:     st.TotalChecks - st.SuccessChecks,
			LastChecked:   lastChecked,
		})

		overallTotal += st.TotalChecks
		overallSuccess += st.SuccessChecks
	}

	var overallUptime float64
	if overallTotal > 0 {
		overallUptime = math.Round((float64(overallSuccess)/float64(overallTotal))*10000) / 100
	}

	return map[string]interface{}{
		"apis":           results,
		"overall_uptime": overallUptime,
		"total_checks":   overallTotal,
		"total_failures": overallTotal - overallSuccess,
		"period":         period,
	}, nil
}

func (s *analyticsService) GetLatencyTrend(projectID string, period string, userID uuid.UUID, role string) (map[string]interface{}, error) {
	if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
		return nil, err
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

	// To support SQLite in tests which doesn't have TO_CHAR, we use STRFTIME if SQLite, else TO_CHAR
	isSQLite := s.db.Dialector.Name() == "sqlite"
	
	var selectQuery string
	if isSQLite {
		sqliteFormat := "%Y-%m-%d %H"
		if period == "30d" {
			sqliteFormat = "%Y-%m-%d"
		}
		selectQuery = `
			STRFTIME('` + sqliteFormat + `', monitor_logs.checked_at) as bucket_time,
			ROUND(AVG(monitor_logs.response_time), 2) as avg_latency,
			MAX(monitor_logs.response_time) as max_latency,
			MIN(monitor_logs.response_time) as min_latency,
			COUNT(*) as total_checks,
			SUM(CASE WHEN monitor_logs.is_success = 0 THEN 1 ELSE 0 END) as fail_count,
			ROUND((CAST(SUM(CASE WHEN monitor_logs.is_success = 1 THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*)) * 100, 2) as success_rate
		`
	} else {
		selectQuery = `
			TO_CHAR(monitor_logs.checked_at, '` + groupFormat + `') as bucket_time,
			ROUND(AVG(monitor_logs.response_time)::numeric, 2) as avg_latency,
			MAX(monitor_logs.response_time) as max_latency,
			MIN(monitor_logs.response_time) as min_latency,
			COUNT(*) as total_checks,
			COUNT(*) FILTER (WHERE monitor_logs.is_success = false) as fail_count,
			ROUND((COUNT(*) FILTER (WHERE monitor_logs.is_success = true)::numeric / NULLIF(COUNT(*)::numeric, 0)) * 100, 2) as success_rate
		`
	}

	err := s.db.Model(&models.MonitorLog{}).
		Select(selectQuery).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id = ? AND monitor_logs.checked_at >= ? AND apis.deleted_at IS NULL", projectID, since).
		Group("bucket_time").
		Order("bucket_time ASC").
		Scan(&dataPoints).Error

	if err != nil {
		return nil, errors.New("Failed to fetch trend data")
	}

	return map[string]interface{}{
		"data_points": dataPoints,
		"period":      period,
	}, nil
}

func (s *analyticsService) GetIncidentTimeline(projectID string, limit int) (map[string]interface{}, error) {
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

	// Using SQLite compatibility for is_success (0 instead of false)
	isSuccessVal := interface{}(false)
	if s.db.Dialector.Name() == "sqlite" {
		isSuccessVal = 0
	}

	s.db.Model(&models.MonitorLog{}).
		Select("monitor_logs.id, apis.name as api_name, apis.url as api_url, apis.method as api_method, monitor_logs.error_message, monitor_logs.status_code, monitor_logs.response_time, monitor_logs.checked_at").
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id = ? AND monitor_logs.is_success = ?", projectID, isSuccessVal).
		Order("monitor_logs.checked_at DESC").
		Limit(limit).
		Scan(&incidents)

	return map[string]interface{}{
		"incidents": incidents,
		"total":     len(incidents),
	}, nil
}

func (s *analyticsService) GetGlobalPulse(userID uuid.UUID, role string, companyID string, projectID string) (map[string]interface{}, error) {
	now := time.Now()
	since24h := now.Add(-24 * time.Hour)

	var accessibleProjectIDs []uuid.UUID

	if role == "admin" {
		query := s.db.Model(&models.Project{})
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
		q1 := s.db.Model(&models.Project{}).Where("user_id = ?", userID)
		if companyID != "" {
			q1 = q1.Where("company_id = ?", companyID)
		}
		if projectID != "" && projectID != "all" {
			q1 = q1.Where("id = ?", projectID)
		}
		q1.Pluck("id", &owned)

		// Projects user is member of
		var memberOf []uuid.UUID
		q2 := s.db.Model(&models.ProjectMember{}).
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
		return map[string]interface{}{
			"active_apis":    0,
			"global_uptime":  100,
			"avg_latency":    0,
			"recent_pings":   []interface{}{},
		}, nil
	}

	isSQLite := s.db.Dialector.Name() == "sqlite"
	isActiveVal := interface{}(true)
	isSuccessVal := interface{}(true)
	if isSQLite {
		isActiveVal = 1
		isSuccessVal = 1
	}

	// 1. Active APIs count
	var activeAPIsCount int64
	s.db.Model(&models.API{}).Where("project_id IN ? AND is_active = ? AND deleted_at IS NULL", accessibleProjectIDs, isActiveVal).Count(&activeAPIsCount)

	// 2. Global Uptime (Last 24h)
	var totalChecks, successChecks int64
	s.db.Model(&models.MonitorLog{}).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id IN ? AND monitor_logs.checked_at >= ?", accessibleProjectIDs, since24h).
		Count(&totalChecks)

	s.db.Model(&models.MonitorLog{}).
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id IN ? AND monitor_logs.checked_at >= ? AND monitor_logs.is_success = ?", accessibleProjectIDs, since24h, isSuccessVal).
		Count(&successChecks)

	var globalUptime float64 = 100.0
	if totalChecks > 0 {
		globalUptime = math.Round((float64(successChecks)/float64(totalChecks))*10000) / 100
	}

	// 3. Average Global Latency
	var avgLatency float64
	s.db.Model(&models.MonitorLog{}).
		Select("COALESCE(AVG(monitor_logs.response_time), 0)").
		Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id IN ? AND monitor_logs.checked_at >= ? AND monitor_logs.is_success = ?", accessibleProjectIDs, since24h, isSuccessVal).
		Scan(&avgLatency)
	avgLatency = math.Round(avgLatency*100) / 100

	// 4. Live Pings
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
	s.db.Model(&models.MonitorLog{}).
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

	return map[string]interface{}{
		"active_apis":    activeAPIsCount,
		"global_uptime":  globalUptime,
		"avg_latency":    avgLatency,
		"recent_pings":   recentPings,
	}, nil
}

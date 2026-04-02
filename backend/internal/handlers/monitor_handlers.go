package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

func GetMonitorLogs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var logs []models.MonitorLog

	query := database.DB.Preload("API", func(db *gorm.DB) *gorm.DB {
		return db.Unscoped()
	}).Order("checked_at desc").Limit(200)

	// If admin, they see all logs.
	// If user, they only see logs for APIs matching their projects.
	if role != "admin" {
		query = query.Joins("JOIN apis ON apis.id = monitor_logs.api_id").
			Joins("JOIN projects ON projects.id = apis.project_id").
			Where("projects.user_id = ?", userID)
	}

	// Filter by project_id if provided
	projectID := c.Query("project_id")
	if projectID != "" {
		if role == "admin" {
			// Admin: need to join to filter by project
			query = query.Joins("JOIN apis a2 ON a2.id = monitor_logs.api_id").
				Where("a2.project_id = ?", projectID)
		} else {
			// Already joined: just add the filter
			query = query.Where("apis.project_id = ?", projectID)
		}
		// If project is filtered, we likely want more logs for graphs/status
		query = query.Limit(1000) 
	}

	// Date range filters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate != "" && endDate != "" {
		query = query.Where("checked_at BETWEEN ? AND ?", startDate+" 00:00:00", endDate+" 23:59:59")
		// For reports, we want all logs in the range
		query = query.Limit(5000)
	}

	if err := query.Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch logs"})
	}

	return c.JSON(logs)
}

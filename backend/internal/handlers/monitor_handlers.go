package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func GetMonitorLogs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var logs []models.MonitorLog

	query := database.DB.Order("checked_at desc").Limit(100)

	// If admin, they see all logs.
	// If user, they only see logs for APIs matching their projects.
	if role != "admin" {
		query = query.Joins("JOIN apis ON apis.id = monitor_logs.api_id").
			Joins("JOIN projects ON projects.id = apis.project_id").
			Where("projects.user_id = ?", userID)
	}

	if err := query.Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch logs"})
	}

	return c.JSON(logs)
}

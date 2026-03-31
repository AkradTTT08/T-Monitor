package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var notifications []models.DashboardNotification
	
	// Filter by current user_id OR system-wide notifications (user_id = 0)
	query := database.DB.Where("is_read = ? AND (user_id = ? OR user_id = 0)", false, userID)
	
	query.Order("created_at DESC").Limit(20).Find(&notifications)

	return c.JSON(notifications)
}

func MarkNotificationRead(c *fiber.Ctx) error {
	notificationID := c.Params("id")
	userID := c.Locals("user_id").(uint)
	
	if err := database.DB.Model(&models.DashboardNotification{}).
		Where("id = ? AND (user_id = ? OR user_id = 0)", notificationID, userID).
		Update("is_read", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark notification as read"})
	}

	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

func GetNotificationConfig(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	var config models.NotificationConfig
	if err := database.DB.Where("project_id = ?", projectID).First(&config).Error; err != nil {
		return c.JSON(models.NotificationConfig{ProjectID: 0}) // Return empty instead of error maybe? 
	}
	return c.JSON(config)
}

func UpsertNotificationConfig(c *fiber.Ctx) error {
	var input models.NotificationConfig
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var existing models.NotificationConfig
	result := database.DB.Where("project_id = ?", input.ProjectID).First(&existing)
	
	if result.RowsAffected > 0 {
		input.ID = existing.ID
		database.DB.Save(&input)
	} else {
		database.DB.Create(&input)
	}

	return c.JSON(input)
}

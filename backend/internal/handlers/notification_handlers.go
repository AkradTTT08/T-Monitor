package handlers

import (
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var notifications []models.DashboardNotification
	
	// Filter by current user_id OR system-wide notifications (user_id = uuid.Nil)
	query := database.DB.Where("is_read = ? AND (user_id = ? OR user_id = ?)", false, userID, uuid.Nil)
	
	query.Order("created_at DESC").Limit(20).Find(&notifications)

	return c.JSON(notifications)
}

func MarkNotificationRead(c *fiber.Ctx) error {
	notificationID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	
	if err := database.DB.Model(&models.DashboardNotification{}).
		Where("id = ? AND (user_id = ? OR user_id = ?)", notificationID, userID, uuid.Nil).
		Update("is_read", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark notification as read"})
	}

	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

func GetNotificationConfig(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	var config models.NotificationConfig
	if err := database.DB.Where("project_id = ?", projectID).First(&config).Error; err != nil {
		projectUUID, _ := uuid.Parse(projectID)
		return c.JSON(models.NotificationConfig{ProjectID: projectUUID}) // Return empty instead of error maybe? 
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

// CreateProjectNotification sends a dashboard notification to all project members
func CreateProjectNotification(projectID uuid.UUID, notifType string, title string, message string) {
	// 1. Find the project and its owner
	var project models.Project
	if err := database.DB.First(&project, "id = ?", projectID).Error; err != nil {
		return
	}

	// 2. Find all project members
	var members []models.ProjectMember
	database.DB.Where("project_id = ?", projectID).Find(&members)

	// 3. Collect unique user IDs
	userIDs := make(map[uuid.UUID]bool)
	userIDs[project.UserID] = true // Add owner
	for _, m := range members {
		userIDs[m.UserID] = true // Add member
	}

	// 4. Create notification records for each user
	for userID := range userIDs {
		notification := models.DashboardNotification{
			UserID:    userID,
			ProjectID: projectID,
			Type:      notifType,
			Title:     title,
			Message:   message,
		}
		database.DB.Create(&notification)
	}
}

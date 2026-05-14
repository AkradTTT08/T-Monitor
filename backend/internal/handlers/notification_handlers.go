package handlers

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

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

// MarkAllNotificationsRead marks ALL unread notifications for the current user as read in one DB call
func MarkAllNotificationsRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	result := database.DB.Model(&models.DashboardNotification{}).
		Where("is_read = ? AND (user_id = ? OR user_id = ?)", false, userID, uuid.Nil).
		Update("is_read", true)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark all notifications as read"})
	}

	return c.JSON(fiber.Map{"message": "All notifications marked as read", "count": result.RowsAffected})
}

func GetNotificationConfig(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	// Verify project ownership or membership
	var project models.Project
	if role != "admin" {
		if err := database.DB.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to view notification config"})
		}
	}

	var config models.NotificationConfig
	if err := database.DB.Where("project_id = ?", projectID).First(&config).Error; err != nil {
		projectUUID, _ := uuid.Parse(projectID)
		return c.JSON(models.NotificationConfig{ProjectID: projectUUID})
	}
	return c.JSON(config)
}

func UpsertNotificationConfig(c *fiber.Ctx) error {
	var input models.NotificationConfig
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	// Verify project ownership or membership
	var project models.Project
	if role != "admin" {
		if err := database.DB.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", input.ProjectID, userID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to update notification config"})
		}
	}

	// Build an explicit snake_case map so GORM updates all columns correctly,
	// including boolean fields that are false (zero value).
	updateMap := map[string]interface{}{
		"enable_telegram":   input.EnableTelegram,
		"telegram_bot_token": input.TelegramBotToken,
		"telegram_chat_id":  input.TelegramChatID,
		"enable_line":       input.EnableLINE,
		"line_user_id":      input.LINEUserID,
		"enable_email":      input.EnableEmail,
		"email_address":     input.EmailAddress,
		"smtp_host":         input.SmtpHost,
		"smtp_port":         input.SmtpPort,
		"smtp_user":         input.SmtpUser,
		"smtp_pass":         input.SmtpPass,
		"enable_webhook":    input.EnableWebhook,
		"webhook_url":       input.WebhookURL,
		"webhook_secret":    input.WebhookSecret,
		"enable_ticketing":  input.EnableTicketing,
	}

	var existing models.NotificationConfig
	findErr := database.DB.Where("project_id = ?", input.ProjectID).First(&existing).Error

	var result models.NotificationConfig

	if findErr == nil {
		// Record exists — update with explicit map (handles false booleans correctly)
		log.Printf("[NotifConfig] Updating config for project %s: enable_telegram=%v token_len=%d chat_id=%s",
			input.ProjectID, input.EnableTelegram, len(input.TelegramBotToken), input.TelegramChatID)
		if dbErr := database.DB.Session(&gorm.Session{}).Model(&existing).Updates(updateMap).Error; dbErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update notification config"})
		}
		// Reload the updated record to return fresh data
		database.DB.Where("project_id = ?", input.ProjectID).First(&result)
	} else {
		// No existing record — create new
		log.Printf("[NotifConfig] Creating new config for project %s: enable_telegram=%v", input.ProjectID, input.EnableTelegram)
		if dbErr := database.DB.Create(&input).Error; dbErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create notification config"})
		}
		result = input
	}

	log.Printf("[NotifConfig] Saved — project=%s enable_telegram=%v bot_token_len=%d chat_id=%s",
		result.ProjectID, result.EnableTelegram, len(result.TelegramBotToken), result.TelegramChatID)

	return c.JSON(result)
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

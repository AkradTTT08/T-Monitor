package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

type NotificationConfigInput struct {
	ProjectID       uint   `json:"project_id"`
	EnableTelegram  bool   `json:"enable_telegram"`
	TelegramChatID  string `json:"telegram_chat_id"`
	EnableLINE      bool   `json:"enable_line"`
	LINEUserID      string `json:"line_user_id"`
	EnableEmail     bool   `json:"enable_email"`
	EmailAddress    string `json:"email_address"`
	EnableTicketing bool   `json:"enable_ticketing"`
}

func GetNotificationConfig(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	var config models.NotificationConfig
	
	err := database.DB.Where("project_id = ?", projectID).First(&config).Error
	if err != nil {
		// Return 200 with null config instead of 404
		return c.JSON(fiber.Map{"config": nil})
	}
	
	return c.JSON(fiber.Map{"config": config})
}

func UpsertNotificationConfig(c *fiber.Ctx) error {
	var input NotificationConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var config models.NotificationConfig
	
	// Check if exists
	err := database.DB.Where("project_id = ?", input.ProjectID).First(&config).Error
	if err != nil {
		// Does not exist, create
		config = models.NotificationConfig{
			ProjectID:       input.ProjectID,
			EnableTelegram:  input.EnableTelegram,
			TelegramChatID:  input.TelegramChatID,
			EnableLINE:      input.EnableLINE,
			LINEUserID:      input.LINEUserID,
			EnableEmail:     input.EnableEmail,
			EmailAddress:    input.EmailAddress,
			EnableTicketing: input.EnableTicketing,
		}
		database.DB.Create(&config)
	} else {
		// Exists, update
		config.EnableTelegram = input.EnableTelegram
		config.TelegramChatID = input.TelegramChatID
		config.EnableLINE = input.EnableLINE
		config.LINEUserID = input.LINEUserID
		config.EnableEmail = input.EnableEmail
		config.EmailAddress = input.EmailAddress
		config.EnableTicketing = input.EnableTicketing
		database.DB.Save(&config)
	}

	return c.JSON(config)
}

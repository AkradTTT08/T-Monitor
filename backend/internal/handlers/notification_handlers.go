package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

type NotificationConfigInput struct {
	ProjectID        uint   `json:"project_id"`
	EnableTelegram   bool   `json:"enable_telegram"`
	TelegramBotToken string `json:"telegram_bot_token"`
	TelegramChatID   string `json:"telegram_chat_id"`
	EnableLINE       bool   `json:"enable_line"`
	LINEUserID       string `json:"line_user_id"`
	EnableEmail      bool   `json:"enable_email"`
	EmailAddress     string `json:"email_address"`
	SmtpHost         string `json:"smtp_host"`
	SmtpPort         int    `json:"smtp_port"`
	SmtpUser         string `json:"smtp_user"`
	SmtpPass         string `json:"smtp_pass"`
	EnableTicketing  bool   `json:"enable_ticketing"`
}

func GetNotificationConfig(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	var config models.NotificationConfig

	err := database.DB.Where("project_id = ?", projectID).Last(&config).Error
	if err != nil {
		return c.JSON(fiber.Map{"config": nil})
	}

	return c.JSON(fiber.Map{"config": config})
}

func UpsertNotificationConfig(c *fiber.Ctx) error {
	var input NotificationConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Delete all existing rows for this project (raw SQL avoids GORM zero-value guard)
	database.DB.Exec("DELETE FROM notification_configs WHERE project_id = ?", input.ProjectID)

	config := models.NotificationConfig{
		ProjectID:        input.ProjectID,
		EnableTelegram:   input.EnableTelegram,
		TelegramBotToken: input.TelegramBotToken,
		TelegramChatID:   input.TelegramChatID,
		EnableLINE:       input.EnableLINE,
		LINEUserID:       input.LINEUserID,
		EnableEmail:      input.EnableEmail,
		EmailAddress:     input.EmailAddress,
		SmtpHost:         input.SmtpHost,
		SmtpPort:         input.SmtpPort,
		SmtpUser:         input.SmtpUser,
		SmtpPass:         input.SmtpPass,
		EnableTicketing:  input.EnableTicketing,
	}

	if err := database.DB.Create(&config).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save config: " + err.Error()})
	}

	return c.JSON(config)
}

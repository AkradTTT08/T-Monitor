package handlers

import (
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/services"
)

func GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewNotificationService(nil)
	notifications, err := svc.GetNotifications(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(notifications)
}

func MarkNotificationRead(c *fiber.Ctx) error {
	notificationID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	
	svc := services.NewNotificationService(nil)
	if err := svc.MarkNotificationRead(notificationID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

// MarkAllNotificationsRead marks ALL unread notifications for the current user as read in one DB call
func MarkAllNotificationsRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewNotificationService(nil)
	count, err := svc.MarkAllNotificationsRead(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "All notifications marked as read", "count": count})
}

func GetNotificationConfig(c *fiber.Ctx) error {
	projectID := c.Params("projectId")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewNotificationService(nil)
	config, err := svc.GetNotificationConfig(projectID, userID, role)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
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

	svc := services.NewNotificationService(nil)
	config, err := svc.UpsertNotificationConfig(&input, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Unauthorized to access this project" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(config)
}

// CreateProjectNotification sends a dashboard notification to all project members
func CreateProjectNotification(projectID uuid.UUID, notifType string, title string, message string) {
	svc := services.NewNotificationService(nil)
	_ = svc.CreateProjectNotification(projectID, notifType, title, message)
}

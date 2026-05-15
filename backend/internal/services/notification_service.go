package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

type NotificationService interface {
	GetNotifications(userID uuid.UUID) ([]models.DashboardNotification, error)
	MarkNotificationRead(notificationID string, userID uuid.UUID) error
	MarkAllNotificationsRead(userID uuid.UUID) (int64, error)
	GetNotificationConfig(projectID string, userID uuid.UUID, role string) (*models.NotificationConfig, error)
	UpsertNotificationConfig(input *models.NotificationConfig, userID uuid.UUID, role string) (*models.NotificationConfig, error)
	CreateProjectNotification(projectID uuid.UUID, notifType, title, message string) error
}

type notificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) NotificationService {
	if db == nil {
		db = database.DB
	}
	return &notificationService{db: db}
}

func (s *notificationService) GetNotifications(userID uuid.UUID) ([]models.DashboardNotification, error) {
	var notifications []models.DashboardNotification
	
	// Filter by current user_id OR system-wide notifications (user_id = uuid.Nil)
	query := s.db.Where("is_read = ? AND (user_id = ? OR user_id = ?)", false, userID, uuid.Nil)
	
	if err := query.Order("created_at DESC").Limit(20).Find(&notifications).Error; err != nil {
		return nil, errors.New("Failed to fetch notifications")
	}

	return notifications, nil
}

func (s *notificationService) MarkNotificationRead(notificationID string, userID uuid.UUID) error {
	if err := s.db.Model(&models.DashboardNotification{}).
		Where("id = ? AND (user_id = ? OR user_id = ?)", notificationID, userID, uuid.Nil).
		Update("is_read", true).Error; err != nil {
		return errors.New("Failed to mark notification as read")
	}
	return nil
}

func (s *notificationService) MarkAllNotificationsRead(userID uuid.UUID) (int64, error) {
	result := s.db.Model(&models.DashboardNotification{}).
		Where("is_read = ? AND (user_id = ? OR user_id = ?)", false, userID, uuid.Nil).
		Update("is_read", true)

	if result.Error != nil {
		return 0, errors.New("Failed to mark all notifications as read")
	}

	return result.RowsAffected, nil
}

// verifyProjectAccess checks if the user has access to the project
func (s *notificationService) verifyProjectAccess(projectID string, userID uuid.UUID, role string) error {
	var project models.Project
	if role == "admin" {
		if err := s.db.First(&project, "id = ?", projectID).Error; err != nil {
			return errors.New("Project not found")
		}
	} else {
		if err := s.db.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return errors.New("Unauthorized to access this project")
		}
	}
	return nil
}

func (s *notificationService) GetNotificationConfig(projectID string, userID uuid.UUID, role string) (*models.NotificationConfig, error) {
	if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
		return nil, err
	}

	var config models.NotificationConfig
	if err := s.db.Where("project_id = ?", projectID).First(&config).Error; err != nil {
		projectUUID, _ := uuid.Parse(projectID)
		return &models.NotificationConfig{ProjectID: projectUUID}, nil
	}
	return &config, nil
}

func (s *notificationService) UpsertNotificationConfig(input *models.NotificationConfig, userID uuid.UUID, role string) (*models.NotificationConfig, error) {
	if err := s.verifyProjectAccess(input.ProjectID.String(), userID, role); err != nil {
		return nil, err
	}

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
	findErr := s.db.Where("project_id = ?", input.ProjectID).First(&existing).Error

	var result models.NotificationConfig

	if findErr == nil {
		// Record exists
		if dbErr := s.db.Session(&gorm.Session{}).Model(&existing).Updates(updateMap).Error; dbErr != nil {
			return nil, errors.New("Failed to update notification config")
		}
		s.db.Where("project_id = ?", input.ProjectID).First(&result)
	} else {
		// No existing record
		if dbErr := s.db.Create(input).Error; dbErr != nil {
			return nil, errors.New("Failed to create notification config")
		}
		result = *input
	}

	return &result, nil
}

func (s *notificationService) CreateProjectNotification(projectID uuid.UUID, notifType, title, message string) error {
	var project models.Project
	if err := s.db.First(&project, "id = ?", projectID).Error; err != nil {
		return errors.New("Project not found")
	}

	var members []models.ProjectMember
	s.db.Where("project_id = ?", projectID).Find(&members)

	var notifications []models.DashboardNotification
	
	// Create for owner
	notifications = append(notifications, models.DashboardNotification{
		UserID:  project.UserID,
		Type:    notifType,
		Title:   title,
		Message: message,
	})

	// Create for members
	for _, member := range members {
		notifications = append(notifications, models.DashboardNotification{
			UserID:  member.UserID,
			Type:    notifType,
			Title:   title,
			Message: message,
		})
	}

	if err := s.db.Create(&notifications).Error; err != nil {
		return errors.New("Failed to create notifications")
	}

	return nil
}

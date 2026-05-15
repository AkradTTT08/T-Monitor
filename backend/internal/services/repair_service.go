package services

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

type RepairService interface {
	GetRepairTasks(projectID string, userID uuid.UUID, role string) ([]models.RepairTask, error)
	ApproveRepairTask(taskID string, userID uuid.UUID, role string) (*models.RepairTask, error)
	CloseRepairTask(taskID string, reason, fixerName, docURL string, docs []string, userID uuid.UUID, role string) (*models.RepairTask, error)
	FailRepairTask(taskID string, description string, userID uuid.UUID, role string) (*models.RepairTask, error)
}

type repairService struct {
	db                  *gorm.DB
	notificationService NotificationService
}

func NewRepairService(db *gorm.DB, notifSvc NotificationService) RepairService {
	if db == nil {
		db = database.DB
	}
	if notifSvc == nil {
		notifSvc = NewNotificationService(db)
	}
	return &repairService{db: db, notificationService: notifSvc}
}

// verifyProjectAccess checks if the user has access to the project
func (s *repairService) verifyProjectAccess(projectID string, userID uuid.UUID, role string) error {
	if role == "admin" {
		return nil
	}

	var memberCount int64
	s.db.Model(&models.ProjectMember{}).
		Joins("JOIN projects ON projects.id = project_members.project_id").
		Where("projects.id = ? AND (projects.user_id = ? OR project_members.user_id = ?)", projectID, userID, userID).
		Count(&memberCount)
	if memberCount == 0 {
		var proj models.Project
		s.db.Where("id = ? AND user_id = ?", projectID, userID).First(&proj)
		if proj.ID == uuid.Nil {
			return errors.New("Unauthorized")
		}
	}
	return nil
}

func (s *repairService) GetRepairTasks(projectID string, userID uuid.UUID, role string) ([]models.RepairTask, error) {
	if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
		return nil, err
	}

	var tasks []models.RepairTask
	s.db.Preload("API").Preload("Approver").Where("project_id = ?", projectID).Order("created_at DESC").Find(&tasks)

	return tasks, nil
}

func (s *repairService) ApproveRepairTask(taskID string, userID uuid.UUID, role string) (*models.RepairTask, error) {
	var task models.RepairTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, errors.New("Task not found")
	}

	if err := s.verifyProjectAccess(task.ProjectID.String(), userID, role); err != nil {
		return nil, errors.New("Unauthorized to approve this task")
	}

	now := time.Now()
	task.Status = "pending"
	task.ApprovedBy = &userID
	task.ApprovedAt = &now

	if err := s.db.Save(&task).Error; err != nil {
		return nil, errors.New("Failed to approve task")
	}

	s.db.Preload("Approver").First(&task, "id = ?", task.ID)

	var project models.Project
	s.db.First(&project, "id = ?", task.ProjectID)
	_ = s.notificationService.CreateProjectNotification(
		task.ProjectID,
		"task_approve",
		"Repair Task Approved",
		"A repair task for project '"+project.Name+"' has been approved.",
	)

	return &task, nil
}

func (s *repairService) CloseRepairTask(taskID string, reason, fixerName, docURL string, docs []string, userID uuid.UUID, role string) (*models.RepairTask, error) {
	var task models.RepairTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, errors.New("Task not found")
	}

	if err := s.verifyProjectAccess(task.ProjectID.String(), userID, role); err != nil {
		return nil, errors.New("Unauthorized to close this task")
	}

	now := time.Now()
	task.Status = "closed"
	task.Reason = reason
	task.FixerName = fixerName
	task.DocumentURL = docURL
	task.ClosedAt = &now

	if len(docs) > 0 {
		docsJSON, err := json.Marshal(docs)
		if err == nil {
			task.Documents = string(docsJSON)
		}
	} else {
		task.Documents = "[]"
	}

	if err := s.db.Save(&task).Error; err != nil {
		return nil, errors.New("Failed to save task resolution")
	}

	var project models.Project
	s.db.First(&project, "id = ?", task.ProjectID)
	_ = s.notificationService.CreateProjectNotification(
		task.ProjectID,
		"task_close",
		"Repair Task Closed",
		"A repair task for project '"+project.Name+"' has been closed. Reason: "+reason,
	)

	return &task, nil
}

func (s *repairService) FailRepairTask(taskID string, description string, userID uuid.UUID, role string) (*models.RepairTask, error) {
	var task models.RepairTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, errors.New("Task not found")
	}

	if err := s.verifyProjectAccess(task.ProjectID.String(), userID, role); err != nil {
		return nil, errors.New("Unauthorized to mark this task as failed")
	}

	task.Status = "failed"
	task.Description = description

	if err := s.db.Save(&task).Error; err != nil {
		return nil, errors.New("Failed to mark task as failed")
	}

	var project models.Project
	s.db.First(&project, "id = ?", task.ProjectID)
	_ = s.notificationService.CreateProjectNotification(
		task.ProjectID,
		"task_fail",
		"Repair Task Failed",
		"A repair task for project '"+project.Name+"' has been marked as failed: "+description,
	)

	return &task, nil
}

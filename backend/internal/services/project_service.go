package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

type ProjectService interface {
	GetProjectsForUser(userID uuid.UUID, role string) ([]models.Project, error)
	GetProjectByID(projectID string, userID uuid.UUID, role string) (*models.Project, error)
	CreateProject(project *models.Project) error
	UpdateProject(projectID string, updateData map[string]interface{}, userID uuid.UUID, role string) (*models.Project, error)
	DeleteProject(projectID string, userID uuid.UUID, role string) error
	
	GetProjectMembers(projectID string) ([]models.ProjectMember, error)
	AddProjectMember(projectID string, targetUserID uuid.UUID, memberRole string, requestUserID uuid.UUID, requestUserRole string) (*models.ProjectMember, error)
	RemoveProjectMember(projectID string, targetUserID string, requestUserID uuid.UUID, requestUserRole string) error
}

type projectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) ProjectService {
	if db == nil {
		db = database.DB
	}
	return &projectService{db: db}
}

func (s *projectService) GetProjectsForUser(userID uuid.UUID, role string) ([]models.Project, error) {
	var projects []models.Project
	query := s.db.Preload("APIs")
	
	var err error
	if role == "admin" {
		err = query.Find(&projects).Error
	} else {
		err = query.Where("user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID, userID).Find(&projects).Error
	}
	if err != nil {
		return nil, err
	}

	s.preloadLatestLogs(&projects)

	return projects, nil
}

func (s *projectService) GetProjectByID(projectID string, userID uuid.UUID, role string) (*models.Project, error) {
	var project models.Project
	query := s.db.Preload("APIs")
	
	if role != "admin" {
		query = query.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID)
	} else {
		query = query.Where("id = ?", projectID)
	}

	if err := query.First(&project).Error; err != nil {
		return nil, errors.New("Project not found or unauthorized")
	}

	// Preload latest log for each API — wrap in slice so pointer mutation works
	projects := []models.Project{project}
	s.preloadLatestLogs(&projects)
	project.APIs = projects[0].APIs

	return &project, nil
}

// preloadLatestLogs fetches the latest MonitorLog per API and injects it back into the project list.
func (s *projectService) preloadLatestLogs(projects *[]models.Project) {
	if projects == nil || len(*projects) == 0 {
		return
	}

	// Collect all API IDs across all projects
	var apiIDs []interface{}
	for pi := range *projects {
		for _, a := range (*projects)[pi].APIs {
			apiIDs = append(apiIDs, a.ID)
		}
	}
	if len(apiIDs) == 0 {
		return
	}

	var latestLogs []models.MonitorLog
	s.db.Raw(`
		SELECT DISTINCT ON (api_id) *
		FROM monitor_logs
		WHERE api_id IN (?)
		  AND deleted_at IS NULL
		ORDER BY api_id, checked_at DESC
	`, apiIDs).Scan(&latestLogs)

	logMap := make(map[string]models.MonitorLog)
	for _, log := range latestLogs {
		logMap[log.ApiID.String()] = log
	}

	for pi := range *projects {
		for ai := range (*projects)[pi].APIs {
			if log, ok := logMap[(*projects)[pi].APIs[ai].ID.String()]; ok {
				(*projects)[pi].APIs[ai].Logs = []models.MonitorLog{log}
			}
		}
	}
}

func (s *projectService) CreateProject(project *models.Project) error {
	if err := s.db.Create(project).Error; err != nil {
		return err
	}

	// Create default notification config
	defaultConfig := models.NotificationConfig{
		ProjectID: project.ID,
	}
	return s.db.Create(&defaultConfig).Error
}

func (s *projectService) UpdateProject(projectID string, updateData map[string]interface{}, userID uuid.UUID, role string) (*models.Project, error) {
	var project models.Project
	
	query := s.db
	if role != "admin" {
		query = query.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID)
	} else {
		query = query.Where("id = ?", projectID)
	}

	if err := query.First(&project).Error; err != nil {
		return nil, errors.New("Unauthorized to update this project or project not found")
	}

	if err := s.db.Model(&project).Updates(updateData).Error; err != nil {
		return nil, errors.New("Failed to update project")
	}

	return &project, nil
}

func (s *projectService) DeleteProject(projectID string, userID uuid.UUID, role string) error {
	var project models.Project
	query := s.db
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, "id = ?", projectID).Error; err != nil {
		return errors.New("Project not found or unauthorized")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("api_id IN (SELECT id FROM apis WHERE project_id = ?)", project.ID).Delete(&models.MonitorLog{})
		tx.Where("project_id = ?", project.ID).Delete(&models.API{})
		tx.Where("project_id = ?", project.ID).Delete(&models.ProjectMember{})
		tx.Where("project_id = ?", project.ID).Delete(&models.NotificationConfig{})
		return tx.Delete(&project).Error
	})
}

func (s *projectService) GetProjectMembers(projectID string) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := s.db.Preload("User").Where("project_id = ?", projectID).Find(&members).Error
	return members, err
}

func (s *projectService) AddProjectMember(projectID string, targetUserID uuid.UUID, memberRole string, requestUserID uuid.UUID, requestUserRole string) (*models.ProjectMember, error) {
	var project models.Project
	
	if requestUserRole != "admin" {
		if err := s.db.Where("id = ? AND user_id = ?", projectID, requestUserID).First(&project).Error; err != nil {
			return nil, errors.New("Only project owners or admins can manage members")
		}
	} else {
		if err := s.db.First(&project, "id = ?", projectID).Error; err != nil {
			return nil, errors.New("Project not found")
		}
	}

	member := models.ProjectMember{
		ProjectID: project.ID,
		UserID:    targetUserID,
		Role:      memberRole,
	}

	if err := s.db.Create(&member).Error; err != nil {
		return nil, errors.New("Failed to add project member (Likely already exists)")
	}

	return &member, nil
}

func (s *projectService) RemoveProjectMember(projectID string, targetUserID string, requestUserID uuid.UUID, requestUserRole string) error {
	if requestUserRole != "admin" {
		var project models.Project
		if err := s.db.Where("id = ? AND user_id = ?", projectID, requestUserID).First(&project).Error; err != nil {
			return errors.New("Only project owners or admins can manage members")
		}
	}

	if err := s.db.Where("project_id = ? AND user_id = ?", projectID, targetUserID).Delete(&models.ProjectMember{}).Error; err != nil {
		return errors.New("Failed to remove project member")
	}

	return nil
}

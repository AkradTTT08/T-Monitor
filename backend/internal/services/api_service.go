package services

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/utils"
	"gorm.io/gorm"
)

type APIService interface {
	CreateAPI(api *models.API, mode string, userID uuid.UUID, role string) error
	GetAPIs(projectID string, search string, page int, limit int, userID uuid.UUID, role string) ([]models.API, int64, error)
	GetAPIByID(apiID string, userID uuid.UUID, role string) (*models.API, error)
	UpdateAPI(apiID string, updateData map[string]interface{}, userID uuid.UUID, role string) (*models.API, error)
	DeleteAPI(apiID string, userID uuid.UUID, role string) error
	PauseAPI(apiID string, durationMinutes int, userID uuid.UUID, role string) error
	ReorderAPIs(projectID string, items []struct{ID uuid.UUID; Folder string; OrderIndex int}, userID uuid.UUID, role string) error
	ImportPostmanCollection(projectID string, mode string, file io.Reader, userID uuid.UUID, role string) (int, error)
}

type apiService struct {
	db *gorm.DB
}

func NewAPIService(db *gorm.DB) APIService {
	if db == nil {
		db = database.DB
	}
	return &apiService{db: db}
}

// verifyProjectAccess checks if the user has access to the project
func (s *apiService) verifyProjectAccess(projectID string, userID uuid.UUID, role string) error {
	var project models.Project
	if role == "admin" {
		if err := s.db.First(&project, "id = ?", projectID).Error; err != nil {
			return errors.New("Project not found")
		}
	} else {
		if err := s.db.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return errors.New("Project not found or unauthorized")
		}
	}
	return nil
}

// verifyAPIAccess checks if the user has access to the API via its project
func (s *apiService) verifyAPIAccess(apiID string, userID uuid.UUID, role string) (*models.API, error) {
	var api models.API
	query := s.db.Model(&models.API{})
	if role != "admin" {
		query = query.Where("id = ? AND project_id IN (SELECT id FROM projects WHERE user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", apiID, userID, userID)
	} else {
		query = query.Where("id = ?", apiID)
	}

	if err := query.First(&api).Error; err != nil {
		return nil, errors.New("API not found or unauthorized")
	}
	return &api, nil
}

func (s *apiService) CreateAPI(api *models.API, mode string, userID uuid.UUID, role string) error {
	if err := s.verifyProjectAccess(api.ProjectID.String(), userID, role); err != nil {
		return err
	}

	if mode == "replace" {
		if err := s.db.Where("project_id = ?", api.ProjectID).Delete(&models.API{}).Error; err != nil {
			return errors.New("Failed to clear existing APIs")
		}
	}

	// New APIs start active (no PausedUntil set)
	if err := s.db.Create(api).Error; err != nil {
		return errors.New("Failed to create API endpoint")
	}

	return nil
}

func (s *apiService) GetAPIs(projectID string, search string, page int, limit int, userID uuid.UUID, role string) ([]models.API, int64, error) {
	if projectID != "" {
		if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
			return nil, 0, err
		}
	}

	var apis []models.API
	var total int64
	query := s.db.Model(&models.API{})

	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	} else if role != "admin" {
		query = query.Where("project_id IN (SELECT id FROM projects WHERE user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", userID, userID)
	}

	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	query.Count(&total)

	offset := (page - 1) * limit
	if err := query.
		Order("order_index ASC").Order("created_at DESC").
		Offset(offset).Limit(limit).Find(&apis).Error; err != nil {
		return nil, 0, errors.New("Failed to fetch APIs")
	}

	// Manually load latest log per API (GORM Preload with Limit(1) applies globally, not per record)
	if len(apis) > 0 {
		apiIDs := make([]interface{}, len(apis))
		for i, a := range apis {
			apiIDs[i] = a.ID
		}
		var latestLogs []models.MonitorLog
		s.db.Raw(`
			SELECT DISTINCT ON (api_id) *
			FROM monitor_logs
			WHERE api_id IN (?)
			  AND deleted_at IS NULL
			ORDER BY api_id, checked_at DESC
		`, apiIDs).Scan(&latestLogs)

		// Map logs back to APIs
		logMap := make(map[string]models.MonitorLog)
		for _, log := range latestLogs {
			logMap[log.ApiID.String()] = log
		}
		for i, a := range apis {
			if log, ok := logMap[a.ID.String()]; ok {
				apis[i].Logs = []models.MonitorLog{log}
			}
		}
	}

	return apis, total, nil
}

func (s *apiService) GetAPIByID(apiID string, userID uuid.UUID, role string) (*models.API, error) {
	return s.verifyAPIAccess(apiID, userID, role)
}

func (s *apiService) UpdateAPI(apiID string, updateData map[string]interface{}, userID uuid.UUID, role string) (*models.API, error) {
	api, err := s.verifyAPIAccess(apiID, userID, role)
	if err != nil {
		return nil, err
	}

	if len(updateData) > 0 {
		if err := s.db.Model(api).Updates(updateData).Error; err != nil {
			return nil, errors.New("Failed to update API")
		}
	}

	return api, nil
}

func (s *apiService) DeleteAPI(apiID string, userID uuid.UUID, role string) error {
	api, err := s.verifyAPIAccess(apiID, userID, role)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("api_id = ?", api.ID).Delete(&models.MonitorLog{})
		return tx.Delete(api).Error
	})
}

func (s *apiService) PauseAPI(apiID string, durationMinutes int, userID uuid.UUID, role string) error {
	api, err := s.verifyAPIAccess(apiID, userID, role)
	if err != nil {
		return err
	}

	var pausedUntil *time.Time
	if durationMinutes > 0 {
		// Pause for specific duration
		t := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
		pausedUntil = &t
	} else if durationMinutes == -1 {
		// Pause indefinitely
		t := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		pausedUntil = &t
	} else {
		// durationMinutes == 0 → Resume (set to past time)
		t := time.Now().Add(-1 * time.Second)
		pausedUntil = &t
	}

	if err := s.db.Model(api).Update("paused_until", pausedUntil).Error; err != nil {
		return errors.New("Failed to update pause status")
	}

	return nil
}

func (s *apiService) ReorderAPIs(projectID string, items []struct{ID uuid.UUID; Folder string; OrderIndex int}, userID uuid.UUID, role string) error {
	if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
		return err
	}

	tx := s.db.Begin()

	for _, item := range items {
		if err := tx.Model(&models.API{}).Where("id = ? AND project_id = ?", item.ID, projectID).
			Updates(map[string]interface{}{
				"folder":      item.Folder,
				"order_index": item.OrderIndex,
			}).Error; err != nil {
			tx.Rollback()
			return errors.New("Failed to reorder APIs")
		}
	}

	tx.Commit()
	return nil
}

func (s *apiService) ImportPostmanCollection(projectID string, mode string, file io.Reader, userID uuid.UUID, role string) (int, error) {
	if err := s.verifyProjectAccess(projectID, userID, role); err != nil {
		return 0, err
	}

	projectUUID, _ := uuid.Parse(projectID)

	parsedAPIs, envMap, err := utils.ParsePostmanCollection(file, projectUUID)
	if err != nil {
		return 0, errors.New("Invalid Postman JSON structure")
	}

	if len(parsedAPIs) > 0 {
		if mode == "replace" {
			if err := s.db.Unscoped().Where("api_id IN (SELECT id FROM apis WHERE project_id = ?)", projectID).Delete(&models.MonitorLog{}).Error; err != nil {
				return 0, errors.New("Failed to clear existing monitor logs")
			}
			if err := s.db.Unscoped().Where("project_id = ?", projectID).Delete(&models.API{}).Error; err != nil {
				return 0, errors.New("Failed to clear existing APIs")
			}
		}

		if err := s.db.Create(&parsedAPIs).Error; err != nil {
			return 0, errors.New("Failed to save APIs to DB")
		}
	}

	// Update Project Environment Variables
	if len(envMap) > 0 {
		var project models.Project
		if err := s.db.First(&project, "id = ?", projectID).Error; err == nil {
			existingEnvMap := make(map[string]string)
			
			if mode == "append" && project.EnvironmentVariables != "" {
				json.Unmarshal([]byte(project.EnvironmentVariables), &existingEnvMap)
			}
			
			for k, v := range envMap {
				existingEnvMap[k] = v
			}
			
			envBytes, _ := json.Marshal(existingEnvMap)
			project.EnvironmentVariables = string(envBytes)
			s.db.Save(&project)
		}
	}

	return len(parsedAPIs), nil
}

package handlers

import (
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

// GetPublicProjectStatus returns status data for a project (Unauthenticated)
func GetPublicProjectStatus(c *fiber.Ctx) error {
	idStr := c.Params("id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid project ID format"})
	}

	var project models.Project
	if err := database.DB.Preload("APIs").Preload("Company").First(&project, "id = ?", projectID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	type PublicAPIInfo struct {
		ID            uuid.UUID `json:"id"`
		Name          string    `json:"name"`
		Folder        string    `json:"folder"`
		Status        string    `json:"status"` // "UP" or "DOWN"
		UptimePercent float64   `json:"uptime_percent"`
		LastChecked   time.Time `json:"last_checked"`
	}

	since := time.Now().AddDate(0, 0, -7) // Last 7 days for public stats
	var publicAPIs []PublicAPIInfo

	for _, api := range project.APIs {
		if !api.IsActive {
			continue
		}

		// Get latest status
		var lastLog models.MonitorLog
		status := "UNKNOWN"
		if err := database.DB.Where("api_id = ?", api.ID).Order("checked_at desc").First(&lastLog).Error; err == nil {
			if lastLog.IsSuccess {
				status = "UP"
			} else {
				status = "DOWN"
			}
		}

		// Calculate 7d Uptime
		var total, success int64
		database.DB.Model(&models.MonitorLog{}).Where("api_id = ? AND checked_at >= ?", api.ID, since).Count(&total)
		database.DB.Model(&models.MonitorLog{}).Where("api_id = ? AND checked_at >= ? AND is_success = true", api.ID, since).Count(&success)

		uptime := 100.0
		if total > 0 {
			uptime = math.Round((float64(success)/float64(total))*10000) / 100
		}

		publicAPIs = append(publicAPIs, PublicAPIInfo{
			ID:            api.ID,
			Name:          api.Name,
			Folder:        api.Folder,
			Status:        status,
			UptimePercent: uptime,
			LastChecked:   lastLog.CheckedAt,
		})
	}

	// Historical Daily Uptime (Last 7 days)
	type DayStat struct {
		Day           string  `json:"day"`
		UptimePercent float64 `json:"uptime_percent"`
	}
	var history []DayStat
	for i := 6; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i)
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		dayEnd := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.Local)

		var total, success int64
		database.DB.Model(&models.MonitorLog{}).
			Joins("JOIN apis ON apis.id = monitor_logs.api_id").
			Where("apis.project_id = ? AND checked_at BETWEEN ? AND ?", projectID, dayStart, dayEnd).
			Count(&total)
		
		database.DB.Model(&models.MonitorLog{}).
			Joins("JOIN apis ON apis.id = monitor_logs.api_id").
			Where("apis.project_id = ? AND checked_at BETWEEN ? AND ? AND is_success = true", projectID, dayStart, dayEnd).
			Count(&success)

		uptime := 100.0
		if total > 0 {
			uptime = math.Round((float64(success)/float64(total))*10000) / 100
		}

		history = append(history, DayStat{
			Day:           d.Format("2006-01-02"),
			UptimePercent: uptime,
		})
	}

	return c.JSON(fiber.Map{
		"project_name": project.Name,
		"description":  project.Description,
		"company":      project.Company,
		"apis":         publicAPIs,
		"history":      history,
	})
}

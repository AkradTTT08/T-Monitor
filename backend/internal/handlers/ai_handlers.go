package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/services"
)

type AIQueryRequest struct {
	Query   string                 `json:"query"`
	History []services.ChatMessage `json:"history"`
}

type AIQueryResponse struct {
	Answer string `json:"answer"`
}

// ChatWithAI is an alias for QueryData, kept for backward compatibility with route registration
var ChatWithAI = QueryData

// AnalyzeIncident provides an AI-generated analysis of an API failure
func AnalyzeIncident(c *fiber.Ctx) error {
	incidentID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	// Fetch incident details
	var logEntry models.MonitorLog
	if err := database.DB.Preload("API").First(&logEntry, "id = ?", incidentID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Incident not found"})
	}

	// Verify project access
	projectID := logEntry.API.ProjectID
	if role != "admin" {
		var project models.Project
		if err := database.DB.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
		}
	}

	// Determine if we should run in mock mode
	isTest := c.Get("Test-Mock-AI") == "true"
	svc := services.NewAIService(isTest)

	response, err := svc.AnalyzeIncident(logEntry.ErrorMessage, logEntry.ResponseBody, logEntry.API.URL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"analysis": response})
}

// QueryData provides an AI chat interface to query analytics data
func QueryData(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	projectID := c.Query("project_id")

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	// Verify project access
	if role != "admin" {
		var project models.Project
		if err := database.DB.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
		}
	}

	var req AIQueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	// Gather project context
	var apis []models.API
	database.DB.Where("project_id = ?", projectID).Find(&apis)

	apiCount := len(apis)
	var totalChecks, failedChecks int64
	
	// Fast context gathering
	if apiCount > 0 {
		apiIDs := make([]uuid.UUID, apiCount)
		for i, a := range apis {
			apiIDs[i] = a.ID
		}
		
		isSQLite := database.DB.Dialector.Name() == "sqlite"
		isSuccessVal := interface{}(false)
		if isSQLite {
			isSuccessVal = 0
		}

		database.DB.Model(&models.MonitorLog{}).Where("api_id IN ?", apiIDs).Count(&totalChecks)
		database.DB.Model(&models.MonitorLog{}).Where("api_id IN ? AND is_success = ?", apiIDs, isSuccessVal).Count(&failedChecks)
	}

	// Simplify the context to pass to the AI
	contextData := map[string]interface{}{
		"project_id":    projectID,
		"api_count":     apiCount,
		"total_checks":  totalChecks,
		"failed_checks": failedChecks,
	}

	contextJSON, _ := json.Marshal(contextData)

	isTest := c.Get("Test-Mock-AI") == "true"
	svc := services.NewAIService(isTest)

	response, err := svc.QueryData(req.Query, req.History, string(contextJSON))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(AIQueryResponse{Answer: response})
}

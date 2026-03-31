package handlers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func GetRepairTasks(c *fiber.Ctx) error {
	projectID := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	// Verify project ownership
	var project models.Project
	if role != "admin" {
		if err := database.DB.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
		}
	}

	var tasks []models.RepairTask
	database.DB.Preload("API").Where("project_id = ?", projectID).Order("created_at DESC").Find(&tasks)

	return c.JSON(tasks)
}

func ApproveRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var task models.RepairTask
	if err := database.DB.First(&task, taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	now := time.Now()
	task.Status = "pending"
	task.ApprovedBy = &userID
	task.ApprovedAt = &now

	database.DB.Save(&task)

	// Create Dashboard Notification
	var project models.Project
	database.DB.First(&project, task.ProjectID)

	notification := models.DashboardNotification{
		UserID:    project.UserID,
		ProjectID: task.ProjectID,
		Type:      "task_approve",
		Title:     "Repair Task Approved",
		Message:   "A repair task for project '" + project.Name + "' has been approved.",
	}
	database.DB.Create(&notification)

	return c.JSON(task)
}

func CloseRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	
	type CloseInput struct {
		Reason      string   `json:"reason"`
		DocumentURL string   `json:"document_url"`
		Documents   []string `json:"documents"`
	}
	var input CloseInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var task models.RepairTask
	if err := database.DB.First(&task, taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	now := time.Now()
	task.Status = "closed"
	task.Reason = input.Reason
	task.DocumentURL = input.DocumentURL
	
	if len(input.Documents) > 0 {
		docsJSON, _ := json.Marshal(input.Documents)
		task.Documents = string(docsJSON)
	}
	
	task.ClosedAt = &now

	database.DB.Save(&task)

	// Create Dashboard Notification
	var project models.Project
	database.DB.First(&project, task.ProjectID)

	notification := models.DashboardNotification{
		UserID:    project.UserID,
		ProjectID: task.ProjectID,
		Type:      "task_close",
		Title:     "Repair Task Closed",
		Message:   "A repair task for project '" + project.Name + "' has been closed. Reason: " + input.Reason,
	}
	database.DB.Create(&notification)

	return c.JSON(task)
}

func FailRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	
	type FailInput struct {
		Description string `json:"description"`
	}
	var input FailInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var task models.RepairTask
	if err := database.DB.First(&task, taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	task.Status = "failed"
	task.Description = input.Description

	database.DB.Save(&task)

	// Create Dashboard Notification
	var project models.Project
	database.DB.First(&project, task.ProjectID)

	notification := models.DashboardNotification{
		UserID:    project.UserID,
		ProjectID: task.ProjectID,
		Type:      "task_fail",
		Title:     "Repair Task Failed",
		Message:   "A repair task for project '" + project.Name + "' has been marked as failed: " + input.Description,
	}
	database.DB.Create(&notification)

	return c.JSON(task)
}

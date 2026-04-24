package handlers

import (
	"github.com/google/uuid"

	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func GetRepairTasks(c *fiber.Ctx) error {
	projectID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	// Verify project ownership or membership
	var project models.Project
	if role != "admin" {
		if err := database.DB.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", projectID, userID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized"})
		}
	}

	var tasks []models.RepairTask
	database.DB.Preload("API").Preload("Approver").Where("project_id = ?", projectID).Order("created_at DESC").Find(&tasks)

	return c.JSON(tasks)
}

func ApproveRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	var task models.RepairTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	role := c.Locals("role").(string)
	// Verify project ownership or membership
	if role != "admin" {
		var memberCount int64
		database.DB.Model(&models.ProjectMember{}).
			Joins("JOIN projects ON projects.id = project_members.project_id").
			Where("projects.id = ? AND (projects.user_id = ? OR project_members.user_id = ?)", task.ProjectID, userID, userID).
			Count(&memberCount)
		if memberCount == 0 {
			// Double check if user is the owner (in case they are not in project_members but are the creator)
			var proj models.Project
			database.DB.Where("id = ? AND user_id = ?", task.ProjectID, userID).First(&proj)
			if proj.ID == uuid.Nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to approve this task"})
			}
		}
	}

	now := time.Now()
	task.Status = "pending"
	task.ApprovedBy = &userID
	task.ApprovedAt = &now

	database.DB.Save(&task)
	database.DB.Preload("Approver").First(&task, "id = ?", task.ID)

	// Create Dashboard Notification
	var project models.Project
	database.DB.First(&project, "id = ?", task.ProjectID)
	CreateProjectNotification(
		task.ProjectID,
		"task_approve",
		"Repair Task Approved",
		"A repair task for project '" + project.Name + "' has been approved.",
	)

	return c.JSON(task)
}

func CloseRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	
	type CloseInput struct {
		Reason      string   `json:"reason"`
		DocumentURL string   `json:"document_url"`
		Documents   []string `json:"documents"`
		FixerName   string   `json:"fixer_name"`
	}
	var input CloseInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var task models.RepairTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	// Verify project ownership or membership
	if role != "admin" {
		var memberCount int64
		database.DB.Model(&models.ProjectMember{}).
			Joins("JOIN projects ON projects.id = project_members.project_id").
			Where("projects.id = ? AND (projects.user_id = ? OR project_members.user_id = ?)", task.ProjectID, userID, userID).
			Count(&memberCount)
		if memberCount == 0 {
			var proj models.Project
			database.DB.Where("id = ? AND user_id = ?", task.ProjectID, userID).First(&proj)
			if proj.ID == uuid.Nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to close this task"})
			}
		}
	}

	now := time.Now()
	task.Status = "closed"
	task.Reason = input.Reason
	task.FixerName = input.FixerName
	task.DocumentURL = input.DocumentURL
	
	task.ClosedAt = &now
	
	if len(input.Documents) > 0 {
		docsJSON, err := json.Marshal(input.Documents)
		if err == nil {
			task.Documents = string(docsJSON)
		}
	} else {
		task.Documents = "[]"
	}

	if err := database.DB.Save(&task).Error; err != nil {
		fmt.Printf("❌ Failed to close task: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save task resolution"})
	}

	// Create Dashboard Notification
	var project models.Project
	database.DB.First(&project, "id = ?", task.ProjectID)
	CreateProjectNotification(
		task.ProjectID,
		"task_close",
		"Repair Task Closed",
		"A repair task for project '" + project.Name + "' has been closed. Reason: " + input.Reason,
	)

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
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	// Verify project ownership or membership
	if role != "admin" {
		var memberCount int64
		database.DB.Model(&models.ProjectMember{}).
			Joins("JOIN projects ON projects.id = project_members.project_id").
			Where("projects.id = ? AND (projects.user_id = ? OR project_members.user_id = ?)", task.ProjectID, userID, userID).
			Count(&memberCount)
		if memberCount == 0 {
			var proj models.Project
			database.DB.Where("id = ? AND user_id = ?", task.ProjectID, userID).First(&proj)
			if proj.ID == uuid.Nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to mark this task as failed"})
			}
		}
	}

	task.Status = "failed"
	task.Description = input.Description

	database.DB.Save(&task)

	// Create Dashboard Notification
	var project models.Project
	database.DB.First(&project, "id = ?", task.ProjectID)
	CreateProjectNotification(
		task.ProjectID,
		"task_fail",
		"Repair Task Failed",
		"A repair task for project '" + project.Name + "' has been marked as failed: " + input.Description,
	)

	return c.JSON(task)
}

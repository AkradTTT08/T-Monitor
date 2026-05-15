package handlers

import (
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/services"
)

func GetRepairTasks(c *fiber.Ctx) error {
	projectID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewRepairService(nil, nil)
	tasks, err := svc.GetRepairTasks(projectID, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Unauthorized" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tasks)
}

func ApproveRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewRepairService(nil, nil)
	task, err := svc.ApproveRepairTask(taskID, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Task not found" {
			status = fiber.StatusNotFound
		} else if err.Error() == "Unauthorized to approve this task" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(task)
}

func CloseRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	
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

	svc := services.NewRepairService(nil, nil)
	task, err := svc.CloseRepairTask(taskID, input.Reason, input.FixerName, input.DocumentURL, input.Documents, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Task not found" {
			status = fiber.StatusNotFound
		} else if err.Error() == "Unauthorized to close this task" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(task)
}

func FailRepairTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	
	type FailInput struct {
		Description string `json:"description"`
	}
	var input FailInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	svc := services.NewRepairService(nil, nil)
	task, err := svc.FailRepairTask(taskID, input.Description, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Task not found" {
			status = fiber.StatusNotFound
		} else if err.Error() == "Unauthorized to mark this task as failed" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(task)
}

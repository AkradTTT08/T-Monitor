package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

type ProjectInput struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	EnvironmentVariables string `json:"environment_variables"`
}

func CreateProject(c *fiber.Ctx) error {
	var input ProjectInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	userID := c.Locals("user_id").(uint)

	// If not provided, default to empty JSON object
	if input.EnvironmentVariables == "" {
		input.EnvironmentVariables = "{}"
	}

	project := models.Project{
		Name:                 input.Name,
		Description:          input.Description,
		EnvironmentVariables: input.EnvironmentVariables,
		UserID:               userID,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project"})
	}

	return c.Status(fiber.StatusCreated).JSON(project)
}

func GetProjects(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var projects []models.Project
	
	// Admins can see all projects; users see only theirs
	if role == "admin" {
		database.DB.Preload("APIs").Find(&projects)
	} else {
		database.DB.Where("user_id = ?", userID).Preload("APIs").Find(&projects)
	}

	return c.JSON(projects)
}

func GetProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var project models.Project

	query := database.DB.Preload("APIs")
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}

	return c.JSON(project)
}

func UpdateProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var input ProjectInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var project models.Project
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}

	// If not provided, default to empty JSON object
	if input.EnvironmentVariables == "" {
		input.EnvironmentVariables = "{}"
	}

	project.Name = input.Name
	project.Description = input.Description
	project.EnvironmentVariables = input.EnvironmentVariables

	database.DB.Save(&project)

	return c.JSON(project)
}

func DeleteProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var project models.Project
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}

	// Delete child API logs first (requires joined delete or subquery, but easier to use a subquery)
	database.DB.Exec("DELETE FROM monitor_logs WHERE api_id IN (SELECT id FROM apis WHERE project_id = ?)", project.ID)
	
	// Delete child Records to satisfy foreign key constraints
	database.DB.Unscoped().Where("project_id = ?", project.ID).Delete(&models.API{})
	database.DB.Unscoped().Where("project_id = ?", project.ID).Delete(&models.NotificationConfig{})

	database.DB.Unscoped().Delete(&project)

	return c.JSON(fiber.Map{"message": "Project deleted successfully"})
}

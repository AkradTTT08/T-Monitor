package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

type ProjectInput struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	EnvironmentVariables string `json:"environment_variables"`
	CoverImageURL        string `json:"cover_image_url"`
	CoverPosition        int    `json:"cover_position"`
	CompanyID            *uint  `json:"company_id"`
}

func UploadProjectCover(c *fiber.Ctx) error {
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

	file, err := c.FormFile("cover")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

	// Get absolute path for upload directory
	uploadDir, err := filepath.Abs("./uploads/projects")
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error: path resolution failed"})
	}

	// Create directory if not exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		fmt.Printf("Creating uploads directory: %s\n", uploadDir)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload directory"})
		}
	}

	// Generate filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("project_%d_%d%s", project.ID, time.Now().Unix(), ext)
	savePath := filepath.Join(uploadDir, filename)
	fmt.Printf("Attempting to save project cover: ProjectID=%d, Filename=%s, FullPath=%s\n", project.ID, filename, savePath)

	if err := c.SaveFile(file, savePath); err != nil {
		fmt.Printf("❌ Failed to save file for Project ID %d: %v\n", project.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
	}

	fmt.Printf("✅ File saved successfully: %s\n", savePath)

	// Update DB
	project.CoverImageURL = "/uploads/projects/" + filename
	// Default position 50 if not set, but we usually want to keep it if it exists
	database.DB.Save(&project)

	return c.JSON(fiber.Map{
		"message":         "Cover image uploaded successfully",
		"cover_image_url": project.CoverImageURL,
	})
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
		CoverPosition:        input.CoverPosition,
		UserID:               userID,
		CompanyID:            input.CompanyID,
	}

	if c.Locals("is_dry_run") == true {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "DRY_RUN: Project validation successful. Data not persisted.",
			"data":    project,
		})
	}

	if err := database.DB.Create(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project"})
	}

	// Create default notification config
	defaultConfig := models.NotificationConfig{
		ProjectID: project.ID,
	}
	database.DB.Create(&defaultConfig)

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
	project.CoverPosition = input.CoverPosition
	project.CompanyID = input.CompanyID

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

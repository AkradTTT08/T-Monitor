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
	fmt.Println(">>> Starting UploadProjectCover")
	id := c.Params("id")
	fmt.Printf(">>> ID: %s\n", id)
	
	rawUserID := c.Locals("user_id")
	fmt.Printf(">>> Raw UserID: %v\n", rawUserID)
	userID := rawUserID.(uint)
	
	rawRole := c.Locals("role")
	fmt.Printf(">>> Raw Role: %v\n", rawRole)
	role := rawRole.(string)

	var project models.Project
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, id).Error; err != nil {
		fmt.Printf(">>> Project not found: %v\n", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}
	fmt.Println(">>> Project loaded")

	file, err := c.FormFile("cover")
	if err != nil {
		fmt.Printf(">>> FormFile error: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}
	fmt.Printf(">>> File received: %s, Size: %d\n", file.Filename, file.Size)

	// Get absolute path for upload directory
	uploadDir, err := filepath.Abs("./uploads/projects")
	if err != nil {
		fmt.Printf(">>> filepath.Abs error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error: path resolution failed"})
	}
	fmt.Printf(">>> Upload dir: %s\n", uploadDir)

	// Create directory if not exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		fmt.Printf(">>> Creating directory: %s\n", uploadDir)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fmt.Printf(">>> MkdirAll error: %v\n", err)
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

	// Update DB - only the cover_image_url field to be safe
	project.CoverImageURL = "/uploads/projects/" + filename
	if err := database.DB.Model(&project).Update("cover_image_url", project.CoverImageURL).Error; err != nil {
		fmt.Printf("❌ Failed to update database for Project ID %d: %v\n", project.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update database"})
	}

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
		database.DB.Preload("APIs").
			Where("user_id = ? OR company_id IN (SELECT company_id FROM company_members WHERE user_id = ?)", userID, userID).
			Find(&projects)
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
		query = query.Where("user_id = ? OR company_id IN (SELECT company_id FROM company_members WHERE user_id = ?)", userID, userID)
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
	// Use Updates with a map to only update specific fields, avoiding overwriting LogoURL/CoverImageURL
	updateData := map[string]interface{}{
		"name":                  input.Name,
		"description":           input.Description,
		"environment_variables": input.EnvironmentVariables,
		"cover_position":        input.CoverPosition,
		"company_id":            input.CompanyID,
	}

	if err := database.DB.Model(&project).Updates(updateData).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update project"})
	}

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

	// Soft-delete child API logs
	database.DB.Where("api_id IN (SELECT id FROM apis WHERE project_id = ?)", project.ID).Delete(&models.MonitorLog{})

	// Soft-delete child Records
	database.DB.Where("project_id = ?", project.ID).Delete(&models.API{})
	// NotificationConfig doesn't have Soft Delete yet, but let's be consistent or add it later. 
	// For now, keep hard delete for configs if they don't have DeletedAt, or just regular Delete.
	database.DB.Where("project_id = ?", project.ID).Delete(&models.NotificationConfig{})

	database.DB.Delete(&project)

	return c.JSON(fiber.Map{"message": "Project deleted successfully"})
}

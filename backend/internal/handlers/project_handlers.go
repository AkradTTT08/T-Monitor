package handlers

import (
	"github.com/google/uuid"

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
	CompanyID            *uuid.UUID  `json:"company_id"`
}

func UploadProjectCover(c *fiber.Ctx) error {
	fmt.Println(">>> Starting UploadProjectCover")
	id := c.Params("id")
	fmt.Printf(">>> ID: %s\n", id)
	
	rawUserID := c.Locals("user_id")
	fmt.Printf(">>> Raw UserID: %v\n", rawUserID)
	userID := rawUserID.(uuid.UUID)
	
	rawRole := c.Locals("role")
	fmt.Printf(">>> Raw Role: %v\n", rawRole)
	role := rawRole.(string)

	var project models.Project
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, "id = ?", id).Error; err != nil {
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

	userID := c.Locals("user_id").(uuid.UUID)

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
		fmt.Printf(">>> CreateProject DB Error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project: " + err.Error()})
	}

	// Create default notification config
	defaultConfig := models.NotificationConfig{
		ProjectID: project.ID,
	}
	database.DB.Create(&defaultConfig)

	return c.Status(fiber.StatusCreated).JSON(project)
}

func GetProjects(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var projects []models.Project

	// Admins can see all projects; users see only theirs or those they are members of
	if role == "admin" {
		database.DB.Preload("APIs").Find(&projects)
	} else {
		database.DB.Preload("APIs").
			Where("user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID, userID).
			Find(&projects)
	}

	return c.JSON(projects)
}

func GetProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var project models.Project

	query := database.DB.Preload("APIs")
	if role != "admin" {
		query = query.Where("user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?)", userID, userID)
	}

	if err := query.First(&project, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}

	return c.JSON(project)
}

func UpdateProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var input ProjectInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Use a map to check which fields were actually provided in the request
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var project models.Project
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}

	// Prepare update data only for fields present in the request
	updateData := make(map[string]interface{})
	
	if _, ok := body["name"]; ok {
		updateData["name"] = input.Name
	}
	if _, ok := body["description"]; ok {
		updateData["description"] = input.Description
	}
	if _, ok := body["environment_variables"]; ok {
		if input.EnvironmentVariables == "" {
			input.EnvironmentVariables = "{}"
		}
		updateData["environment_variables"] = input.EnvironmentVariables
	}
	if _, ok := body["cover_position"]; ok {
		updateData["cover_position"] = input.CoverPosition
	}
	if _, ok := body["company_id"]; ok {
		updateData["company_id"] = input.CompanyID
	}

	if len(updateData) > 0 {
		if err := database.DB.Model(&project).Updates(updateData).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update project"})
		}
	}

	return c.JSON(project)
}

func DeleteProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var project models.Project
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
	}

	// Soft-delete child API logs
	database.DB.Where("api_id IN (SELECT id FROM apis WHERE project_id = ?)", project.ID).Delete(&models.MonitorLog{})

	// Soft-delete child Records
	database.DB.Where("project_id = ?", project.ID).Delete(&models.API{})
	// Delete project members
	database.DB.Where("project_id = ?", project.ID).Delete(&models.ProjectMember{})
	// NotificationConfig doesn't have Soft Delete yet, but let's be consistent or add it later. 
	// For now, keep hard delete for configs if they don't have DeletedAt, or just regular Delete.
	database.DB.Where("project_id = ?", project.ID).Delete(&models.NotificationConfig{})

	database.DB.Delete(&project)

	return c.JSON(fiber.Map{"message": "Project deleted successfully"})
}

func GetProjectMembers(c *fiber.Ctx) error {
	id := c.Params("id")
	var members []models.ProjectMember
	if err := database.DB.Preload("User").Where("project_id = ?", id).Find(&members).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch project members"})
	}
	return c.JSON(members)
}

func AddProjectMember(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	type MemberInput struct {
		UserID uuid.UUID   `json:"user_id"`
		Role   string `json:"role"`
	}
	var input MemberInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var project models.Project
	if role != "admin" {
		if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only project owners or admins can manage members"})
		}
	} else {
		if err := database.DB.First(&project, "id = ?", id).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
		}
	}

	// Add member
	member := models.ProjectMember{
		ProjectID: project.ID,
		UserID:    input.UserID,
		Role:      input.Role,
	}

	if err := database.DB.Create(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to add project member (Likely already exists)"})
	}

	return c.JSON(fiber.Map{"message": "Member added successfully", "member": member})
}

func RemoveProjectMember(c *fiber.Ctx) error {
	id := c.Params("id")
	targetUserID := c.Params("userId")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var project models.Project
	if role != "admin" {
		if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only project owners or admins can manage members"})
		}
	}

	if err := database.DB.Where("project_id = ? AND user_id = ?", id, targetUserID).Delete(&models.ProjectMember{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove project member"})
	}

	return c.JSON(fiber.Map{"message": "Member removed successfully"})
}

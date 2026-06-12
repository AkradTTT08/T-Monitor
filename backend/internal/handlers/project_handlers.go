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
	"github.com/monitor-api/backend/internal/services"
)

type ProjectInput struct {
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	EnvironmentVariables string     `json:"environment_variables"`
	Folders              string     `json:"folders"`
	ExecutionMode        string     `json:"execution_mode"`
	CoverImageURL        string     `json:"cover_image_url"`
	CoverPosition        int        `json:"cover_position"`
	CompanyID            *uuid.UUID `json:"company_id"`
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
	if role == "admin" {
		if err := database.DB.First(&project, "id = ?", id).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
		}
	} else {
		if err := database.DB.Where("id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))", id, userID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to update project cover"})
		}
	}
	fmt.Println(">>> Project loaded")

	file, err := c.FormFile("cover")
	if err != nil {
		fmt.Printf(">>> FormFile error: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}
	fmt.Printf(">>> File received: %s, Size: %d\n", file.Filename, file.Size)

	// Check file extension
	ext := filepath.Ext(file.Filename)
	validExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !validExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file extension. Only jpg, png, and webp are allowed."})
	}

	// Check MIME type
	contentType := file.Header.Get("Content-Type")
	validMimes := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
	if !validMimes[contentType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid MIME type. Only image files are allowed."})
	}

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

	execMode := input.ExecutionMode
	if execMode == "" {
		execMode = "sequential"
	}

	project := models.Project{
		Name:                 input.Name,
		Description:          input.Description,
		EnvironmentVariables: input.EnvironmentVariables,
		CoverPosition:        input.CoverPosition,
		ExecutionMode:        execMode,
		UserID:               userID,
		CompanyID:            input.CompanyID,
	}

	if c.Locals("is_dry_run") == true {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "DRY_RUN: Project validation successful. Data not persisted.",
			"data":    project,
		})
	}

	svc := services.NewProjectService(nil)
	if err := svc.CreateProject(&project); err != nil {
		fmt.Printf(">>> CreateProject DB Error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(project)
}

func GetProjects(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewProjectService(nil)
	projects, err := svc.GetProjectsForUser(userID, role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch projects"})
	}

	return c.JSON(projects)
}

func GetProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewProjectService(nil)
	project, err := svc.GetProjectByID(id, userID, role)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
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
	if _, ok := body["folders"]; ok {
		if input.Folders == "" {
			input.Folders = "[]"
		}
		updateData["folders"] = input.Folders
	}
	if _, ok := body["cover_position"]; ok {
		updateData["cover_position"] = input.CoverPosition
	}
	if _, ok := body["company_id"]; ok {
		updateData["company_id"] = input.CompanyID
	}
	if _, ok := body["execution_mode"]; ok {
		mode := input.ExecutionMode
		if mode != "sequential" && mode != "parallel" {
			mode = "sequential"
		}
		updateData["execution_mode"] = mode
	}

	svc := services.NewProjectService(nil)
	project, err := svc.UpdateProject(id, updateData, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Unauthorized to update this project or project not found" {
			status = fiber.StatusForbidden // Or StatusNotFound depending on the case, StatusForbidden is safer here
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(project)
}

func DeleteProject(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewProjectService(nil)
	if err := svc.DeleteProject(id, userID, role); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Project deleted successfully"})
}

func GetProjectMembers(c *fiber.Ctx) error {
	id := c.Params("id")
	svc := services.NewProjectService(nil)
	members, err := svc.GetProjectMembers(id)
	if err != nil {
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

	svc := services.NewProjectService(nil)
	member, err := svc.AddProjectMember(id, input.UserID, input.Role, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Only project owners or admins can manage members" {
			status = fiber.StatusForbidden
		} else if err.Error() == "Project not found" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Member added successfully", "member": member})
}

func RemoveProjectMember(c *fiber.Ctx) error {
	id := c.Params("id")
	targetUserID := c.Params("userId")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewProjectService(nil)
	if err := svc.RemoveProjectMember(id, targetUserID, userID, role); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Only project owners or admins can manage members" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Member removed successfully"})
}

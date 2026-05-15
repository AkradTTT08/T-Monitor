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

type CompanyInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func GetCompanies(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewCompanyService(nil)
	companies, err := svc.GetCompaniesForUser(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(companies)
}

func GetCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewCompanyService(nil)
	company, err := svc.GetCompanyByID(id, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(company)
}

func CreateCompany(c *fiber.Ctx) error {
	var input CompanyInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewCompanyService(nil)
	company, err := svc.CreateCompany(input.Name, input.Description, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create company: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(company)
}

func UpdateCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	var input CompanyInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	svc := services.NewCompanyService(nil)
	company, err := svc.UpdateCompany(id, input.Name, input.Description, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(company)
}

func DeleteCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewCompanyService(nil)
	if err := svc.DeleteCompany(id, userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Company deleted successfully"})
}

func UploadCompanyLogo(c *fiber.Ctx) error {
	fmt.Println(">>> Starting UploadCompanyLogo")
	id := c.Params("id")
	fmt.Printf(">>> ID: %s\n", id)

	rawUserID := c.Locals("user_id")
	fmt.Printf(">>> Raw UserID: %v\n", rawUserID)
	userID := rawUserID.(uuid.UUID)

	rawRole := c.Locals("role")
	fmt.Printf(">>> Raw Role: %v\n", rawRole)

	var company models.Company
	query := database.DB
	// Strictly enforce uploading rights
	query = query.Where("user_id = ?", userID)

	if err := query.First(&company, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Company not found or unauthorized"})
	}

	file, err := c.FormFile("logo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

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

	uploadDir, err := filepath.Abs("./uploads/companies")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Path resolution failed"})
	}

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload directory"})
		}
	}

	filename := fmt.Sprintf("company_%d_%d%s", company.ID, time.Now().Unix(), ext)
	savePath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
	}

	// Update DB - only the logo_url field
	company.LogoURL = "/uploads/companies/" + filename
	if err := database.DB.Model(&company).Update("logo_url", company.LogoURL).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update database"})
	}

	return c.JSON(fiber.Map{
		"message":  "Logo uploaded successfully",
		"logo_url": company.LogoURL,
	})
}

func DebugCompany(c *fiber.Ctx) error {
	var companies []models.Company
	err := database.DB.Preload("Projects").Preload("Owner").Preload("Members.User").Find(&companies).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(companies)
}

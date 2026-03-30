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

type CompanyInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func GetCompanies(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var companies []models.Company
	query := database.DB.Preload("Projects")

	if role == "admin" {
		query.Find(&companies)
	} else {
		query.Where("user_id = ?", userID).Find(&companies)
	}

	return c.JSON(companies)
}

func CreateCompany(c *fiber.Ctx) error {
	var input CompanyInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	userID := c.Locals("user_id").(uint)

	company := models.Company{
		Name:        input.Name,
		Description: input.Description,
		UserID:      userID,
	}

	if err := database.DB.Create(&company).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create company"})
	}

	return c.Status(fiber.StatusCreated).JSON(company)
}

func UpdateCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var input CompanyInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var company models.Company
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&company, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Company not found or unauthorized"})
	}

	company.Name = input.Name
	company.Description = input.Description
	database.DB.Save(&company)

	return c.JSON(company)
}

func DeleteCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var company models.Company
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&company, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Company not found or unauthorized"})
	}

	// Disassociate projects (set company_id to null) instead of deleting them
	database.DB.Model(&models.Project{}).Where("company_id = ?", company.ID).Update("company_id", nil)

	database.DB.Delete(&company)

	return c.JSON(fiber.Map{"message": "Company deleted successfully"})
}

func UploadCompanyLogo(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var company models.Company
	query := database.DB
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&company, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Company not found or unauthorized"})
	}

	file, err := c.FormFile("logo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
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

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("company_%d_%d%s", company.ID, time.Now().Unix(), ext)
	savePath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
	}

	company.LogoURL = "/uploads/companies/" + filename
	database.DB.Save(&company)

	return c.JSON(fiber.Map{
		"message":  "Logo uploaded successfully",
		"logo_url": company.LogoURL,
	})
}

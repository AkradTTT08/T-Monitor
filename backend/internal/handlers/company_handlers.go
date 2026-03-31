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
	db := database.DB.Preload("Projects").Preload("Owner").Preload("Members.User")

	if role == "admin" {
		if err := db.Find(&companies).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch companies"})
		}
	} else {
		if err := db.Where("user_id = ? OR id IN (SELECT company_id FROM company_members WHERE user_id = ?)", userID, userID).Find(&companies).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch companies"})
		}
	}

	// TRACE LOGGING
	f, _ := os.OpenFile("companies_access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		for _, comp := range companies {
			fmt.Fprintf(f, "[%s] User:%d Role:%s CompID:%d OwnerID:%d OwnerPreloaded:%v Members:%d\n", 
				time.Now().Format("15:04:05"), userID, role, comp.ID, comp.UserID, comp.Owner != nil, len(comp.Members))
		}
	}

	return c.JSON(companies)
}

func GetCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var company models.Company
	db := database.DB.Preload("Projects").Preload("Owner").Preload("Members.User")
	
	if role != "admin" {
		db = db.Where("user_id = ? OR id IN (SELECT company_id FROM company_members WHERE user_id = ?)", userID, userID)
	}

	if err := db.First(&company, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Company not found or unauthorized"})
	}

	return c.JSON(company)
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

	// Use Updates with a map to avoid overwriting LogoURL
	updateData := map[string]interface{}{
		"name":        input.Name,
		"description": input.Description,
	}

	if err := database.DB.Model(&company).Updates(updateData).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update company"})
	}

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

	// Soft-delete projects associated with the company
	database.DB.Where("company_id = ?", company.ID).Delete(&models.Project{})

	database.DB.Delete(&company)

	return c.JSON(fiber.Map{"message": "Company deleted successfully"})
}

func UploadCompanyLogo(c *fiber.Ctx) error {
	fmt.Println(">>> Starting UploadCompanyLogo")
	id := c.Params("id")
	fmt.Printf(">>> ID: %s\n", id)

	rawUserID := c.Locals("user_id")
	fmt.Printf(">>> Raw UserID: %v\n", rawUserID)
	userID := rawUserID.(uint)

	rawRole := c.Locals("role")
	fmt.Printf(">>> Raw Role: %v\n", rawRole)
	role := rawRole.(string)

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

package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

// GetCompanyMembers returns all members of a company
func GetCompanyMembers(c *fiber.Ctx) error {
	companyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	var members []models.CompanyMember
	database.DB.Preload("User").
		Where("company_id = ?", companyID).
		Find(&members)

	return c.JSON(members)
}

// InviteMemberByEmail adds a user to a company by their email
func InviteMemberByEmail(c *fiber.Ctx) error {
	companyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	type Input struct {
		Email string `json:"email"`
	}
	var input Input
	if err := c.BodyParser(&input); err != nil || input.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	// 1. Check user exists
	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User with this email was not found in the system"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	// 2. Check for duplicate
	var existing models.CompanyMember
	if err := database.DB.Where("company_id = ? AND user_id = ?", companyID, user.ID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "This user is already a member of this company"})
	}

	// 3. Add member
	member := models.CompanyMember{
		CompanyID: uint(companyID),
		UserID:    user.ID,
		Role:      "member",
	}
	if err := database.DB.Create(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add member"})
	}

	// Return with user details preloaded
	database.DB.Preload("User").First(&member, member.ID)
	return c.Status(fiber.StatusCreated).JSON(member)
}

// RemoveCompanyMember removes a user from a company
func RemoveCompanyMember(c *fiber.Ctx) error {
	companyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}
	memberID, err := c.ParamsInt("memberId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid member id"})
	}

	result := database.DB.Where("id = ? AND company_id = ?", memberID, companyID).Delete(&models.CompanyMember{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}
	return c.JSON(fiber.Map{"message": "member removed"})
}

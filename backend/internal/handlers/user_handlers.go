package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func GetAllUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch users"})
	}
	return c.JSON(users)
}

func UpdateUserRole(c *fiber.Ctx) error {
	id := c.Params("id")
	
	type UpdateRoleInput struct {
		Role string `json:"role"`
	}
	
	var input UpdateRoleInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	
	if input.Role != "admin" && input.Role != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role must be 'admin' or 'user'"})
	}
	
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	
	user.Role = input.Role
	database.DB.Save(&user)
	
	return c.JSON(user)
}

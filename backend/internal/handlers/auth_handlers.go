package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/middleware"
	"github.com/monitor-api/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
	Position   string `json:"position"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Register(c *fiber.Ctx) error {
	var input RegisterInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	user := models.User{
		Email:      input.Email,
		Password:   string(hash),
		Name:       input.Name,
		Phone:      input.Phone,
		Department: input.Department,
		Position:   input.Position,
		Role:       "user", // Default role
	}

	// Make the first user an admin automatically and approve them
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		user.Role = "admin"
		user.IsApproved = true
	}

	if err := database.DB.Create(&user).Error; err != nil {
		// Assuming failure here is due to unique email constraint violation since email is unique index
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "อีเมลนี้มีผู้ใช้งานแล้ว (Email already exists)"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User registered successfully", "user": user})
}

func Login(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Password หรือ Username ที่ไม่ถูกต้อง"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Password หรือ Username ที่ไม่ถูกต้อง"})
	}

	if !user.IsApproved {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Waiting for admin approval"})
	}

	if user.IsBlocked {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Your account has been blocked by an administrator"})
	}

	token, err := middleware.GenerateToken(user)
	if err != nil {
		log.Println("Error generating token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{"token": token, "user": user})
}

// RefreshToken generates a new bearer token using the user details from the existing validated token
func RefreshToken(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	user := models.User{
		ID:   userID,
		Role: role,
	}

	token, err := middleware.GenerateToken(user)
	if err != nil {
		log.Println("Error refreshing token:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to refresh token"})
	}

	return c.JSON(fiber.Map{"token": token, "message": "Token refreshed successfully"})
}

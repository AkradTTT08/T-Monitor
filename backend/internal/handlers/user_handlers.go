package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
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

func ApproveUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	user.IsApproved = true
	database.DB.Save(&user)

	return c.JSON(fiber.Map{"message": "User approved successfully", "user": user})
}

func DisapproveUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Delete the user
	database.DB.Delete(&user)

	return c.JSON(fiber.Map{"message": "User disapproved and removed successfully"})
}

func ResetPassword(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("T@monitor123"), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reset password"})
	}

	user.Password = string(hash)
	database.DB.Save(&user)

	return c.JSON(fiber.Map{"message": "Password reset to default successfully"})
}

func GetProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User
	if err := database.DB.Select("id", "email", "name", "department", "position", "phone", "profile_image_url", "role", "is_approved", "created_at").First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func UpdateProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	type UpdateProfileInput struct {
		Name            string `json:"name"`
		Department      string `json:"department"`
		Position        string `json:"position"`
		Phone           string `json:"phone"`
		ProfileImageURL string `json:"profile_image_url"`
	}

	var input UpdateProfileInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	user.Name = input.Name
	user.Department = input.Department
	user.Position = input.Position
	user.Phone = input.Phone
	if input.ProfileImageURL != "" {
		user.ProfileImageURL = input.ProfileImageURL
	}
	database.DB.Save(&user)

	// Exclude password from response
	user.Password = ""
	return c.JSON(user)
}

func UpdatePassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	type UpdatePasswordInput struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	var input UpdatePasswordInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Incorrect current password"})
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}

	user.Password = string(hash)
	database.DB.Save(&user)

	return c.JSON(fiber.Map{"message": "Password updated successfully"})
}

func ToggleBlockUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Prevent blocking admins
	if user.Role == "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cannot block an administrator"})
	}

	user.IsBlocked = !user.IsBlocked
	database.DB.Save(&user)

	status := "unblocked"
	if user.IsBlocked {
		status = "blocked"
	}

	return c.JSON(fiber.Map{
		"message":    "User " + status + " successfully",
		"is_blocked": user.IsBlocked,
	})
}

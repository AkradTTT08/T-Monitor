package middleware

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/monitor-api/backend/internal/models"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		return []byte("super-secret-monitor-key")
	}
	return []byte(secret)
}

// GenerateToken creates a JWT token for standard user login
func GenerateToken(user models.User) (string, error) {
	secret := getJWTSecret()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 1).Unix(), // 1 hour expiration
	})

	return token.SignedString(secret)
}

// Protected route middleware
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing Authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid Authorization header format"})
		}

		tokenString := parts[1]
		
		secret := getJWTSecret()

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Could not parse claims"})
		}

		fmt.Printf(">>> JWT Claims: %v\n", claims)

		userID, ok := claims["user_id"].(float64)
		if !ok {
			fmt.Printf(">>> ERROR: user_id claim is NOT float64 but %T\n", claims["user_id"])
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user_id in token"})
		}
		
		role, ok := claims["role"].(string)
		if !ok {
			fmt.Printf(">>> ERROR: role claim is NOT string but %T\n", claims["role"])
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid role in token"})
		}

		fmt.Printf(">>> Auth Success: userID=%v, role=%s\n", userID, role)

		c.Locals("user_id", uint(userID))
		c.Locals("role", role)

		return c.Next()
	}
}

// AdminOnly middleware
func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Admin access required"})
		}
		return c.Next()
	}
}

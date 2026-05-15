package testutils

import (
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB initializes an in-memory SQLite database for unit testing.
func SetupTestDB() *gorm.DB {
	// Use pure-go sqlite with memory database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate models needed for testing
	err = db.AutoMigrate(
		&models.User{},
		&models.Company{},
		&models.CompanyMember{},
		&models.Project{},
		&models.API{},
		&models.MonitorLog{},
		&models.NotificationConfig{},
		&models.RepairTask{},
		&models.DashboardNotification{},
		&models.CompanyInvitation{},
		&models.ProjectMember{},
	)
	if err != nil {
		log.Fatalf("Test DB AutoMigrate failed: %v", err)
	}

	// Override the global DB instance for handlers
	database.DB = db
	return db
}

type FiberApp struct {
	App *fiber.App
	DB  *gorm.DB
}

// SetupTestApp initializes a basic Fiber app for testing HTTP handlers
func SetupTestApp() *fiber.App {
	app := fiber.New()
	return app
}

// GenerateMockJWT creates a valid JWT token string for a given user ID and role
func GenerateMockJWT(userID string, role string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 1).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// We use "secret" as a mock secret for testing
	t, _ := token.SignedString([]byte("secret"))
	return t
}

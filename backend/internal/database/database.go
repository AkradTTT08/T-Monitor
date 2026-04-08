package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	godotenv.Load()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Bangkok",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var db *gorm.DB
	var err error
	
	// Retry connection loop
	for i := 1; i <= 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Attempt %d: Failed to connect to database. Retrying in 2 seconds...", i)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Failed to connect to database after several attempts. \n", err)
	}

	log.Println("Database connection successfully opened")
	
	// List of all models for migration in dependency order
	allModels := []interface{}{
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
	}

	// 2. Perform AutoMigrate
	err = db.AutoMigrate(allModels...)

	if err != nil {
		log.Printf("Auto-migration failed: %v\n", err)
	}

	// ONE-TIME FIX: Populate empty schedules for historical logs
	db.Model(&models.MonitorLog{}).Where("schedule IS NULL OR schedule = ''").Update("schedule", "EVERY 1 MIN")

	DB = db
}

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Bangkok",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection successfully opened for fix.")

	// Update all logs with empty schedule to "EVERY 1 MIN"
	result := db.Model(&models.MonitorLog{}).
		Where("schedule IS NULL OR schedule = ''").
		Update("schedule", "EVERY 1 MIN")

	if result.Error != nil {
		log.Fatalf("Failed to update logs: %v", result.Error)
	}

	fmt.Printf("✅ Success! Updated %d historical log entries to 'EVERY 1 MIN'.\n", result.RowsAffected)
}

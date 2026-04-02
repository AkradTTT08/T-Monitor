package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func main() {
	godotenv.Load()
	database.ConnectDB()

	// Add missing columns by running AutoMigrate again
	// GORM AutoMigrate is safe: it only ADDS columns, never removes them
	err := database.DB.AutoMigrate(&models.NotificationConfig{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("✅ Migration complete - telegram_bot_token column added successfully!")
}

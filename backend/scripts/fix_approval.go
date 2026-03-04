package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()
	database.ConnectDB()

	// Approve all existing users to fix the schema migration
	if err := database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&models.User{}).Update("is_approved", true).Error; err != nil {
		log.Fatal("Failed to update users:", err)
	}

	log.Println("Successfully updated all existing users to approved!")
}

package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
	"github.com/monitor-api/backend/internal/models"
)

func main() {
	godotenv.Load()
	database.ConnectDB()

	// 1. Find a project
	var project models.Project
	if err := database.DB.Preload("Members").First(&project).Error; err != nil {
		log.Println("No project found to test")
		return
	}

	fmt.Printf("Testing with Project ID: %d, Owner ID: %d, Member Count: %d\n", project.ID, project.UserID, len(project.Members))

	// 2. Count existing notifications
	var countBefore int64
	database.DB.Model(&models.DashboardNotification{}).Count(&countBefore)

	// 3. Trigger notification
	handlers.CreateProjectNotification(project.ID, "test_type", "Test Title", "Test Message")

	// 4. Count after
	var countAfter int64
	database.DB.Model(&models.DashboardNotification{}).Count(&countAfter)

	diff := countAfter - countBefore
	fmt.Printf("Total notifications difference: %d\n", diff)
	
	expected := int64(1 + len(project.Members))
	if diff == expected {
		fmt.Printf("✅ SUCCESS: Created %d notifications (1 owner + %d members)\n", diff, len(project.Members))
	} else {
		fmt.Printf("❌ FAILED: Created %d notifications, but expected %d\n", diff, expected)
	}
}

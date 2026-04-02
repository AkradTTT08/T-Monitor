package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	database.ConnectDB()

	var project models.Project
	if err := database.DB.Preload("APIs").First(&project, 4).Error; err != nil {
		fmt.Printf("Error fetching project 4: %v\n", err)
		return
	}

	b, _ := json.MarshalIndent(project, "", "  ")
	fmt.Println("Project 4 details:")
	fmt.Println(string(b))

	var logs []models.MonitorLog
	database.DB.Joins("JOIN apis ON apis.id = monitor_logs.api_id").
		Where("apis.project_id = ?", 4).
		Order("checked_at desc").
		Limit(5).
		Find(&logs)
	
	fmt.Printf("\nLatest 5 logs for project 4 (Count: %d):\n", len(logs))
	for _, l := range logs {
		lb, _ := json.MarshalIndent(l, "", "  ")
		fmt.Println(string(lb))
	}
}

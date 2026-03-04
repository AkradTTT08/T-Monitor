package main

import (
	"encoding/json"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func main() {
	godotenv.Load()
	database.ConnectDB()

	var logs []models.MonitorLog
	database.DB.Order("checked_at desc").Limit(3).Find(&logs)

	for _, l := range logs {
		b, _ := json.MarshalIndent(l, "", "  ")
		fmt.Println(string(b))
	}
}

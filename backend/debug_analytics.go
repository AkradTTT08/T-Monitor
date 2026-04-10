package main

import (
	"fmt"
	"log"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"time"
)

func main() {
	database.ConnectDB()
	
	var project models.Project
	if err := database.DB.First(&project).Error; err != nil {
		log.Fatal("No project found:", err)
	}
	fmt.Printf("Testing for Project: %s (%s)\n", project.Name, project.ID)

	now := time.Now()
	periods := []string{"24h", "7d", "30d"}
	
	for _, period := range periods {
		var since time.Time
		var groupFormat string
		switch period {
		case "7d":
			since = now.AddDate(0, 0, -7)
			groupFormat = "YYYY-MM-DD HH24"
		case "30d":
			since = now.AddDate(0, 0, -30)
			groupFormat = "YYYY-MM-DD"
		default:
			since = now.Add(-24 * time.Hour)
			groupFormat = "YYYY-MM-DD HH24"
		}

		type DataPoint struct {
			Timestamp   string
			Count       int64
		}
		var dataPoints []DataPoint

		err := database.DB.Model(&models.MonitorLog{}).
			Select("TO_CHAR(checked_at, '"+groupFormat+"') as timestamp, COUNT(*) as count").
			Joins("JOIN apis ON apis.id = monitor_logs.api_id").
			Where("apis.project_id = ? AND checked_at >= ?", project.ID, since).
			Group("timestamp").
			Order("timestamp ASC").
			Scan(&dataPoints).Error

		if err != nil {
			fmt.Printf("Period %s: Error: %v\n", period, err)
		} else {
			fmt.Printf("Period %s: Found %d data points. Since: %v\n", period, len(dataPoints), since)
			if len(dataPoints) > 0 {
				fmt.Printf("  First: %s, Last: %s\n", dataPoints[0].Timestamp, dataPoints[len(dataPoints)-1].Timestamp)
			}
		}
	}
}

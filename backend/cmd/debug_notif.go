package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

func main() {
	godotenv.Load()
	database.ConnectDB()

	var configs []models.NotificationConfig
	result := database.DB.Find(&configs)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	fmt.Printf("\n=== notification_configs table (%d rows) ===\n", len(configs))
	for _, c := range configs {
		fmt.Printf("ID=%d | project_id=%d | enable_telegram=%v | telegram_chat_id=%q | enable_email=%v | email=%q | created=%v | updated=%v\n",
			c.ID, c.ProjectID, c.EnableTelegram, c.TelegramChatID, c.EnableEmail, c.EmailAddress, c.CreatedAt.Format("15:04:05"), c.UpdatedAt.Format("15:04:05"))
	}
}

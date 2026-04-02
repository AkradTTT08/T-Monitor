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

	var companies []models.Company
	err := database.DB.Preload("Projects").Preload("Owner").Preload("Members.User").Find(&companies).Error
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range companies {
		fmt.Printf("\n--- COMPANY ID: %d NAME: %s ---\n", c.ID, c.Name)
		fmt.Printf("OWNER ID: %d\n", c.UserID)
		if c.Owner != nil {
			fmt.Printf("OWNER PRELOADED: %s (%s)\n", c.Owner.Name, c.Owner.Email)
		} else {
			fmt.Printf("OWNER NOT PRELOADED!\n")
		}
		fmt.Printf("MEMBERS COUNT: %d\n", len(c.Members))
		for _, m := range c.Members {
			if m.User != nil {
				fmt.Printf("  - MEMBER: %s (%s) ROLE: %s\n", m.User.Name, m.User.Email, m.Role)
			} else {
				fmt.Printf("  - MEMBER USER NOT PRELOADED! (UserID: %d)\n", m.UserID)
			}
		}

		b, _ := json.Marshal(c)
		fmt.Printf("JSON: %s\n", string(b))
	}
}

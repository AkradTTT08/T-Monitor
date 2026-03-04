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
	godotenv.Load()
	database.ConnectDB()

	var user models.User
	if err := database.DB.Where("email = ?", "o.akrad.ttt08@gmail.com").First(&user).Error; err != nil {
		log.Fatal("User not found: ", err)
	}

	b, _ := json.MarshalIndent(user, "", "  ")
	fmt.Println("User:", string(b))
	fmt.Println("Hashed Password:", user.Password)
}

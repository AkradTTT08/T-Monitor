package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	godotenv.Load()
	database.ConnectDB()

	var user models.User
	if err := database.DB.Where("email = ?", "o.akrad.ttt08@gmail.com").First(&user).Error; err != nil {
		log.Fatal("User not found: ", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("T@monitor123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	user.Password = string(hash)
	database.DB.Save(&user)
	log.Println("Successfully reset password for o.akrad.ttt08@gmail.com to T@monitor123")
}

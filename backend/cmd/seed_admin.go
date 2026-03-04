package main

import (
	"fmt"
	"log"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	database.ConnectDB()

	email := "o.akrad.ttt08@gmail.com"
	password := "T@monitor123"

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	user := models.User{
		Email:      email,
		Password:   string(hash),
		Name:       "Admin",
		Department: "Transform",
		Position:   "System Administrator",
		Role:       "admin",
		IsApproved: true,
	}

	// Proceed to create or update
	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	
	if count > 0 {
		fmt.Println("User already exists, updating password and role...")
		database.DB.Model(&models.User{}).Where("email = ?", email).Updates(models.User{
			Password: string(hash),
			Role: "admin",
			IsApproved: true,
		})
	} else {
		if err := database.DB.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create admin user: %v", err)
		}
		fmt.Println("Admin user created successfully!")
	}
}

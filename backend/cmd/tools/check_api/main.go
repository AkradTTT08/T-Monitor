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

	var api models.API
	database.DB.First(&api, 10)

	fmt.Println("Name:", api.Name)
	fmt.Println("URL:", api.URL)
	fmt.Println("Headers:", api.Headers)
	fmt.Println("Body:", api.Body)

	b, _ := json.MarshalIndent(api, "", "  ")
	fmt.Println(string(b))
}

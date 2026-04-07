package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	godotenv.Load()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Bangkok",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var db *gorm.DB
	var err error
	
	// Retry connection loop
	for i := 1; i <= 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Attempt %d: Failed to connect to database. Retrying in 2 seconds...", i)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Failed to connect to database after several attempts. \n", err)
	}

	log.Println("Database connection successfully opened")
	
	// List of all models for migration
	allModels := []interface{}{
		&models.Company{},
		&models.CompanyMember{},
		&models.User{},
		&models.Project{},
		&models.API{},
		&models.MonitorLog{},
		&models.NotificationConfig{},
		&models.RepairTask{},
		&models.DashboardNotification{},
		&models.CompanyInvitation{},
		&models.ProjectMember{},
	}

	// 1. Try to drop incompatible columns first (for cases where rows can be preserved)
	log.Println("Pre-migration: Checking for incompatible bigint columns...")
	columnsToDrop := []struct {
		Table  string
		Column string
	}{
		{"companies", "id"},
		{"companies", "user_id"},
		{"company_members", "id"},
		{"company_members", "company_id"},
		{"company_members", "user_id"},
		{"users", "id"},
		{"projects", "id"},
		{"projects", "user_id"},
		{"projects", "company_id"},
		{"apis", "id"},
		{"apis", "project_id"},
		{"monitor_logs", "id"},
		{"monitor_logs", "api_id"},
		{"notification_configs", "id"},
		{"notification_configs", "project_id"},
		{"repair_tasks", "id"},
		{"repair_tasks", "project_id"},
		{"repair_tasks", "api_id"},
		{"repair_tasks", "approved_by"},
		{"dashboard_notifications", "id"},
		{"dashboard_notifications", "user_id"},
		{"dashboard_notifications", "project_id"},
		{"dashboard_notifications", "invitation_id"},
		{"company_invitations", "id"},
		{"company_invitations", "company_id"},
		{"company_invitations", "inviter_id"},
		{"company_invitations", "invitee_id"},
		{"project_members", "id"},
		{"project_members", "project_id"},
		{"project_members", "user_id"},
	}

	for _, col := range columnsToDrop {
		db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s CASCADE", col.Table, col.Column))
	}

	// 2. Perform AutoMigrate
	err = db.AutoMigrate(allModels...)

	if err != nil {
		log.Println("Auto-migration failed (likely due to existing rows violating NOT NULL constraints).")
		log.Println("Automatically dropping all tables to recreate schema fresh...")
		
		// This is equivalent to your drop_db.go logic
		db.Exec("DROP SCHEMA public CASCADE")
		db.Exec("CREATE SCHEMA public")
		
		// Retry migration on fresh schema
		err = db.AutoMigrate(allModels...)
		if err != nil {
			log.Fatal("Critical Error: Failed to auto-migrate database even after schema reset. \n", err)
		}
		log.Println("Database schema reset and migration successful.")
	}

	// ONE-TIME FIX: Populate empty schedules for historical logs
	db.Model(&models.MonitorLog{}).Where("schedule IS NULL OR schedule = ''").Update("schedule", "EVERY 1 MIN")

	DB = db
}

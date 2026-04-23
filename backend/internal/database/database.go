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

	// List of all models for migration in dependency order
	allModels := []interface{}{
		&models.User{},
		&models.Company{},
		&models.CompanyMember{},
		&models.Project{},
		&models.API{},
		&models.MonitorLog{},
		&models.NotificationConfig{},
		&models.RepairTask{},
		&models.DashboardNotification{},
		&models.CompanyInvitation{},
		&models.ProjectMember{},
	}

	// Perform AutoMigrate
	err = db.AutoMigrate(allModels...)
	if err != nil {
		log.Printf("Auto-migration failed: %v\n", err)
	}

	// IMPORTANT: Drop broken FK constraints AFTER AutoMigrate
	// AutoMigrate recreates FKs from GORM relationship tags, so we must
	// drop them AFTER migration. GORM still resolves relationships in Go
	// code (Preload etc.) without needing DB-level FK constraints.
	// FK naming: fk_{parent_table}_{child_table} (GORM convention)
	brokenFKs := []string{
		// companies table
		`ALTER TABLE companies DROP CONSTRAINT IF EXISTS fk_companies_owner`,
		// projects table - GORM names: fk_{parent}_{child}
		`ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_users_projects`,
		`ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_user`,
		`ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_companies_projects`,
		`ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_company`,
		// project_members, company_members
		`ALTER TABLE project_members DROP CONSTRAINT IF EXISTS fk_projects_members`,
		`ALTER TABLE project_members DROP CONSTRAINT IF EXISTS fk_users_project_members`,
		`ALTER TABLE company_members DROP CONSTRAINT IF EXISTS fk_companies_members`,
		`ALTER TABLE company_members DROP CONSTRAINT IF EXISTS fk_users_company_members`,
	}
	for _, sql := range brokenFKs {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("FK drop warning: %v\n", err)
		}
	}
	log.Println("Post-migration FK cleanup done")

	// ONE-TIME FIX: Populate empty schedules for historical logs
	db.Model(&models.MonitorLog{}).Where("schedule IS NULL OR schedule = ''").Update("schedule", "EVERY 1 MIN")

	DB = db
}

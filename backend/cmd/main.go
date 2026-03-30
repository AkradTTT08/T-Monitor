package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/handlers"
	"github.com/monitor-api/backend/internal/middleware"
	"github.com/monitor-api/backend/internal/workers"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; assuming environment variables are set")
	}

	// Connect to database
	database.ConnectDB()

	// Start Monitoring Worker
	workers.StartHealthCheckWorker()

	app := fiber.New()

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Adjust for production security
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Health-Check",
	}))
	app.Use(middleware.TestDryRunMiddleware())

	// Serve static files
	uploadPath := "./uploads"
	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		log.Printf("Warning: Uploads directory %s does not exist yet", uploadPath)
	} else {
		log.Printf("Serving static files from %s", uploadPath)
	}
	app.Static("/uploads", uploadPath)

	// API Routes setup
	setupRoutes(app)

	// Start Server
	log.Fatal(app.Listen(":8082"))
}

func setupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Base Route
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "message": "API Monitoring Backend is running"})
	})

	// Auth Routes
	auth := api.Group("/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)

	// Protected Routes
	protected := api.Group("/", middleware.Protected())

	authProtected := protected.Group("/auth")
	authProtected.Post("/refresh", handlers.RefreshToken)

	// Profile Routes
	profile := protected.Group("/profile")
	profile.Get("/", handlers.GetProfile)
	profile.Put("/", handlers.UpdateProfile)
	profile.Put("/password", handlers.UpdatePassword)

	// Project Routes
	project := protected.Group("/projects")
	project.Post("/", handlers.CreateProject)
	project.Get("/", handlers.GetProjects)
	project.Get("/:id", handlers.GetProject)
	project.Put("/:id", handlers.UpdateProject)
	project.Delete("/:id", handlers.DeleteProject)
	project.Post("/:id/cover", handlers.UploadProjectCover)

	// Company Routes
	companies := protected.Group("/companies")
	companies.Get("/", handlers.GetCompanies)
	companies.Post("/", handlers.CreateCompany)
	companies.Put("/:id", handlers.UpdateCompany)
	companies.Delete("/:id", handlers.DeleteCompany)
	companies.Post("/:id/logo", handlers.UploadCompanyLogo)

	// API Management Routes
	apis := protected.Group("/apis")
	apis.Post("/", handlers.CreateAPI)
	apis.Get("/", handlers.GetAPIs)
	apis.Put("/reorder/:id", handlers.ReorderAPIs)
	apis.Post("/test", handlers.TestAPI) // Added this line based on the instruction
	apis.Post("/:id/pause", handlers.PauseAPI)
	apis.Put("/:id", handlers.UpdateAPI)
	apis.Delete("/:id", handlers.DeleteAPI)
	apis.Post("/import-postman", handlers.UploadPostmanCollection)

	// Notifications Config
	protected.Get("/projects/:projectId/notifications", handlers.GetNotificationConfig)
	protected.Post("/notifications", handlers.UpsertNotificationConfig)

	// Logs
	protected.Get("/logs", handlers.GetMonitorLogs)

	// Admin User Routes
	users := protected.Group("/users", middleware.AdminOnly())
	users.Get("/", handlers.GetAllUsers)
	users.Put("/:id/role", handlers.UpdateUserRole)
	users.Put("/:id/approve", handlers.ApproveUser)
	users.Delete("/:id/disapprove", handlers.DisapproveUser)
	users.Put("/:id/block", handlers.ToggleBlockUser)
	users.Put("/:id/reset-password", handlers.ResetPassword)
}

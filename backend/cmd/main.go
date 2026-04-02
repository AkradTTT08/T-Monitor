package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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

	app := fiber.New(fiber.Config{
		BodyLimit: 50 * 1024 * 1024, // 50 MB limit for file uploads
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS,PATCH",
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
	profile.Post("/upload", handlers.UploadProfileImage)
	profile.Put("/password", handlers.UpdatePassword)

	// Project Routes
	project := protected.Group("/projects")
	project.Post("/", handlers.CreateProject)
	project.Get("/", handlers.GetProjects)
	project.Get("/:id", handlers.GetProject)
	project.Put("/:id", handlers.UpdateProject)
	project.Delete("/:id", handlers.DeleteProject)
	project.Post("/:id/cover", handlers.UploadProjectCover)
	// Project Members management
	project.Get("/:id/members", handlers.GetProjectMembers)
	project.Post("/:id/members", handlers.AddProjectMember)
	project.Delete("/:id/members/:userId", handlers.RemoveProjectMember)

	// Company Routes
	companies := protected.Group("/companies")
	companies.Get("/", handlers.GetCompanies)
	companies.Get("/:id", handlers.GetCompany)
	companies.Post("/", handlers.CreateCompany)
	companies.Put("/:id", handlers.UpdateCompany)
	companies.Delete("/:id", handlers.DeleteCompany)
	companies.Post("/:id/logo", handlers.UploadCompanyLogo)
	companies.Post("/:id/invite", handlers.InviteMemberByEmail)
	companies.Post("/invitations/:id/accept", handlers.AcceptCompanyInvitation)
	companies.Post("/invitations/:id/decline", handlers.DeclineCompanyInvitation)
	companies.Delete("/:id/members/:memberId", handlers.RemoveCompanyMember)

	// Debug Routes
	api.Get("/debug-companies", handlers.DebugCompany)

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

	// Repair Task Routes
	protected.Post("/upload", handlers.UploadMultipleFiles)
	protected.Get("/download", handlers.DownloadFile)
	protected.Get("/projects/:id/repair-tasks", handlers.GetRepairTasks)
	protected.Post("/repair-tasks/:id/approve", handlers.ApproveRepairTask)
	protected.Post("/repair-tasks/:id/close", handlers.CloseRepairTask)
	protected.Post("/repair-tasks/:id/fail", handlers.FailRepairTask)

	// Dashboard Notification Routes
	protected.Get("/notifications/unread", handlers.GetNotifications)
	protected.Put("/notifications/:id/read", handlers.MarkNotificationRead)

	// Admin User Routes
	users := protected.Group("/users", middleware.AdminOnly())
	users.Get("/", handlers.GetAllUsers)
	users.Put("/:id/role", handlers.UpdateUserRole)
	users.Put("/:id/approve", handlers.ApproveUser)
	users.Delete("/:id/disapprove", handlers.DisapproveUser)
	users.Delete("/:id", handlers.DeleteUser)
	users.Put("/:id/block", handlers.ToggleBlockUser)
	users.Put("/:id/reset-password", handlers.ResetPassword)

	// Generic User Routes
	protected.Get("/users/search", handlers.SearchUsers)
}

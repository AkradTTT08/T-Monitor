package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/services"
)

// GetUptimeStats returns uptime statistics for all APIs in a project
func GetUptimeStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	projectID := c.Query("project_id")
	period := c.Query("period", "24h")

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	svc := services.NewAnalyticsService(nil)
	stats, err := svc.GetUptimeStats(projectID, period, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Access denied" || err.Error() == "Project not found" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(stats)
}

// GetLatencyTrend returns time-series latency data for charts
func GetLatencyTrend(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	projectID := c.Query("project_id")
	period := c.Query("period", "24h")

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	svc := services.NewAnalyticsService(nil)
	trend, err := svc.GetLatencyTrend(projectID, period, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Access denied" || err.Error() == "Project not found" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(trend)
}

// GetIncidentTimeline returns recent incidents for timeline display
func GetIncidentTimeline(c *fiber.Ctx) error {
	projectID := c.Query("project_id")
	limit := c.QueryInt("limit", 20)

	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	svc := services.NewAnalyticsService(nil)
	timeline, err := svc.GetIncidentTimeline(projectID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(timeline)
}

// GetGlobalPulse returns high-level metrics and recent pings across all accessible projects
func GetGlobalPulse(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	
	companyID := c.Query("company_id")
	projectID := c.Query("project_id")

	svc := services.NewAnalyticsService(nil)
	pulse, err := svc.GetGlobalPulse(userID, role, companyID, projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(pulse)
}

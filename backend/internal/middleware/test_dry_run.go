package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// TestDryRunMiddleware checks if the request is a test/monitoring request
func TestDryRunMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check for dry_run=true query parameter
		isDryRunQuery := c.Query("dry_run") == "true"
		
		// Check for X-Health-Check: T-Monitor header
		isHealthCheckHeader := c.Get("X-Health-Check") == "T-Monitor"

		if isDryRunQuery || isHealthCheckHeader {
			c.Locals("is_dry_run", true)
		} else {
			c.Locals("is_dry_run", false)
		}

		return c.Next()
	}
}

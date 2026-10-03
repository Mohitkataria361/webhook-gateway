package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger is a simple request/response logging middleware.
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		log.Printf("[HTTP] %s %s → %d (%dms)",
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration.Milliseconds(),
		)
		return err
	}
}

// CORS adds permissive CORS headers for the React dashboard during development.
// In production, restrict AllowOrigins to specific domains.
func CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key")

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

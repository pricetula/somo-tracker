package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// newSecurityHeaders injects standard HTTP protection headers.
func newSecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(c.Path(), "/api/") || c.Path() == "/health" {
			c.Set("Content-Security-Policy", "default-src 'self'")
		}
		return c.Next()
	}
}

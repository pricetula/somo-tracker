// Package helmet provides security header middleware for the Somotracker API.
// It injects standard production security headers into every HTTP response.
package helmet

import (
	"github.com/gofiber/fiber/v3"
)

// Middleware returns a Fiber handler that injects standard security headers
// into each response. Headers are set after the handler chain completes via defer
// to ensure they are present on successful responses, errors, and panics recovered upstream.
// X-Request-ID set by the request ID middleware is intentionally left untouched.
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			// Content-Type sniffing prevention.
			c.Set("X-Content-Type-Options", "nosniff")
			// Clickjacking protection.
			c.Set("X-Frame-Options", "DENY")
			// Content Security Policy for API responses (no inline scripts/styles).
			c.Set("Content-Security-Policy", "default-src 'none'")
		}()

		return c.Next()
	}
}

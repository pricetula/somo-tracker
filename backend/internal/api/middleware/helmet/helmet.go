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
			// Referrer policy.
			c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// Content Security Policy for API responses (no inline scripts/styles).
			c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'")
			// HSTS (only set in production via config, but we set it here for HTTPS)
			// Note: In production behind TLS termination, this header may be set by reverse proxy.
			// c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}()

		return c.Next()
	}
}

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func newSecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Prevent MIME-type sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// 2. Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")

		// 3. Limit referrer leakage
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 4. Disable buggy legacy XSS auditor in older browsers (favor CSP)
		c.Set("X-XSS-Protection", "0")

		// 5. Restrict access to browser hardware APIs
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// 6. Cross-Origin Resource Policy
		c.Set("Cross-Origin-Resource-Policy", "same-site")

		// 7. Enforce HTTPS via HSTS (only on TLS or when behind an SSL-terminating reverse proxy)
		if c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// 8. Scope CSP to API & Health endpoints
		if strings.HasPrefix(c.Path(), "/api/") || c.Path() == "/health" {
			c.Set("Content-Security-Policy", "default-src 'self'")
		}

		return c.Next()
	}
}

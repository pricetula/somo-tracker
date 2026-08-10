package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// newCSRFGuard enforces double-submit cookie validation for state-changing requests.
func newCSRFGuard() fiber.Handler {
	ignoredPrefixes := []string{"/api/auth/discover", "/api/auth/verify", "/api/auth/register"}

	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}

		path := c.Path()
		for _, prefix := range ignoredPrefixes {
			if path == prefix || strings.HasPrefix(path, prefix) {
				return c.Next()
			}
		}

		cookieToken := c.Cookies("csrf_token")
		headerToken := c.Get("X-CSRF-Token")

		if cookieToken == "" || headerToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    "forbidden",
				"message": "csrf token missing",
			})
		}

		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    "forbidden",
				"message": "csrf token mismatch",
			})
		}

		return c.Next()
	}
}

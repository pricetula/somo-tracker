package middleware

import (
	"crypto/subtle"
	"somotracker/backend/internal/config"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func NewCSRFGuard(cfg config.Config) fiber.Handler {
	ignoredPrefixes := []string{
		"/api/auth/discover",
		"/api/auth/verify",
		"/api/auth/register",
	}

	return func(c *fiber.Ctx) error {
		method := c.Method()

		// 1. Safe methods skip CSRF checks
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}

		path := c.Path()

		// 2. Strict path exemption check (exact match or path boundary check)
		for _, prefix := range ignoredPrefixes {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return c.Next()
			}
		}

		// 3. Defense-in-Depth: Origin Header Check
		if origin := c.Get("Origin"); origin != "" && cfg.AllowedOrigins != "" {
			allowed := false
			for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
				if strings.TrimSpace(o) == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				return c.Status(fiber.StatusForbidden).JSON(withRequestID(c, fiber.Map{
					"code":    "forbidden",
					"message": "invalid origin source",
				}))
			}
		}

		// 4. Double-Submit Cookie Matching
		cookieToken := c.Cookies("csrf_token")
		headerToken := c.Get("X-CSRF-Token")

		if cookieToken == "" || headerToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(withRequestID(c, fiber.Map{
				"code":    "forbidden",
				"message": "csrf token missing",
			}))
		}

		// Constant-time comparison to prevent timing side-channel attacks
		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(withRequestID(c, fiber.Map{
				"code":    "forbidden",
				"message": "csrf token mismatch",
			}))
		}

		return c.Next()
	}
}

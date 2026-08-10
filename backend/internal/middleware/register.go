package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Register mounts all security and context middleware on the Fiber app in strict execution order.
func Register(app *fiber.App, pools *database.Pools, cfg config.Config) {
	app.Use(newPanicRecover())
	app.Use(newCORS(cfg))
	app.Use(newSecurityHeaders())
	app.Use(newCSRFGuard(cfg))
	app.Use(newRateLimiter(pools.Redis, RateLimiterConfig{
		Limit:  300,
		Window: 1 * time.Minute,
		Prefix: "ip_coarse",
		KeyLookup: func(c *fiber.Ctx) string {
			return c.IP()
		},
	}))
	app.Use(newSessionResolver(pools))
	app.Use(newRateLimiter(pools.Redis, RateLimiterConfig{
		Limit:  60,
		Window: 1 * time.Minute,
		Prefix: "user_fine",
		KeyLookup: func(c *fiber.Ctx) string {
			if sess, ok := c.Locals("session").(*SessionInfo); ok && sess.UserID != "" {
				return sess.UserID
			}
			return "" // Skips fine rate limiter if user is unauthenticated
		},
	}))
	app.Use(newDeviceFingerprinter())
}

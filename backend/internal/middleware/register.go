package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Register mounts all security and context middleware on the Fiber app in strict execution order.
func Register(app *fiber.App, pools *database.Pools, cfg config.Config) {
	app.Use(NewPanicRecover())
	app.Use(NewCORS(cfg))
	app.Use(NewSecurityHeaders())
	app.Use(NewCSRFGuard(cfg))
	app.Use(NewRateLimiter(pools.Redis, RateLimiterConfig{
		Limit:  300,
		Window: 1 * time.Minute,
		Prefix: "ip_coarse",
		KeyLookup: func(c *fiber.Ctx) string {
			return c.IP()
		},
	}))
	// Device fingerprint MUST be computed before session resolution so the
	// resolver can enforce device-bound sessions in production (C5). It also
	// feeds session creation (auth handlers read c.Locals("device_fingerprint")).
	app.Use(NewDeviceFingerprinter())
	app.Use(NewSessionResolver(pools, cfg))
	app.Use(NewRateLimiter(pools.Redis, RateLimiterConfig{
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
}

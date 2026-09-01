package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Register mounts all security and context middleware on the Fiber app in
// strict execution order:
//
//	panic recover → request ID → CORS → security headers → CSRF → coarse IP
//	limit → device fingerprint → session resolver → access log → fine user limit
//
// Request ID runs early so every log line and error response downstream can
// carry a correlation ID. The access log runs after the session resolver so
// it can attach user_id/tenant_id/role; requests rejected earlier in the chain
// (CORS, CSRF, coarse IP limit) are therefore not access-logged.
func Register(app *fiber.App, pools *database.Pools, cfg config.Config, logger *zap.SugaredLogger) {
	// WithLogger MUST be the first middleware: HTTPError and the other
	// middleware helpers read the logger from c.Locals, and the global error
	// handler runs after the chain unwinds.
	app.Use(WithLogger(logger))
	app.Use(NewPanicRecover())
	app.Use(NewRequestID())
	app.Use(NewCORS(cfg))
	app.Use(NewSecurityHeaders(cfg))
	app.Use(NewCSRFGuard(cfg))
	app.Use(NewRateLimiter(pools.Redis, RateLimiterConfig{
		Limit:  cfg.RateLimitIPMax,
		Window: cfg.RateLimitWindow,
		Prefix: "ip_coarse",
		KeyLookup: func(c *fiber.Ctx) string {
			// C3: Stytch magic-link redirect targets must not be IP-limited —
			// the one-time token in the URL is the auth proof and schools sit
			// behind shared NAT egress IPs, so a per-IP throttle would lock an
			// entire school out of login.
			if isStytchCallback(c.Path()) {
				return ""
			}
			return c.IP()
		},
	}))
	// Device fingerprint MUST be computed before session resolution so the
	// resolver can enforce device-bound sessions in production (C5). It also
	// feeds session creation (auth handlers read c.Locals("device_fingerprint")).
	app.Use(NewDeviceFingerprinter())
	app.Use(NewSessionResolver(pools, cfg))
	// Tenant context MUST run after session resolution (it reads the resolved
	// session's tenant) and before any handler touches the database.
	app.Use(WithTenantContext(pools))
	app.Use(NewAccessLog())
	app.Use(NewRateLimiter(pools.Redis, RateLimiterConfig{
		Limit:  cfg.RateLimitUserMax,
		Window: cfg.RateLimitWindow,
		Prefix: "user_fine",
		KeyLookup: func(c *fiber.Ctx) string {
			if sess, ok := c.Locals("session").(*SessionInfo); ok && sess.UserID != "" {
				return sess.UserID
			}
			return "" // Skips fine rate limiter if user is unauthenticated
		},
	}))
}

// isStytchCallback reports whether path is a Stytch magic-link redirect
// target. These routes are exempt from IP-based throttling (C3).
func isStytchCallback(path string) bool {
	return path == "/api/auth/callback" || path == "/api/auth/invite/callback"
}

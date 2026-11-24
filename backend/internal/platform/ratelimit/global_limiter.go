// Package ratelimit provides Redis-backed rate limiting using sliding window algorithm.
package ratelimit

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"somotracker/internal/platform/config"
	"somotracker/internal/platform/middleware"
)

// GlobalLimiter implements global rate limiting for all /api/v1/* routes.
// Keys on userID when authenticated, falls back to IP for unauthenticated requests.
type GlobalLimiter struct {
	limiter    Limiter
	keyBuilder KeyBuilder
	cfg        *config.Config
}

// NewGlobalLimiter creates a new global rate limiter.
func NewGlobalLimiter(limiter Limiter, keyBuilder KeyBuilder, cfg *config.Config) *GlobalLimiter {
	return &GlobalLimiter{
		limiter:    limiter,
		keyBuilder: keyBuilder,
		cfg:        cfg,
	}
}

// Middleware returns the Fiber middleware for global rate limiting.
func (gl *GlobalLimiter) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Skip non-API routes
		if len(c.Path()) < 8 || c.Path()[:8] != "/api/v1/" {
			return c.Next()
		}

		// Determine rate limit key: userID > IP
		var key string
		var limit int
		var window time.Duration

		userID := middleware.GetUserIDFromContext(c)
		if userID != "" {
			key = gl.keyBuilder.UserKey(userID)
			limit = gl.cfg.RateLimitGlobalUserPerMin
			window = time.Minute
		} else {
			clientIP := middleware.ClientIPFromContext(c)
			if clientIP == "" {
				clientIP = NormalizeIP(c.IP())
			}
			key = gl.keyBuilder.IPKey(clientIP)
			limit = gl.cfg.RateLimitGlobalIPPerMin
			window = time.Minute
		}

		ctx := c.Context()
		decision, err := gl.limiter.Allow(ctx, key, limit, window)
		if err != nil {
			// Log error but don't block on limiter failure
			// TODO: Add proper logging
			return c.Next()
		}

		// Set rate limit headers
		c.Set("RateLimit-Limit", fmt.Sprintf("%d", decision.Limit))
		c.Set("RateLimit-Remaining", fmt.Sprintf("%d", decision.Remaining))
		c.Set("RateLimit-Reset", fmt.Sprintf("%d", int(decision.ResetAfter.Seconds())))
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", decision.Limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", decision.Remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(decision.ResetAfter).Unix()))

		if !decision.Allowed {
			c.Set("Retry-After", fmt.Sprintf("%d", int(decision.RetryAfter.Seconds())))
			return c.Status(429).JSON(fiber.Map{
				"code":                 "rate_limited",
				"message":              "Too many requests",
				"retry_after_seconds":  int(decision.RetryAfter.Seconds()),
			})
		}

		return c.Next()
	}
}
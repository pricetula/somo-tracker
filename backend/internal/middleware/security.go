package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	fibermiddleware "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Register mounts the full security middleware pipeline on the provided Fiber app.
// Layers are registered in order: CORS, panic recovery, security headers,
// CSRF guard, rate limiter, device fingerprinting.
func Register(app *fiber.App, pools *database.Pools, cfg config.Config) {
	// Layer 0 — CORS (before everything else for preflight handling)
	registerCORS(app, cfg)

	// Layer 1 — Panic recovery
	app.Use(fibermiddleware.New())

	// Layer 2 — Security headers
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		// Scope CSP to API routes only; the Next.js frontend owns its own CSP
		if strings.HasPrefix(c.Path(), "/api/") || c.Path() == "/health" {
			c.Set("Content-Security-Policy", "default-src 'self'")
		}
		return c.Next()
	})

	// Layer 3 — CSRF double-submit cookie pattern
	// On state-changing requests (POST, PUT, DELETE, PATCH), compares the
	// csrf_token cookie value against the X-CSRF-Token request header.
	// Safe methods (GET, HEAD, OPTIONS) are not checked.
	// The csrf_token cookie is set as non-HttpOnly so the frontend JS can read it.
	app.Use(func(c *fiber.Ctx) error {
		method := c.Method()
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}

		cookieToken := c.Cookies("csrf_token")
		headerToken := c.Get("X-CSRF-Token")

		if cookieToken == "" || headerToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "forbidden",
				"reason": "csrf token missing",
			})
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "forbidden",
				"reason": "csrf token mismatch",
			})
		}

		return c.Next()
	})

	// Layer 4 — Redis sliding-window rate limiter
	app.Use(func(c *fiber.Ctx) error {
		allowed, err := checkRateLimit(c, pools.Redis)
		if err != nil {
			// If Redis is down, fail open (allow the request) to avoid cascading outages
			return c.Next()
		}
		if !allowed {
			c.Set("Retry-After", "60")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":               "rate_limit_exceeded",
				"retry_after_seconds": 60,
			})
		}
		return c.Next()
	})

	// Layer 5 — Device fingerprinting
	app.Use(func(c *fiber.Ctx) error {
		ip := c.IP()
		ua := c.Get("User-Agent")
		al := c.Get("Accept-Language")
		raw := ip + "|" + ua + "|" + al
		hash := sha256.Sum256([]byte(raw))
		fingerprint := hex.EncodeToString(hash[:])
		c.Locals("device_fingerprint", fingerprint)
		return c.Next()
	})
}

// checkRateLimit implements a sliding-window rate limiter using a Redis sorted set.
// Returns true if the request is within the limit, false if exceeded.
// On Redis errors, it fails open (returns true) to avoid cascading failures.
func checkRateLimit(c *fiber.Ctx, rdb *redis.Client) (bool, error) {
	const (
		window = int64(60000) // 1 minute in milliseconds
		limit  = int64(60)    // 60 requests per window
	)

	ip := c.IP()
	key := "ratelimit:" + ip
	now := time.Now().UnixMilli()
	uid := fmt.Sprintf("%d", now) + ":" + ip // best-effort unique ID

	script := redis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local id     = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
  return 0
end
redis.call('ZADD', key, now, id)
redis.call('PEXPIRE', key, window)
return 1
`)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	result, err := script.Run(ctx, rdb, []string{key}, now, window, limit, uid).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

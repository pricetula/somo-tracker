package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig configures a sliding-window rate limiter tier.
type RateLimiterConfig struct {
	Limit     int64
	Window    time.Duration
	KeyLookup func(c *fiber.Ctx) string // Returns unique key to limit by (e.g. "ip:1.2.3.4" or "user:123")
	Prefix    string
}

// NewRateLimiter returns a Redis sliding-window rate limiter middleware.
func NewRateLimiter(rdb *redis.Client, cfg RateLimiterConfig) fiber.Handler {
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

	return func(c *fiber.Ctx) error {
		identifier := cfg.KeyLookup(c)
		if identifier == "" {
			// If key evaluator returns empty (e.g., User Limiter on unauthenticated request), skip
			return c.Next()
		}

		key := fmt.Sprintf("ratelimit:%s:%s", cfg.Prefix, identifier)
		now := time.Now().UnixMilli()
		windowMs := cfg.Window.Milliseconds()

		// Use nanosecond + random seed to guarantee ZADD uniqueness in high-concurrency windows
		memberID := fmt.Sprintf("%d:%d", time.Now().UnixNano(), rand.Intn(100000))

		ctx, cancel := context.WithTimeout(c.UserContext(), 100*time.Millisecond)
		defer cancel()

		result, err := script.Run(ctx, rdb, []string{key}, now, windowMs, cfg.Limit, memberID).Int()
		if err != nil {
			// Fail-open strategy: If Redis fails, allow request through to prevent system outage
			return c.Next()
		}

		if result == 0 {
			c.Set("Retry-After", fmt.Sprintf("%d", int(cfg.Window.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":                "rate_limit_exceeded",
				"message":             "too many requests, please try again later",
				"retry_after_seconds": int(cfg.Window.Seconds()),
			})
		}

		return c.Next()
	}
}

// NewIPLimiter returns a sliding‑window limiter keyed by client IP.
// Typical use: global anti‑abuse throttle (e.g. 300 req/min).
func NewIPLimiter(rdb *redis.Client, limit int64, window time.Duration) fiber.Handler {
	return NewRateLimiter(rdb, RateLimiterConfig{
		Limit:  limit,
		Window: window,
		Prefix: "ip_limiter",
		KeyLookup: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})
}

// NewUserLimiter returns a sliding‑window limiter keyed by the authenticated user ID.
// If the request has no session, the limiter is skipped (fail‑open).
func NewUserLimiter(rdb *redis.Client, limit int64, window time.Duration) fiber.Handler {
	return NewRateLimiter(rdb, RateLimiterConfig{
		Limit:  limit,
		Window: window,
		Prefix: "user_limiter",
		KeyLookup: func(c *fiber.Ctx) string {
			if sess, ok := c.Locals("session").(*SessionInfo); ok && sess.UserID != "" {
				return sess.UserID
			}
			return "" // skip limiter for unauthenticated requests
		},
	})
}

// NewEndpointLimiter returns a limiter keyed by the combination of user ID + request path.
// Useful for throttling expensive endpoints on a per‑user basis.
func NewEndpointLimiter(rdb *redis.Client, limit int64, window time.Duration) fiber.Handler {
	return NewRateLimiter(rdb, RateLimiterConfig{
		Limit:  limit,
		Window: window,
		Prefix: "endpoint_limiter",
		KeyLookup: func(c *fiber.Ctx) string {
			if sess, ok := c.Locals("session").(*SessionInfo); ok && sess.UserID != "" {
				return sess.UserID + "|" + c.Path()
			}
			return "" // skip if no user
		},
	})
}

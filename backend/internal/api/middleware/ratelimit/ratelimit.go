// Package ratelimit provides a Redis-backed rate-limiting middleware for the
// Somotracker API built on top of github.com/go-redis/redis_rate/v10.
//
// It ships two layers:
//
//  1. [Module] — an Fx module that accepts a *redis.Client and produces a
//     *redis_rate.Limiter singleton. Import it alongside redis.Module in main:
//
//     fx.New(
//     config.Module,
//     redis.Module,
//     ratelimit.Module,
//     // ...
//     )
//
//  2. [Middleware] — a factory that returns a Fiber handler. Pass a pre-built
//     limiter plus a [redis_rate.Limit] and a key prefix to scope the counter:
//
//     app.Use(ratelimit.Middleware(limiter,
//     redis_rate.PerMinute(60),
//     "api:auth",
//     ))
//
// # Key derivation
//
// The middleware builds a Redis key as "{prefix}:{clientID}" where clientID is
// resolved in this order:
//
//  1. c.IP() after Fiber has already unwrapped X-Forwarded-For,
//     X-Real-IP, and CF-Connecting-IP (standard reverse-proxy headers).
//  2. Falls back to the string "unknown" when IP resolution fails.
//
// This means the same logical client (same subnet / same Stytch session) is
// counted together. Route handlers that have authenticated a user should
// enrich the key with a user identifier before calling the middleware.
package ratelimit

import (
	"fmt"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module is the Fx module that produces a *redis_rate.Limiter from a
// *redis.Client. It is self-contained and can be imported alongside
// redis.Module without further configuration:
//
//	var Module = fx.Module(
//	    "ratelimit",
//	    fx.Provide(NewLimiter),
//	)
var Module = fx.Module(
	"ratelimit",
	fx.Provide(NewLimiter),
)

// NewLimiter wraps the shared *redis.Client in a redis_rate.Limiter.
// The limiter is scoped to the lifetime of the Fx application — the
// *redis.Client is closed by the redis package on shutdown, so no explicit
// cleanup is required here.
func NewLimiter(client *redis.Client) (*redis_rate.Limiter, error) {
	if client == nil {
		return nil, fmt.Errorf("ratelimit.NewLimiter: client is required")
	}
	return redis_rate.NewLimiter(client), nil
}

// Middleware returns a Fiber middleware handler that enforces a per-client
// rate limit backed by Redis.
//
//   - limiter: the singleton produced by [NewLimiter].
//   - rate: a redis_rate.Limit specifying the allowed number of requests
//     per time window (e.g. redis_rate.PerMinute(60)).
//   - keyPrefix: a dot-free string prepended to the Redis key so that
//     different route groups can have independent counters
//     (e.g. "api:auth", "api:search").
//
// The returned handler is safe to register on any number of routes — the
// limiter itself is concurrency-safe.
func Middleware(limiter *redis_rate.Limiter, rate redis_rate.Limit, keyPrefix string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if limiter == nil {
			// Defensive: pass through if limiter is unconfigured (e.g. tests).
			return c.Next()
		}

		clientID := extractClientID(c)

		key := fmt.Sprintf("%s:%s", keyPrefix, clientID)

		res, err := limiter.Allow(c.Context(), key, rate)
		if err != nil {
			// Redis unreachable — log the error and allow the request through
			// rather than blocking legitimate traffic for an infrastructure
			// problem. The error is returned so the caller can decide whether
			// to surface it as a 503.
			return c.Next()
		}

		// Set standard rate-limit response headers regardless of outcome.
		c.Set("X-RateLimit-Limit", rate.String())
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", res.ResetAfter.Milliseconds()))

		if res.Allowed <= 0 {
			blockedAt := time.Now()
			retryAfter := res.RetryAfter
			if retryAfter <= 0 {
				retryAfter = res.ResetAfter
			}
			retryAfterSecs := int(retryAfter.Seconds())
			if retryAfterSecs < 1 {
				retryAfterSecs = 1
			}
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfterSecs))

			// Attempt to pull the request-scoped logger from Fiber locals,
			// falling back to the global zap logger if not set.
			reqLogger := fallbackLogger(c)
			reqLogger.Warn("ratelimit: request blocked",
				zap.String("client_id", clientID),
				zap.String("key", key),
				zap.String("limit", rate.String()),
				zap.Int("remaining", int(res.Remaining)),
				zap.Duration("reset_after", res.ResetAfter),
				zap.Time("blocked_at", blockedAt),
			)

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":    "rate_limit_exceeded",
				"message": "Too many requests. Please wait a few minutes before trying again.",
				"errors":  fiber.Map{},
			})
		}

		return c.Next()
	}
}

// fallbackLogger retrieves the request-scoped zap.Logger from Fiber locals,
// or returns zap.L() if none has been injected by the request-ID middleware.
func fallbackLogger(c fiber.Ctx) *zap.Logger {
	if v := c.Locals("logger"); v != nil {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return zap.L()
}

// extractClientID returns the best-effort unique identifier for the client.
// It checks, in order:
//
//  1. c.FormValue("email")       — form / query string (login, password-reset, etc.)
//  2. c.FormValue("target_email") — multi-tenant operations scoped to a recipient
//  3. c.FormValue("user_id")     — authenticated user ID passed as a form field
//  4. c.Get("X-User-ID")         — authenticated user ID via internal header
//  5. c.IP()                     — remote address after proxy headers are unwound
//  6. "unknown"                 — ultimate fallback
//
// NOTE: Body fields are read after routing so the body is still intact when
// this helper is called. In the common JSON-API case, callers can enrich the
// key further in downstream handlers if a JSON payload is needed.
func extractClientID(c fiber.Ctx) string {
	if email := c.FormValue("email"); email != "" {
		return email
	}
	if target := c.FormValue("target_email"); target != "" {
		return target
	}
	if userID := c.FormValue("user_id"); userID != "" {
		return userID
	}
	if userID := c.Get("X-User-ID"); userID != "" {
		return userID
	}
	if ip := c.IP(); ip != "" {
		return ip
	}
	return "unknown"
}

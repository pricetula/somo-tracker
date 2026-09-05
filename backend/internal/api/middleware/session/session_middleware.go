// Package session provides Fiber middleware for secure session validation
// and multi-tenant Row-Level Security (RLS) context propagation.
//
// The middleware performs:
//   - Opaque session cookie extraction (session_token)
//   - Redis-backed session verification with expiration checks
//   - Multi-tenant metadata injection (user_id, tenant_id) into Fiber locals
//   - Security event logging for unauthorized access attempts
//
// Integration example:
//
//	app.Get("/protected", session.NewSessionMiddleware(redisClient), handler)
//
// The middleware is typically applied to entire route groups using app.Group:
//
//	protected := app.Group("/api/v1", session.NewSessionMiddleware(redisClient))
//
// The constructor function is provided for Fx dependency injection and to
// maintain consistency with other middleware constructors in the codebase.
package session

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// CookieName is the expected session token cookie name.
	CookieName = "session_token"

	// sanitizedUnauthorizedMessage is the client-facing error for missing/invalid sessions.
	// No internal details or token values are ever exposed.
	sanitizedUnauthorizedMessage = "Unauthorized. Please log in to continue."

	// sessionCacheTTL is the Redis TTL for session metadata.
	// Derived from internal/session/session.go: 6h cache vs 7h DB expiry.
	sessionCacheTTL = 6 * time.Hour
)

// NewSessionMiddleware constructs a Fiber middleware handler for session
// validation and multi-tenant RLS context propagation.
//
// The middleware expects a *redis.Client to be injected (typically via Fx DI).
// If the client is nil, requests pass through without validation (useful for
// graceful degradation or testing).
//
// Security guarantees:
//   - Missing cookie → 401 with sanitized message (no login hint)
//   - Missing/expired session in Redis → 401 + cleanup of stale cookie
//   - All errors return the canonical JSON response contract
//   - No internal session tokens or Redis keys leak in responses
//
// On success, the middleware injects:
//   - c.Locals("user_id", userID)   – the authenticated user's UUID
//   - c.Locals("tenant_id", tenantID) – the organization/tenant UUID
//
// The function signature matches the pattern used by other middleware constructors
// in the codebase (e.g. NewRequestIDHandler, NewRateLimitMiddleware).
func NewSessionMiddleware(client *redis.Client, logger *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Early return if redis client is not configured (defensive for tests).
		if client == nil {
			if logger != nil {
				logger.Warn("session middleware: redis client not configured, skipping validation")
			} else {
				zap.L().Warn("session middleware: redis client not configured, skipping validation")
			}
			return c.Next()
		}

		// Extract the opaque session token from the cookie.
		token := c.Cookies(CookieName)
		if token == "" {
			// Log the unauthorized attempt with client context for security monitoring.
			logUnauthorized(c, logger, "missing_session_cookie", "no session cookie provided")

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "unauthorized",
				"message": sanitizedUnauthorizedMessage,
				"errors":  fiber.Map{},
			})
		}

		// Look up session metadata in Redis.
		ctx := c.Context()
		sessionKey := sessionCacheKey(token)

		val, err := client.Get(ctx, sessionKey).Bytes()
		if err != nil {
			if err == redis.Nil {
				// Session not found in cache – likely expired or revoked.
				// Clean up the stale cookie to prevent repeated lookups.
				c.Cookie(&fiber.Cookie{
					Name:     CookieName,
					Value:    "",
					Path:     "/",
					Expires:  time.Now().Add(-24 * time.Hour),
					MaxAge:   -1,
					HTTPOnly: true,
					Secure:   true,
				})

				logUnauthorized(c, logger, "session_not_found", "session not found in cache")

				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"code":    "unauthorized",
					"message": sanitizedUnauthorizedMessage,
					"errors":  fiber.Map{},
				})
			}

			// Redis error (connection issue, etc.) – log and still fail fast.
			// We don't retry here; Redis unavailability is a temporary outage.
			logUnauthorized(c, logger, "redis_error", "failed to query session cache", zap.Error(err))

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "unauthorized",
				"message": sanitizedUnauthorizedMessage,
				"errors":  fiber.Map{},
			})
		}

		// Parse the session data from JSON (stored in Redis via session.Store.Cache).
		sessionData, parseErr := parseSessionData(val)
		if parseErr != nil {
			logUnauthorized(c, logger, "session_parse_error", "failed to parse session data", zap.Error(parseErr))

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "unauthorized",
				"message": sanitizedUnauthorizedMessage,
				"errors":  fiber.Map{},
			})
		}

		// Verify expiration – the cache is only valid if ExpiresAt is in the future.
		now := time.Now()
		if sessionData.ExpiresAt.IsZero() || !sessionData.ExpiresAt.After(now) {
			// Session has expired – clean up the cookie and respond 401.
			c.Cookie(&fiber.Cookie{
				Name:     CookieName,
				Value:    "",
				Path:     "/",
				Expires:  time.Now().Add(-24 * time.Hour),
				MaxAge:   -1,
				HTTPOnly: true,
				Secure:   true,
			})

			logUnauthorized(c, logger, "session_expired", "session has expired",
				zap.Time("expires_at", sessionData.ExpiresAt),
				zap.Time("now", now))

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "unauthorized",
				"message": sanitizedUnauthorizedMessage,
				"errors":  fiber.Map{},
			})
		}

		// Validate required fields are present.
		if sessionData.UserID == "" || sessionData.TenantID == "" {
			logUnauthorized(c, logger, "session_invalid", "session missing required fields",
				zap.String("user_id", sessionData.UserID),
				zap.String("tenant_id", sessionData.TenantID))

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "unauthorized",
				"message": sanitizedUnauthorizedMessage,
				"errors":  fiber.Map{},
			})
		}

		// Inject multi-tenant metadata into Fiber locals for downstream handlers.
		c.Locals("user_id", sessionData.UserID)
		c.Locals("tenant_id", sessionData.TenantID)

		// Also store the Stytch session ID for potential DB operations.
		c.Locals("stytch_session_id", sessionData.StytchSessionID)

		// Update the session's last_seen timestamp if update is needed.
		// This is a best-effort operation; failures are logged but don't affect auth.
		_ = client.Expire(ctx, sessionKey, sessionCacheTTL)

		return c.Next()
	}
}

// SessionData represents the structure stored in Redis for session validation.
type SessionData struct {
	UserID          string    `json:"user_id"`
	TenantID        string    `json:"tenant_id"`
	StytchSessionID string    `json:"stytch_session_id"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// parseSessionData deserializes the JSON payload from Redis into SessionData.
func parseSessionData(data []byte) (SessionData, error) {
	var sd SessionData
	if err := json.Unmarshal(data, &sd); err != nil {
		return SessionData{}, err
	}
	return sd, nil
}

// sessionCacheKey builds the Redis key for a session token.
// This matches the internal/session package's key format.
func sessionCacheKey(token string) string {
	return "session:" + token
}

// logUnauthorized logs security events related to session validation failures.
// It never logs the raw session token to avoid leaking sensitive data in logs.
func logUnauthorized(c fiber.Ctx, logger *zap.Logger, reason, message string, fields ...zap.Field) {
	if logger == nil {
		logger = zap.L()
	}

	// Build log entry with all relevant context.
	logFields := []zap.Field{
		zap.String("reason", reason),
		zap.String("message", message),
		zap.String("request_id", c.Get("X-Request-ID")),
		zap.String("remote_addr", c.IP()),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	}
	logFields = append(logFields, fields...)

	logger.Warn("session middleware: unauthorized", logFields...)
}

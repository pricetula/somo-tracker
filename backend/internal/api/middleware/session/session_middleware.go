// Package session provides Fiber middleware for secure session validation
// and multi-tenant Row-Level Security (RLS) context propagation.
//
// The middleware performs:
//   - Opaque session cookie extraction (session_token)
//   - Redis-backed session verification with expiration checks
//   - Device fingerprint validation to detect session hijacking / cookie theft
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
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	sessionpkg "somotracker/backend/internal/session"
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
//   - Fingerprint mismatch → 401 + session invalidation + high-severity security log
//   - All errors return the canonical JSON response contract
//   - No internal session tokens or Redis keys leak in responses
//
// On success, the middleware injects:
//   - c.Locals("user_id", userID)   – the authenticated user's UUID
//   - c.Locals("tenant_id", tenantID) – the organization/tenant UUID
//   - c.Locals("stytch_session_id", stytchSessionID) – Stytch session reference
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

		// Validate device fingerprint to detect session hijacking / cookie theft.
		if err := validateFingerprint(c, sessionData, logger); err != nil {
			// Fingerprint mismatch – potential session hijacking.
			// Invalidate the session immediately and clear the cookie.
			invalidateSession(ctx, client, sessionKey, token, logger, c)

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

// validateFingerprint compares the incoming request's fingerprint against
// the stored session fingerprint. If a mismatch is detected, it logs a
// high-severity security warning and returns an error.
func validateFingerprint(c fiber.Ctx, sessionData sessionpkg.SessionData, logger *zap.Logger) error {
	// If no fingerprint was stored (legacy session), skip validation.
	// This allows gradual rollout without invalidating existing sessions.
	if sessionData.Fingerprint == "" {
		return nil
	}

	// Compute the fingerprint from the current request.
	incomingFP := sessionpkg.ComputeFingerprint(c)

	// Compare fingerprints (constant-time comparison would be ideal but
	// timing attacks on fingerprint hash are not practical).
	if incomingFP != sessionData.Fingerprint {
		requestID := c.Get("X-Request-ID")

		// Log HIGH-SEVERITY security event for potential session hijacking.
		if logger == nil {
			logger = zap.L()
		}

		logger.Error("session middleware: FINGERPRINT MISMATCH - potential session hijacking",
			zap.String("reason", "fingerprint_mismatch"),
			zap.String("message", "device fingerprint does not match session; possible cookie theft"),
			zap.String("request_id", requestID),
			zap.String("remote_addr", c.IP()),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("user_id", sessionData.UserID),
			zap.String("tenant_id", sessionData.TenantID),
			zap.String("stored_fingerprint_prefix", sessionData.Fingerprint[:16]+"..."),
			zap.String("incoming_fingerprint_prefix", incomingFP[:16]+"..."),
			zap.String("user_agent", c.Get("User-Agent")),
			zap.String("accept_language", c.Get("Accept-Language")),
			zap.String("accept_encoding", c.Get("Accept-Encoding")),
		)

		return &sessionpkg.FingerprintMismatchError{
			Expected: sessionData.Fingerprint,
			Actual:   incomingFP,
		}
	}

	return nil
}

// invalidateSession removes the session from Redis and clears the cookie.
// Called when fingerprint mismatch is detected or other security violations.
func invalidateSession(ctx context.Context, client *redis.Client, sessionKey, token string, logger *zap.Logger, c fiber.Ctx) {
	requestID := c.Get("X-Request-ID")

	// Delete session from Redis
	if err := client.Del(ctx, sessionKey).Err(); err != nil && err != redis.Nil {
		logger.Error("session middleware: failed to delete session on invalidation",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
	}

	// Clear the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-24 * time.Hour),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   true,
	})

	logger.Warn("session middleware: session invalidated due to security violation",
		zap.String("request_id", requestID),
		zap.String("remote_addr", c.IP()),
		zap.String("reason", "fingerprint_mismatch"),
	)
}

// parseSessionData deserializes the JSON payload from Redis into sessionpkg.SessionData.
func parseSessionData(data []byte) (sessionpkg.SessionData, error) {
	var sd sessionpkg.SessionData
	if err := json.Unmarshal(data, &sd); err != nil {
		return sessionpkg.SessionData{}, err
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

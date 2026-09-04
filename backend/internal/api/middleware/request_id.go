// Package middleware provides HTTP middleware for the Somotracker API.
package middleware

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Context keys for request-scoped values.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

// RequestIDMiddleware extracts or generates a request ID and creates a
// request-scoped logger. It injects both into Fiber locals and the standard
// context.Context for downstream use.
func RequestIDMiddleware(baseLogger *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Extract X-Request-ID from headers or generate a new UUID.
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Set the response header.
		c.Set("X-Request-ID", reqID)

		// Create request-scoped logger with request_id field.
		reqLogger := baseLogger.With(zap.String("request_id", reqID))

		// Store in Fiber locals for easy access in handlers.
		c.Locals("request_id", reqID)
		c.Locals("logger", reqLogger)

		// Inject into standard context.Context for downstream sqlc and service calls.
		ctx := context.WithValue(c.Context(), requestIDKey, reqID)
		ctx = context.WithValue(ctx, loggerKey, reqLogger)
		c.SetContext(ctx)

		return c.Next()
	}
}

// NewRequestIDHandler creates the middleware handler for Uber Fx injection.
func NewRequestIDHandler(baseLogger *zap.Logger) fiber.Handler {
	return RequestIDMiddleware(baseLogger)
}

// RequestID returns the request ID from the standard context.
// Returns empty string if not present.
func RequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Logger returns the request-scoped logger from the standard context.
// Returns the base logger if not present.
func Logger(ctx context.Context) *zap.Logger {
	if v := ctx.Value(loggerKey); v != nil {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return zap.L() // fallback to global (should not happen in normal flow)
}

// GetRequestID is a convenience helper to extract the request ID from Fiber locals.
// Falls back to context if locals not set.
func GetRequestID(c fiber.Ctx) string {
	if v := c.Locals("request_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return RequestID(c.Context())
}

// GetLogger is a convenience helper to extract the request-scoped logger from Fiber locals.
// Falls back to context if locals not set.
func GetLogger(c fiber.Ctx) *zap.Logger {
	if v := c.Locals("logger"); v != nil {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return Logger(c.Context())
}

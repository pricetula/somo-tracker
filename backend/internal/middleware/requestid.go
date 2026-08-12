package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the wire name of the correlation ID header.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the Locals key under which NewRequestID stores the ID.
	RequestIDKey = "request_id"
)

// NewRequestID generates or propagates a correlation ID for every request:
//
//   - an incoming X-Request-ID header is honored when present and well-formed
//     (lets edge proxies / client-side tracing keep one ID across hops)
//   - otherwise a UUIDv4 is generated
//
// The ID is stored in locals under RequestIDKey, echoed in the X-Request-ID
// response header, and threaded into error responses and log lines so a
// single support ticket maps to one request across the whole stack.
//
// It must run before any middleware that logs or writes error responses
// (CSRF guard, session resolver, access log, HTTPError).
func NewRequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Get(RequestIDHeader))
		if !isValidRequestID(id) {
			id = uuid.NewString()
		}
		c.Locals(RequestIDKey, id)
		c.Set(RequestIDHeader, id)
		return c.Next()
	}
}

// GetRequestID returns the correlation ID stored by NewRequestID, or "" when
// the middleware hasn't run (e.g. unit tests constructing handlers directly).
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// withRequestID attaches the correlation ID to a response body when one is
// present, keeping error bodies traceable without breaking the canonical
// {code, message, errors} shape.
func withRequestID(c *fiber.Ctx, body fiber.Map) fiber.Map {
	if rid := GetRequestID(c); rid != "" {
		body["request_id"] = rid
	}
	return body
}

// isValidRequestID reports whether id is a safe header value: 1..128 chars of
// [A-Za-z0-9._-]. Anything else is rejected and replaced with a generated ID
// so a hostile header cannot smuggle control bytes into response headers.
func isValidRequestID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

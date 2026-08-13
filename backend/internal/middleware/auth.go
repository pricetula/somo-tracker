package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GetSession is a convenience function for extracting session info
// from the request context. It returns nil if the session is not set.
func GetSession(c *fiber.Ctx) *SessionInfo {
	session, ok := c.Locals("session").(*SessionInfo)
	if !ok {
		return nil
	}
	return session
}

// RequireAuth validates the session from context (loaded by global middleware)
// and sets tenant_id, user_id, and role on locals.
// For API routes only — does not fall back to cookie loading.
// Returns middleware.ErrUnauthorized if unauthenticated.
func RequireAuth(c *fiber.Ctx) error {
	session := GetSession(c)
	if session == nil {
		return HTTPError(c, ErrUnauthorized)
	}
	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	c.Locals("role", session.Role)
	return c.Next()
}

// RequireRole returns a middleware that authenticates and restricts access
// by role. Works for API routes where the global middleware has loaded the
// session. Returns 401 if unauthenticated, 403 if role is not permitted.
//
// Usage:
//
//	router.Patch("/:id", middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN"), h.PatchYear)
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Authenticate first — check session before inspecting role.
		// This ensures a missing session returns 401 (ErrUnauthorized)
		// rather than 403 (ErrForbidden).
		session := GetSession(c)
		if session == nil {
			return ErrUnauthorized
		}

		// Set locals (same as RequireAuth does) so downstream handlers
		// can access tenant_id, user_id, and role regardless of whether
		// RequireAuth was also registered.
		c.Locals("tenant_id", session.TenantID)
		c.Locals("user_id", session.UserID)
		c.Locals("role", session.Role)

		if len(roles) > 0 && !hasRole(session.Role, roles) {
			return ErrForbidden
		}
		return c.Next()
	}
}

// hasRole checks if a role matches any in the allowed list, case-insensitively.
func hasRole(role string, roles []string) bool {
	for _, r := range roles {
		if strings.EqualFold(role, r) {
			return true
		}
	}
	return false
}

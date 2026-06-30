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
// Returns 401 with canonical error body if unauthenticated.
func RequireAuth(c *fiber.Ctx) error {
	session := GetSession(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "authentication required",
		})
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
		if err := RequireAuth(c); err != nil {
			return err
		}
		if len(roles) > 0 {
			role, ok := c.Locals("role").(string)
			if !ok || !hasRole(role, roles) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"code":    "forbidden",
					"message": "insufficient permissions",
				})
			}
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

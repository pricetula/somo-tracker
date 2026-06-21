package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// AllowedRoles configures which roles are permitted.
type AllowedRoles struct {
	Roles []string // empty = any authenticated user
}

// RoleOption defines a functional option for RequireRole middleware.
type RoleOption func(*AllowedRoles)

// WithRoles restricts access to specific roles.
// If not called, any authenticated user is allowed.
func WithRoles(roles ...string) RoleOption {
	return func(a *AllowedRoles) {
		a.Roles = roles
	}
}

// RequireRole returns a Fiber middleware that enforces authentication and
// optional role-based access control. Works for both:
//   - /api/ routes (session loaded by the global middleware into c.Locals("session"))
//   - non-API routes (session loaded from cookie)
//
// Usage:
//
//	router.Get("/admin", middleware.RequireRole(pool, middleware.WithRoles("SYSTEM_ADMIN")))
//	router.Put("/schools/:id", middleware.RequireRole(pool, middleware.WithRoles("SYSTEM_ADMIN", "SCHOOL_ADMIN")))
func RequireRole(pools *database.Pools, opts ...RoleOption) fiber.Handler {
	cfg := &AllowedRoles{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *fiber.Ctx) error {
		// Try session from locals first (/api/ routes)
		session, ok := c.Locals("session").(*SessionInfo)
		if !ok || session == nil {
			// Fallback to cookie extraction (non-API routes)
			s, err := loadSessionFromCookie(c, pools.PG)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "unauthorized",
					"message": "authentication required",
				})
			}
			session = s
			c.Locals("session", session)
		}

		// If specific roles are required, check them
		if len(cfg.Roles) > 0 {
			allowed := false
			for _, role := range cfg.Roles {
				if strings.EqualFold(session.Role, role) {
					allowed = true
					break
				}
			}
			if !allowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "forbidden",
					"message": "insufficient permissions",
				})
			}
		}

		return c.Next()
	}
}

// loadSessionFromCookie reads the somo_sid cookie and looks up the session
// from Postgres. This is the standalone version used by the RequireRole
// middleware for non-API routes.
func loadSessionFromCookie(c *fiber.Ctx, pool *pgxpool.Pool) (*SessionInfo, error) {
	token := c.Cookies("somo_sid")
	if token == "" {
		return nil, fmt.Errorf("no session cookie")
	}

	const query = `
		SELECT s.user_id, s.tenant_id,
		       COALESCE(
		         (SELECT role::text FROM memberships
		           WHERE user_id = s.user_id AND is_active = true
		           ORDER BY
		             CASE role
		               WHEN 'SYSTEM_ADMIN' THEN 1
		               WHEN 'SCHOOL_ADMIN' THEN 2
		               WHEN 'TEACHER' THEN 3
		               WHEN 'NURSE' THEN 4
		               WHEN 'FINANCE' THEN 5
		             END
		           LIMIT 1),
		         'TEACHER'
		       ) as role
		FROM sessions s
		WHERE s.token = $1 AND s.expires_at > NOW()
	`

	var s SessionInfo
	err := pool.QueryRow(c.Context(), query, token).Scan(&s.UserID, &s.TenantID, &s.Role)
	if err != nil {
		return nil, fmt.Errorf("load session from cookie: %w", err)
	}

	return &s, nil
}

// RequireRoleCtx is a convenience function for extracting session info
// from the request context in handlers that use RequireRole middleware.
// It returns nil if the session is not set.
func GetSession(c *fiber.Ctx) *SessionInfo {
	session, ok := c.Locals("session").(*SessionInfo)
	if !ok {
		return nil
	}
	return session
}

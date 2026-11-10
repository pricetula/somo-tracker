package authz

import (
	"github.com/gofiber/fiber/v3"
	"somotracker/internal/platform/apperr"
)

// Permission represents a string permission identifier.
type Permission string

// Require checks that the current session has the required permission.
// Placeholder implementation: always allows for now.
// TODO: Implement per BE-020/BE-021 with role/permission matrix.
func Require(perm Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		// TODO: load session, tenant, membership and check permission.
		// For now, reject to enforce implementation.
		return apperr.New(apperr.FORBIDDEN, "authorization check not implemented")
	}
}

// ScopeFor returns the tenant-scoped context key for a resource.
// Placeholder.
func ScopeFor(resource string) string {
	return "scope:" + resource
}

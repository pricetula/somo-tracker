package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/httpserver"
)

// Permission represents a string permission identifier (BE-020).
type Permission string

// Role is a user role, lowercased database enum value (BE-025 naming).
type Role string

// Role constants aligned with the user_role enum (lowercased).
const (
	RoleOwner           Role = "owner"
	RoleAdmin           Role = "admin"
	RoleHeadteacher     Role = "headteacher"
	RoleTeacher         Role = "teacher"
	RoleGuardian        Role = "guardian"
	RoleFinance         Role = "finance"
	RolePlatformOperator Role = "platformoperator"
)

// Permission constants.
const (
	PermAuthRead        Permission = "auth:read"
	PermAuthLogout      Permission = "auth:logout"
	PermSchoolRead      Permission = "school:read"
	PermSchoolWrite     Permission = "school:write"
	PermUserRead        Permission = "user:read"
	PermUserWrite       Permission = "user:write"
	PermStudentRead     Permission = "student:read"
	PermStudentWrite    Permission = "student:write"
	PermAcademicRead    Permission = "academic:read"
	PermAcademicWrite   Permission = "academic:write"
	PermEnrollmentRead  Permission = "enrollment:read"
	PermEnrollmentWrite Permission = "enrollment:write"
	PermFeeRead         Permission = "fee:read"
	PermFeeWrite        Permission = "fee:write"
	PermHealthRead      Permission = "health:read"
	PermHealthWrite     Permission = "health:write"
	PermBehaviorRead    Permission = "behavior:read"
	PermBehaviorWrite   Permission = "behavior:write"
	PermAuditRead       Permission = "audit:read"
	PermSystemManage    Permission = "system:manage"
)

// rolePermissions is the permission matrix: role -> set of permissions.
// This is data (BE-020), not role lists scattered in handlers.
var rolePermissions = map[Role]map[Permission]struct{}{
	RoleOwner: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermSchoolRead:    {},
		PermSchoolWrite:   {},
		PermUserRead:      {},
		PermUserWrite:     {},
		PermStudentRead:   {},
		PermStudentWrite:  {},
		PermAcademicRead:  {},
		PermAcademicWrite: {},
		PermEnrollmentRead:  {},
		PermEnrollmentWrite: {},
		PermFeeRead:       {},
		PermFeeWrite:      {},
		PermHealthRead:    {},
		PermHealthWrite:   {},
		PermBehaviorRead:  {},
		PermBehaviorWrite: {},
		PermAuditRead:     {},
		PermSystemManage:  {},
	},
	RoleAdmin: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermSchoolRead:    {},
		PermUserRead:      {},
		PermUserWrite:     {},
		PermStudentRead:   {},
		PermStudentWrite:  {},
		PermAcademicRead:  {},
		PermAcademicWrite: {},
		PermEnrollmentRead:  {},
		PermEnrollmentWrite: {},
		PermFeeRead:       {},
		PermFeeWrite:      {},
		PermHealthRead:    {},
		PermHealthWrite:   {},
		PermBehaviorRead:  {},
		PermBehaviorWrite: {},
		PermAuditRead:     {},
	},
	RoleHeadteacher: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermSchoolRead:    {},
		PermUserRead:      {},
		PermStudentRead:   {},
		PermStudentWrite:  {},
		PermAcademicRead:  {},
		PermAcademicWrite: {},
		PermEnrollmentRead:  {},
		PermEnrollmentWrite: {},
		PermFeeRead:       {},
		PermHealthRead:    {},
		PermHealthWrite:   {},
		PermBehaviorRead:  {},
		PermBehaviorWrite: {},
	},
	RoleTeacher: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermSchoolRead:    {},
		PermStudentRead:   {},
		PermAcademicRead:  {},
		PermEnrollmentRead:  {},
		PermHealthRead:    {},
		PermBehaviorRead:  {},
	},
	RoleGuardian: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermStudentRead:   {}, // object-level: only linked students (enforced in service/store)
		PermHealthRead:    {}, // object-level: only linked students
		PermBehaviorRead:  {}, // object-level: only linked students
	},
	RoleFinance: {
		PermAuthRead:   {},
		PermAuthLogout: {},
		PermSchoolRead: {},
		PermFeeRead:    {},
		PermFeeWrite:   {},
	},
	RolePlatformOperator: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermSchoolRead:    {},
		PermSchoolWrite:   {},
		PermUserRead:      {},
		PermUserWrite:     {},
		PermStudentRead:   {},
		PermStudentWrite:  {},
		PermAcademicRead:  {},
		PermAcademicWrite: {},
		PermEnrollmentRead:  {},
		PermEnrollmentWrite: {},
		PermFeeRead:       {},
		PermFeeWrite:      {},
		PermHealthRead:    {},
		PermHealthWrite:   {},
		PermBehaviorRead:  {},
		PermBehaviorWrite: {},
		PermAuditRead:     {},
		PermSystemManage:  {},
	},
}

// Require checks that the current session has the required permission.
// Roles are read from c.Locals("roles") ([]string), set by AuthMiddleware.
// Returns 401 if no session, 403 if the session lacks the permission (BE-020).
func Require(perm Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		rolesRaw, ok := c.Locals("roles").([]string)
		if !ok || len(rolesRaw) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    string(apperr.UNAUTHENTICATED),
				Message: "authentication required",
			})
		}

		roles := make([]Role, 0, len(rolesRaw))
		for _, r := range rolesRaw {
			roles = append(roles, Role(strings.ToLower(r)))
		}

		if !hasPermission(roles, perm) {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    string(apperr.FORBIDDEN),
				Message: "insufficient permission",
			})
		}

		return c.Next()
	}
}

// ErrorResponse standardizes error shape for authz middleware responses.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ScopeFor returns the tenant-scoped context key for a permission (BE-021).
// Reads the tenant ID from request context (set by TenantMiddleware from the
// session, never from the client). Format: tenant:<tenant-id>:<perm>.
func ScopeFor(ctx context.Context, perm Permission) string {
	tenantID := tenantIDFromCtx(ctx)
	return fmt.Sprintf("tenant:%s:%s", tenantID, perm)
}

// HasPermission checks if any of the given roles has the permission.
func HasPermission(roles []Role, perm Permission) bool {
	for _, r := range roles {
		if perms, ok := rolePermissions[r]; ok {
			if _, has := perms[perm]; has {
				return true
			}
		}
	}
	return false
}

// hasPermission is the internal case-normalized check used by Require.
func hasPermission(roles []Role, perm Permission) bool {
	return HasPermission(roles, perm)
}

// tenantIDFromCtx extracts the tenant ID from request context.
// Supports both string and [16]byte representations set by TenantMiddleware.
func tenantIDFromCtx(ctx context.Context) string {
	v := ctx.Value(httpserver.TenantIDKey{})
	if v == nil {
		return ""
	}
	switch tid := v.(type) {
	case string:
		return tid
	case [16]byte:
		return fmtUUID(tid)
	default:
		return ""
	}
}

// fmtUUID formats a [16]byte as UUID string.
func fmtUUID(b [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Public marks a route group as public (no authentication required).
// Routes registered under a Public group bypass authz.Require.
type Public struct{}

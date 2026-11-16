package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/audit"
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

// Role semantics:
//   - Tenant-level roles: owner, platformoperator. These apply across all schools in a tenant.
//     "owner" is created during tenant provisioning and has full permissions.
//     "platformoperator" is a cross-tenant support role (same permissions as owner within a tenant).
//   - School-level roles: admin, headteacher, teacher, guardian, finance. These apply only within
//     the specific school(s) where the user has an active membership.
//   - Permissions are evaluated against the role(s) the user holds in the TARGET school, not the
//     union of roles across all schools. Use authz.RequireSchool() for school-scoped checks and
//     authz.Require() for tenant-level checks.
//   - The permission matrix (rolePermissions) is the single source of truth. Role-based middleware
//     helpers (RequireRole, RequireFinanceAccess, RequireAdminAccess) resolve through it, not
//     through duplicated logic.

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
	PermTimetableRead   Permission = "timetable:read"
	PermTimetableWrite  Permission = "timetable:write"
	PermAttendanceRead  Permission = "attendance:read"
	PermAttendanceWrite Permission = "attendance:write"
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
		PermTimetableRead: {},
		PermTimetableWrite: {},
		PermAttendanceRead: {},
		PermAttendanceWrite: {},
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
		PermTimetableRead: {},
		PermTimetableWrite: {},
		PermAttendanceRead: {},
		PermAttendanceWrite: {},
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
		PermTimetableRead: {},
		PermTimetableWrite: {},
		PermAttendanceRead: {},
		PermAttendanceWrite: {},
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
		PermTimetableRead: {},
		PermAttendanceRead: {},
		PermAttendanceWrite: {},
	},
	RoleGuardian: {
		PermAuthRead:      {},
		PermAuthLogout:    {},
		PermStudentRead:   {}, // object-level: only linked students (enforced in service/store)
		PermHealthRead:    {}, // object-level: only linked students
		PermBehaviorRead:  {}, // object-level: only linked students
		PermTimetableRead: {}, // object-level: only linked children's classes
		PermAttendanceRead: {}, // object-level: only linked children's attendance
	},
	RoleFinance: {
		PermAuthRead:   {},
		PermAuthLogout: {},
		PermSchoolRead: {},
		PermFeeRead:    {},
		PermFeeWrite:   {},
		PermAttendanceRead: {},
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
		PermTimetableRead: {},
		PermTimetableWrite: {},
		PermAttendanceRead: {},
		PermAttendanceWrite: {},
		PermAuditRead:     {},
		PermSystemManage:  {},
	},
}

// Require checks that the current session has the required permission.
// Roles are read from c.Locals("roles") ([]string), set by AuthMiddleware.
// Returns 401 if no session, 403 if the session lacks the permission (BE-020).
// An optional audit.Recorder can be passed to record 403 denials.
func Require(perm Permission, recorder ...*audit.Recorder) fiber.Handler {
	var rec *audit.Recorder
	if len(recorder) > 0 {
		rec = recorder[0]
	}

	return func(c fiber.Ctx) error {
		rolesRaw, ok := c.Locals("roles").([]string)
		if !ok || len(rolesRaw) == 0 {
			if rec != nil {
				rec.Record(c.Context(), audit.Event{
					Type:     audit.EventAuthorizationDenied,
					RequestID: c.Locals("request_id").(string),
					Outcome:  "denied",
					IP:       c.IP(),
					Details:  map[string]string{"permission": string(perm), "reason": "no_session"},
				})
			}
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
			if rec != nil {
				rec.Record(c.Context(), audit.Event{
					Type:     audit.EventAuthorizationDenied,
					RequestID: c.Locals("request_id").(string),
					Outcome:  "denied",
					IP:       c.IP(),
					Details:  map[string]string{"permission": string(perm), "reason": "insufficient_permission"},
				})
			}
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

// RequireSchool checks that the current session has the required permission
// in the target school. The target school is resolved from the route parameter
// (schoolIDParam) or from the session's primary school ("current").
// Permissions are evaluated against the role the user holds in the target school,
// not the union of roles across all schools (Phase 2: per-school role evaluation).
// An optional audit.Recorder can be passed to record 403 denials.
func RequireSchool(perm Permission, schoolIDParam string, recorder ...*audit.Recorder) fiber.Handler {
	var rec *audit.Recorder
	if len(recorder) > 0 {
		rec = recorder[0]
	}

	return func(c fiber.Ctx) error {
		rolesRaw, ok := c.Locals("roles").([]string)
		if !ok || len(rolesRaw) == 0 {
			if rec != nil {
				rec.Record(c.Context(), audit.Event{
					Type:     audit.EventAuthorizationDenied,
					RequestID: c.Locals("request_id").(string),
					Outcome:  "denied",
					IP:       c.IP(),
					Details:  map[string]string{"permission": string(perm), "reason": "no_session"},
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    string(apperr.UNAUTHENTICATED),
				Message: "authentication required",
			})
		}

		schoolIDsRaw, ok := c.Locals("school_ids").([]uuid.UUID)
		if !ok || len(schoolIDsRaw) == 0 {
			if rec != nil {
				rec.Record(c.Context(), audit.Event{
					Type:     audit.EventAuthorizationDenied,
					RequestID: c.Locals("request_id").(string),
					Outcome:  "denied",
					IP:       c.IP(),
					Details:  map[string]string{"permission": string(perm), "reason": "no_active_school_membership"},
				})
			}
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    string(apperr.FORBIDDEN),
				Message: "no active school membership",
			})
		}

		// Resolve target school ID
		var targetSchoolID uuid.UUID
		if schoolIDParam == "current" {
			targetSchoolID = schoolIDsRaw[0]
		} else {
			paramVal := c.Params(schoolIDParam)
			if paramVal == "" {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Code:    string(apperr.INVALID_REQUEST),
					Message: "school_id parameter required",
				})
			}
			var err error
			targetSchoolID, err = uuid.Parse(paramVal)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Code:    string(apperr.INVALID_REQUEST),
					Message: "invalid school_id",
				})
			}
		}

		// Find the role the user holds in the target school
		// rolesRaw and schoolIDsRaw are parallel arrays (index-aligned)
		var targetRole Role
		found := false
		for i, sid := range schoolIDsRaw {
			if sid == targetSchoolID {
				targetRole = Role(strings.ToLower(rolesRaw[i]))
				found = true
				break
			}
		}
		if !found {
			if rec != nil {
				rec.Record(c.Context(), audit.Event{
					Type:     audit.EventAuthorizationDenied,
					RequestID: c.Locals("request_id").(string),
					Outcome:  "denied",
					IP:       c.IP(),
					Details:  map[string]string{"permission": string(perm), "reason": "no_membership_in_target_school"},
				})
			}
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    string(apperr.FORBIDDEN),
				Message: "no membership in target school",
			})
		}

		// Check permission against the role in the target school only
		if !HasPermission([]Role{targetRole}, perm) {
			if rec != nil {
				rec.Record(c.Context(), audit.Event{
					Type:     audit.EventAuthorizationDenied,
					RequestID: c.Locals("request_id").(string),
					Outcome:  "denied",
					IP:       c.IP(),
					Details:  map[string]string{"permission": string(perm), "reason": "insufficient_permission_in_school"},
				})
			}
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Code:    string(apperr.FORBIDDEN),
				Message: "insufficient permission in this school",
			})
		}

		return c.Next()
	}
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

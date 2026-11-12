package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/httpserver"
)

// expectedPermissions defines the expected permission set per role for tests.
var expectedPermissions = map[Role][]Permission{
	RoleOwner: {
		PermAuthRead, PermAuthLogout,
		PermSchoolRead, PermSchoolWrite,
		PermUserRead, PermUserWrite,
		PermStudentRead, PermStudentWrite,
		PermAcademicRead, PermAcademicWrite,
		PermEnrollmentRead, PermEnrollmentWrite,
		PermFeeRead, PermFeeWrite,
		PermHealthRead, PermHealthWrite,
		PermBehaviorRead, PermBehaviorWrite,
		PermTimetableRead, PermTimetableWrite,
		PermAttendanceRead, PermAttendanceWrite,
		PermAuditRead,
		PermSystemManage,
	},
	RoleAdmin: {
		PermAuthRead, PermAuthLogout,
		PermSchoolRead,
		PermUserRead, PermUserWrite,
		PermStudentRead, PermStudentWrite,
		PermAcademicRead, PermAcademicWrite,
		PermEnrollmentRead, PermEnrollmentWrite,
		PermFeeRead, PermFeeWrite,
		PermHealthRead, PermHealthWrite,
		PermBehaviorRead, PermBehaviorWrite,
		PermTimetableRead, PermTimetableWrite,
		PermAttendanceRead, PermAttendanceWrite,
		PermAuditRead,
	},
	RoleHeadteacher: {
		PermAuthRead, PermAuthLogout,
		PermSchoolRead,
		PermUserRead,
		PermStudentRead, PermStudentWrite,
		PermAcademicRead, PermAcademicWrite,
		PermEnrollmentRead, PermEnrollmentWrite,
		PermFeeRead,
		PermHealthRead, PermHealthWrite,
		PermBehaviorRead, PermBehaviorWrite,
		PermTimetableRead, PermTimetableWrite,
		PermAttendanceRead, PermAttendanceWrite,
	},
	RoleTeacher: {
		PermAuthRead, PermAuthLogout,
		PermSchoolRead,
		PermStudentRead,
		PermAcademicRead,
		PermEnrollmentRead,
		PermHealthRead,
		PermBehaviorRead,
		PermTimetableRead,
		PermAttendanceRead,
		PermAttendanceWrite,
	},
	RoleGuardian: {
		PermAuthRead, PermAuthLogout,
		PermStudentRead,
		PermHealthRead,
		PermBehaviorRead,
		PermTimetableRead,
		PermAttendanceRead,
	},
	RoleFinance: {
		PermAuthRead, PermAuthLogout,
		PermSchoolRead,
		PermFeeRead, PermFeeWrite,
		PermAttendanceRead,
	},
	RolePlatformOperator: {
		PermAuthRead, PermAuthLogout,
		PermSchoolRead, PermSchoolWrite,
		PermUserRead, PermUserWrite,
		PermStudentRead, PermStudentWrite,
		PermAcademicRead, PermAcademicWrite,
		PermEnrollmentRead, PermEnrollmentWrite,
		PermFeeRead, PermFeeWrite,
		PermHealthRead, PermHealthWrite,
		PermBehaviorRead, PermBehaviorWrite,
		PermTimetableRead, PermTimetableWrite,
		PermAttendanceRead, PermAttendanceWrite,
		PermAuditRead,
		PermSystemManage,
	},
}

func TestRolePermissions_AllRolesHaveExpectedPermissions(t *testing.T) {
	for role, expected := range expectedPermissions {
		t.Run(string(role), func(t *testing.T) {
			for _, p := range expected {
				require.True(t, HasPermission([]Role{role}, p),
					"role %q should have permission %q", role, p)
			}
		})
	}
}

func TestRolePermissions_UnknownRoleHasNoPermissions(t *testing.T) {
	require.False(t, HasPermission([]Role{"unknown_role"}, PermStudentRead))
	require.False(t, HasPermission([]Role{"unknown_role"}, PermSchoolRead))
}

func TestRolePermissions_EmptyRoleSliceHasNoPermissions(t *testing.T) {
	require.False(t, HasPermission([]Role{}, PermStudentRead))
}

func TestRequire_AllowsWhenRoleHasPermission(t *testing.T) {
	for role, perms := range expectedPermissions {
		for _, perm := range perms {
			t.Run(string(role)+"/"+string(perm), func(t *testing.T) {
				app := fiber.New()
				app.Use(requirePermission(perm))
				app.Get("/test", func(c fiber.Ctx) error {
					return c.SendString("OK")
				})

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req = req.WithContext(context.WithValue(req.Context(), httpserver.TenantIDKey{}, "tenant-123"))
				// Simulate AuthMiddleware setting roles in locals
				// We need to set locals via a middleware before Require
				app2 := fiber.New()
				app2.Use(func(c fiber.Ctx) error {
					c.Locals("roles", []string{string(role)})
					return c.Next()
				})
				app2.Use(requirePermission(perm))
				app2.Get("/test", func(c fiber.Ctx) error {
					return c.SendString("OK")
				})

				resp, err := app2.Test(req, fiber.TestConfig{})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
	}
}

func TestRequire_DeniesWhenRoleLacksPermission(t *testing.T) {
	// teacher should not have student:write
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("roles", []string{"teacher"})
		return c.Next()
	})
	app.Use(Require(PermStudentWrite))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequire_DeniesWhenNoSession(t *testing.T) {
	app := fiber.New()
	app.Use(Require(PermStudentRead))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body := resp.Header.Get("Content-Type")
	require.Contains(t, body, "application/json")
}

func TestRequire_CaseInsensitiveRoleMatching(t *testing.T) {
	// Middleware may send "TEACHER" or "Teacher"; authz normalizes to lowercase.
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("roles", []string{"TEACHER"})
		return c.Next()
	})
	app.Use(Require(PermStudentRead))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestScopeFor_ReturnsTenantScopedKey(t *testing.T) {
	ctx := context.WithValue(context.Background(), httpserver.TenantIDKey{}, "tenant-123")
	key := ScopeFor(ctx, PermStudentRead)
	require.Equal(t, "tenant:tenant-123:student:read", key)
}

func TestScopeFor_ReturnsTenantScopedKey_WithUUIDBytes(t *testing.T) {
	var tid [16]byte
	copy(tid[:], []byte("tenant-uuid-12"))
	ctx := context.WithValue(context.Background(), httpserver.TenantIDKey{}, tid)
	key := ScopeFor(ctx, PermSchoolRead)
	require.Contains(t, key, "tenant:")
	require.Contains(t, key, ":school:read")
}

func TestScopeFor_EmptyTenantID(t *testing.T) {
	ctx := context.Background()
	key := ScopeFor(ctx, PermStudentRead)
	require.Equal(t, "tenant::student:read", key)
}

func TestHasPermission_DirectCall(t *testing.T) {
	require.True(t, HasPermission([]Role{RoleTeacher}, PermStudentRead))
	require.False(t, HasPermission([]Role{RoleTeacher}, PermStudentWrite))
	require.True(t, HasPermission([]Role{RoleGuardian}, PermStudentRead))
	require.False(t, HasPermission([]Role{RoleGuardian}, PermStudentWrite))
	require.False(t, HasPermission([]Role{RoleTeacher}, PermFeeWrite))
	require.True(t, HasPermission([]Role{RoleFinance}, PermFeeWrite))
}

// requirePermission is a thin wrapper for tests that need a bare permission check
// without the locals-setting middleware.
func requirePermission(perm Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		rolesRaw, ok := c.Locals("roles").([]string)
		if !ok || len(rolesRaw) == 0 {
			return apperr.New(apperr.UNAUTHENTICATED, "authentication required")
		}
		if !hasPermission(rolesToRoles(rolesRaw), perm) {
			return apperr.New(apperr.FORBIDDEN, "insufficient permission")
		}
		return c.Next()
	}
}

func rolesToRoles(roles []string) []Role {
	r := make([]Role, 0, len(roles))
	for _, s := range roles {
		r = append(r, Role(s))
	}
	return r
}

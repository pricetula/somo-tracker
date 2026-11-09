package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/config"
)

// MockSessionValidator for RBAC tests - implements all required methods
type mockRBACValidator struct {
	memberships      []service.MembershipContext
	guardianKids     []uuid.UUID
	teacherClasses   []uuid.UUID
}

func (m *mockRBACValidator) ValidateSession(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
	return &service.SessionContext{
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		SessionID: uuid.New(),
		Role:      "tenant_owner",
	}, nil
}

func (m *mockRBACValidator) ValidateSessionWithMemberships(ctx context.Context, tokenHash []byte) (*service.SessionContext, []service.MembershipContext, error) {
	sess, _ := m.ValidateSession(ctx, tokenHash)
	mems, _ := m.GetUserMemberships(ctx, sess.UserID, sess.TenantID)
	return sess, mems, nil
}

func (m *mockRBACValidator) GetUserMemberships(ctx context.Context, userID, tenantID uuid.UUID) ([]service.MembershipContext, error) {
	return m.memberships, nil
}

func (m *mockRBACValidator) GetGuardianStudents(ctx context.Context, guardianUserID, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return m.guardianKids, nil
}

func (m *mockRBACValidator) GetTeacherClasses(ctx context.Context, teacherUserID, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return m.teacherClasses, nil
}

func (m *mockRBACValidator) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return nil
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name        string
		userRoles   []string
		required    []string
		wantStatus  int
	}{
		{"single role match", []string{"school_admin"}, []string{"school_admin"}, http.StatusOK},
		{"multiple roles, one matches", []string{"teacher", "guardian"}, []string{"school_admin", "finance_officer"}, http.StatusForbidden},
		{"multiple roles, one matches (finance)", []string{"teacher", "finance_officer"}, []string{"school_admin", "finance_officer"}, http.StatusOK},
		{"tenant_owner in required", []string{"tenant_owner"}, []string{"tenant_owner", "school_admin"}, http.StatusOK},
		{"no matching role", []string{"teacher"}, []string{"school_admin", "finance_officer"}, http.StatusForbidden},
		{"empty user roles", []string{}, []string{"school_admin"}, http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schoolID := uuid.New()
			mockSvc := &mockRBACValidator{
				memberships: []service.MembershipContext{},
			}
			for _, r := range tc.userRoles {
				mockSvc.memberships = append(mockSvc.memberships, service.MembershipContext{
					SchoolID: schoolID,
					Role:     r,
					IsActive: true,
				})
			}

			auth := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})
			rbac := NewRBACMiddleware(mockSvc)

			app := fiber.New()
			app.Use(auth.RequireAuth())
			app.Use(rbac.RequireRole(tc.required...))
			app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

			token := "test-token"
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{Name: "session_token_test", Value: token})

			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestRequireAllRoles(t *testing.T) {
	tests := []struct {
		name        string
		userRoles   []string
		required    []string
		wantStatus  int
	}{
		{"all roles present", []string{"school_admin", "finance_officer"}, []string{"school_admin", "finance_officer"}, http.StatusOK},
		{"missing one role", []string{"school_admin"}, []string{"school_admin", "finance_officer"}, http.StatusForbidden},
		{"extra roles ok", []string{"school_admin", "finance_officer", "tenant_owner"}, []string{"school_admin", "finance_officer"}, http.StatusOK},
		{"empty required", []string{"school_admin"}, []string{}, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schoolID := uuid.New()
			mockSvc := &mockRBACValidator{}
			for _, r := range tc.userRoles {
				mockSvc.memberships = append(mockSvc.memberships, service.MembershipContext{
					SchoolID: schoolID,
					Role:     r,
					IsActive: true,
				})
			}

			auth := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})
			rbac := NewRBACMiddleware(mockSvc)

			app := fiber.New()
			app.Use(auth.RequireAuth())
			app.Use(rbac.RequireAllRoles(tc.required...))
			app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

			token := "test-token"
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{Name: "session_token_test", Value: token})

			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestRequireFinanceAccess(t *testing.T) {
	tests := []struct {
		name       string
		userRole   string
		wantStatus int
	}{
		{"finance_officer allowed", "finance_officer", http.StatusOK},
		{"school_admin allowed", "school_admin", http.StatusOK},
		{"tenant_owner allowed", "tenant_owner", http.StatusOK},
		{"teacher denied", "teacher", http.StatusForbidden},
		{"guardian denied", "guardian", http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockRBACValidator{
				memberships: []service.MembershipContext{
					{SchoolID: uuid.New(), Role: tc.userRole, IsActive: true},
				},
			}
			auth := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})
			rbac := NewRBACMiddleware(mockSvc)

			app := fiber.New()
			app.Use(auth.RequireAuth())
			app.Use(rbac.RequireFinanceAccess())
			app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

			token := "test-token"
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{Name: "session_token_test", Value: token})

			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestRequireAdminAccess(t *testing.T) {
	tests := []struct {
		name       string
		userRole   string
		wantStatus int
	}{
		{"school_admin allowed", "school_admin", http.StatusOK},
		{"tenant_owner allowed", "tenant_owner", http.StatusOK},
		{"finance_officer denied", "finance_officer", http.StatusForbidden},
		{"teacher denied", "teacher", http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockRBACValidator{
				memberships: []service.MembershipContext{
					{SchoolID: uuid.New(), Role: tc.userRole, IsActive: true},
				},
			}
			auth := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})
			rbac := NewRBACMiddleware(mockSvc)

			app := fiber.New()
			app.Use(auth.RequireAuth())
			app.Use(rbac.RequireAdminAccess())
			app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

			token := "test-token"
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{Name: "session_token_test", Value: token})

			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

// Note: Object-level access tests (RequireSchoolAccess, RequireGuardianAccess, RequireTeacherAccess)
// require proper route parameter parsing which is not reliably available in middleware during Fiber tests.
// These are tested via integration tests in the handler package.
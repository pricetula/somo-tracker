package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/config"
)

// MockSessionValidator implements SessionValidator for testing
type MockSessionValidator struct {
	validateSessionFunc func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error)
}

func (m *MockSessionValidator) ValidateSession(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
	if m.validateSessionFunc != nil {
		return m.validateSessionFunc(ctx, tokenHash)
	}
	return nil, service.ErrSessionNotFound
}

// TestRequireAuth_CookieNamePerEnv verifies middleware reads correct cookie name per env
func TestRequireAuth_CookieNamePerEnv(t *testing.T) {
	tests := []struct {
		name           string
		env            string
		cookieName     string
		wantStatus     int
		setupMock      func(*MockSessionValidator)
	}{
		{
			name:       "valid cookie with correct env suffix",
			env:        "staging",
			cookieName: "__Secure-session_token_staging",
			wantStatus: http.StatusOK,
			setupMock: func(m *MockSessionValidator) {
				m.validateSessionFunc = func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
					return &service.SessionContext{
						UserID:    uuid.New(),
						TenantID:  uuid.New(),
						SessionID: uuid.New(),
						Role:      "tenant_owner",
					}, nil
				}
			},
		},
		{
			name:       "missing cookie",
			env:        "staging",
			cookieName: "",
			wantStatus: http.StatusUnauthorized,
			setupMock:  nil,
		},
		{
			name:       "wrong cookie name (no env suffix in staging)",
			env:        "staging",
			cookieName: "__Secure-session_token",
			wantStatus: http.StatusUnauthorized,
			setupMock:  nil,
		},
		{
			name:       "production env uses base name",
			env:        "production",
			cookieName: "__Secure-session_token",
			wantStatus: http.StatusOK,
			setupMock: func(m *MockSessionValidator) {
				m.validateSessionFunc = func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
					return &service.SessionContext{
						UserID:    uuid.New(),
						TenantID:  uuid.New(),
						SessionID: uuid.New(),
						Role:      "tenant_owner",
					}, nil
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &MockSessionValidator{}
			if tc.setupMock != nil {
				tc.setupMock(mockSvc)
			}

			middleware := NewAuthMiddleware(mockSvc, &config.Config{Env: tc.env})

			app := fiber.New()
			app.Use(middleware.RequireAuth())
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.cookieName != "" {
				token := "valid-token-123"
				req.AddCookie(&http.Cookie{
					Name:  tc.cookieName,
					Value: token,
				})
			}

			resp, err := app.Test(req, fiber.TestConfig{})
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

// TestRequireAuth_ValidatesExpiryRevocation verifies session expiry and revocation checks
func TestRequireAuth_ValidatesExpiryRevocation(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		wantStatus  int
		wantCode    string
	}{
		{
			name:        "valid session",
			mockError:   nil,
			wantStatus:  http.StatusOK,
			wantCode:    "",
		},
		{
			name:        "session not found",
			mockError:   service.ErrSessionNotFound,
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "UNAUTHENTICATED",
		},
		{
			name:        "expired session",
			mockError:   service.ErrSessionNotFound, // Same error for expired
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "UNAUTHENTICATED",
		},
		{
			name:        "revoked session",
			mockError:   service.ErrSessionNotFound, // Same error for revoked
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "UNAUTHENTICATED",
		},
		{
			name:        "internal error",
			mockError:   errors.New("db connection failed"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "INTERNAL_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &MockSessionValidator{
				validateSessionFunc: func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
					if tc.mockError != nil {
						return nil, tc.mockError
					}
					return &service.SessionContext{
						UserID:    uuid.New(),
						TenantID:  uuid.New(),
						SessionID: uuid.New(),
						Role:      "tenant_owner",
					}, nil
				},
			}

			middleware := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})

			app := fiber.New()
			app.Use(middleware.RequireAuth())
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			token := "valid-token-123"
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{
				Name:  "__Secure-session_token_test",
				Value: token,
			})

			resp, err := app.Test(req, fiber.TestConfig{})
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			if tc.wantCode != "" {
				bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
				if !contains(body, tc.wantCode) {
					t.Errorf("response body missing code %q: %s", tc.wantCode, body)
				}
			}
		})
	}
}

// TestRequireAuth_SetsLocals verifies session context is stored in locals
func TestRequireAuth_SetsLocals(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	sessionID := uuid.New()
	role := "tenant_owner"

	mockSvc := &MockSessionValidator{
		validateSessionFunc: func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
			return &service.SessionContext{
				UserID:    userID,
				TenantID:  tenantID,
				SessionID: sessionID,
				Role:      role,
			}, nil
		},
	}

	middleware := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})

	app := fiber.New()
	app.Use(middleware.RequireAuth())
	app.Get("/test", func(c fiber.Ctx) error {
		// Verify locals are set
		if c.Locals("user_id") != userID {
			return c.Status(http.StatusInternalServerError).SendString("user_id not set")
		}
		if c.Locals("tenant_id") != tenantID {
			return c.Status(http.StatusInternalServerError).SendString("tenant_id not set")
		}
		if c.Locals("session_id") != sessionID {
			return c.Status(http.StatusInternalServerError).SendString("session_id not set")
		}
		if c.Locals("role") != role {
			return c.Status(http.StatusInternalServerError).SendString("role not set")
		}
		return c.SendString("OK")
	})

	token := "valid-token-123"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "__Secure-session_token_test",
		Value: token,
	})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestCSRFMiddleware_RejectsMissingOrigin verifies CSRF protection for mutations
func TestCSRFMiddleware_RejectsMissingOrigin(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		secFetchSite   string
		wantStatus     int
	}{
		{
			name:         "GET allowed without Origin",
			method:       http.MethodGet,
			origin:       "",
			secFetchSite: "",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "HEAD allowed without Origin",
			method:       http.MethodHead,
			origin:       "",
			secFetchSite: "",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "OPTIONS allowed without Origin",
			method:       http.MethodOptions,
			origin:       "",
			secFetchSite: "",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "POST missing Origin, not same-origin -> 403",
			method:       http.MethodPost,
			origin:       "",
			secFetchSite: "cross-site",
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "POST missing Origin, same-origin -> allowed",
			method:       http.MethodPost,
			origin:       "",
			secFetchSite: "same-origin",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "POST with allowed Origin -> allowed",
			method:       http.MethodPost,
			origin:       "https://app.example.com",
			secFetchSite: "",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "POST with disallowed Origin -> 403",
			method:       http.MethodPost,
			origin:       "https://evil.com",
			secFetchSite: "",
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "PUT with allowed Origin -> allowed",
			method:       http.MethodPut,
			origin:       "https://app.example.com",
			secFetchSite: "",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "DELETE with allowed Origin -> allowed",
			method:       http.MethodDelete,
			origin:       "https://app.example.com",
			secFetchSite: "",
			wantStatus:   http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &MockSessionValidator{
				validateSessionFunc: func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
					return &service.SessionContext{
						UserID:    uuid.New(),
						TenantID:  uuid.New(),
						SessionID: uuid.New(),
						Role:      "tenant_owner",
					}, nil
				},
			}
			middleware := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})

			app := fiber.New()
			app.Use(middleware.RequireAuth())
			app.Use(middleware.CSRFMiddleware())

			// Use Add with method and path
			app.Add([]string{tc.method}, "/test", func(c fiber.Ctx) error {
				return c.SendString("OK")
			})

			token := "valid-token-123"
			req := httptest.NewRequest(tc.method, "/test", nil)
			req.AddCookie(&http.Cookie{
				Name:  "__Secure-session_token_test",
				Value: token,
			})
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}

			resp, err := app.Test(req, fiber.TestConfig{})
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d (method=%s, origin=%s, secFetchSite=%s)",
					resp.StatusCode, tc.wantStatus, tc.method, tc.origin, tc.secFetchSite)
			}
		})
	}
}

// TestOptionalAuth_AllowsMissingCookie verifies optional auth doesn't require cookie
func TestOptionalAuth_AllowsMissingCookie(t *testing.T) {
	mockSvc := &MockSessionValidator{}
	middleware := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})

	app := fiber.New()
	app.Use(middleware.OptionalAuth())
	app.Get("/test", func(c fiber.Ctx) error {
		// Should proceed without auth
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No cookie

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (optional auth should allow missing cookie)", resp.StatusCode, http.StatusOK)
	}
}

// TestOptionalAuth_ValidatesCookieIfPresent verifies optional auth validates cookie when present
func TestOptionalAuth_ValidatesCookieIfPresent(t *testing.T) {
	mockSvc := &MockSessionValidator{
		validateSessionFunc: func(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
			return nil, service.ErrSessionNotFound // Invalid session
		},
	}
	middleware := NewAuthMiddleware(mockSvc, &config.Config{Env: "test"})

	app := fiber.New()
	app.Use(middleware.OptionalAuth())
	app.Get("/test", func(c fiber.Ctx) error {
		// Should proceed even with invalid cookie (optional)
		return c.SendString("OK")
	})

	token := "invalid-token"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "__Secure-session_token_test",
		Value: token,
	})

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (optional auth should allow invalid cookie)", resp.StatusCode, http.StatusOK)
	}
}

// TestSessionCookieName_EnvSuffix verifies middleware cookie name matches handler
func TestSessionCookieName_EnvSuffix(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		wantCookie string
	}{
		{"development", "development", "__Secure-session_token_development"},
		{"staging", "staging", "__Secure-session_token_staging"},
		{"production", "production", "__Secure-session_token"},
		{"empty", "", "__Secure-session_token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &AuthMiddleware{cfg: &config.Config{Env: tc.env}}
			got := m.sessionCookieName()
			if got != tc.wantCookie {
				t.Errorf("sessionCookieName() = %q, want %q", got, tc.wantCookie)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


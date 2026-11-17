//go:build integration

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/stytch"
)

// mockStytchPKCE implements stytch.Client for PKCE flow testing.
type mockStytchPKCE struct {
	authenticateShouldFail bool
}

func (m *mockStytchPKCE) SendDiscoveryMagicLink(ctx context.Context, email, loginURL, challenge string) error {
	return nil
}

func (m *mockStytchPKCE) AuthenticateDiscovery(ctx context.Context, token, pkceCodeVerifier string) (*stytch.DiscoveryAuthResponse, error) {
	if m.authenticateShouldFail {
		return nil, &stytchErr{msg: "invalid token"}
	}
	return &stytch.DiscoveryAuthResponse{
		Email:                      "test@example.com",
		IntermediateSessionToken:   "test-ist",
		DiscoveredOrganizations:    []stytch.DiscoveredOrg{},
	}, nil
}

func (m *mockStytchPKCE) ExchangeDiscoverySession(ctx context.Context, ist string) (*stytch.DiscoveryExchangeResponse, error) {
	return nil, nil
}

func (m *mockStytchPKCE) CreateOrganizationAndMember(ctx context.Context, name, email string) (*stytch.OrgMemberResponse, error) {
	return nil, nil
}

func (m *mockStytchPKCE) RevokeMemberSession(ctx context.Context, memberID, sessionID string) error {
	return nil
}

type stytchErr struct {
	msg string
}

func (e *stytchErr) Error() string { return e.msg }

// setupPKCEApp creates a Fiber app wired with the identity auth handlers
// and a mock Stytch client, suitable for testing the PKCE cookie flow.
func setupPKCEApp(t *testing.T, env string) (*fiber.App, *Handler) {
	t.Helper()
	svc, err := service.New(nil, &mockStytchPKCE{}, nil, "", 0)
	require.NoError(t, err, "service.New")

	cfg := &config.Config{Env: env}
	h := New(svc, cfg, nil, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/auth/request-magic-link", h.RequestMagicLink)
	app.Post("/api/v1/auth/authenticate-token", h.AuthenticateToken)

	return app, h
}

// extractCookieValue pulls the value of a named cookie from a Set-Cookie header.
func extractCookieValue(t *testing.T, resp *httptest.ResponseRecorder, name string) string {
	t.Helper()
	setCookie := resp.Header().Get("Set-Cookie")
	require.NotEmpty(t, setCookie, "expected Set-Cookie header")
	for _, part := range strings.Split(setCookie, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, name+"=") {
			return strings.TrimPrefix(part, name+"=")
		}
	}
	require.Fail(t, "cookie %q not found in Set-Cookie: %s", name, setCookie)
	return ""
}

// TestPKCE_FullFlow tests the complete PKCE magic-link flow:
//
//	1. Request magic link → backend sets pkce_verifier cookie
//	2. Authenticate without cookie → 400 MISSING_PKCE
//	3. Authenticate with cookie → 200, cookie cleared
func TestPKCE_FullFlow(t *testing.T) {
	app, _ := setupPKCEApp(t, "development")

	// --- Step 1: Request magic link, expect pkce_verifier cookie to be set ---
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-magic-link", strings.NewReader(`{"email":"test@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	app.Handler()(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, "request-magic-link should return 200")
	verifier := extractCookieValue(t, resp, "pkce_verifier")
	require.NotEmpty(t, verifier, "pkce_verifier cookie should be set")

	// --- Step 2: Authenticate without cookie → 400 MISSING_PKCE ---
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/authenticate-token", strings.NewReader(`{"token":"mock-token"}`))
	req2.Header.Set("Content-Type", "application/json")
	resp2 := httptest.NewRecorder()
	app.Handler()(resp2, req2)

	require.Equal(t, http.StatusBadRequest, resp2.Code, "authenticate-token without cookie should return 400")
	body2 := resp2.Body.String()
	require.Contains(t, body2, "MISSING_PKCE", "response should contain MISSING_PKCE error code")

	// --- Step 3: Authenticate with cookie → 200, cookie cleared ---
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/authenticate-token", strings.NewReader(`{"token":"mock-token"}`))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Cookie", "pkce_verifier="+verifier)
	resp3 := httptest.NewRecorder()
	app.Handler()(resp3, req3)

	require.Equal(t, http.StatusOK, resp3.Code, "authenticate-token with cookie should return 200")
	body3 := resp3.Body.String()
	require.Contains(t, body3, "ticket_token", "response should contain ticket_token")

	// Cookie should be cleared after successful authentication
	setCookie3 := resp3.Header().Get("Set-Cookie")
	require.Contains(t, setCookie3, "pkce_verifier=", "should clear pkce_verifier cookie")
	require.Contains(t, setCookie3, "Max-Age=-1", "cleared cookie should have Max-Age=-1")
}

// TestPKCE_Production_Uses_Host_Prefix verifies that in production the cookie
// is __Host-pkce_verifier (not pkce_verifier) and carries the Secure flag.
func TestPKCE_Production_Uses_Host_Prefix(t *testing.T) {
	app, _ := setupPKCEApp(t, "production")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-magic-link", strings.NewReader(`{"email":"test@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	resp := httptest.NewRecorder()
	app.Handler()(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	setCookie := resp.Header().Get("Set-Cookie")

	// Should use __Host- prefix in production
	require.Contains(t, setCookie, "__Host-pkce_verifier=", "production should use __Host-pkce_verifier")
	// Should not use the dev-only name
	require.NotContains(t, setCookie, "pkce_verifier=", "dev cookie name should not appear in production")
	// Should be Secure
	require.Contains(t, setCookie, "Secure", "production cookie should be Secure")
	// Should be HttpOnly
	require.Contains(t, setCookie, "HttpOnly", "production cookie should be HttpOnly")
	// Should have Path=/
	require.Contains(t, setCookie, "Path=/", "production cookie should have Path=/")
}

// TestPKCE_AuthenticateFailure_StillReturnsError verifies that when Stytch
// authentication fails, the handler returns an error but still clears the
// PKCE cookie.
func TestPKCE_AuthenticateFailure_ClearsCookie(t *testing.T) {
	svc, err := service.New(nil, &mockStytchPKCE{authenticateShouldFail: true}, nil, "", 0)
	require.NoError(t, err)
	h := New(svc, &config.Config{Env: "development"}, nil, nil)

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/api/v1/auth/request-magic-link", h.RequestMagicLink)
	app.Post("/api/v1/auth/authenticate-token", h.AuthenticateToken)

	// Get a verifier first
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-magic-link", strings.NewReader(`{"email":"test@example.com"}`))
	req1.Header.Set("Content-Type", "application/json")
	resp1 := httptest.NewRecorder()
	app.Handler()(resp1, req1)
	require.Equal(t, http.StatusOK, resp1.Code)
	verifier := extractCookieValue(t, resp1, "pkce_verifier")

	// Authenticate with the cookie → Stytch fails → 401 or 500, but cookie cleared
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/authenticate-token", strings.NewReader(`{"token":"bad-token"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Cookie", "pkce_verifier="+verifier)
	resp2 := httptest.NewRecorder()
	app.Handler()(resp2, req2)

	require.NotEqual(t, http.StatusOK, resp2.Code, "should not succeed with bad token")
	setCookie2 := resp2.Header().Get("Set-Cookie")
	require.Contains(t, setCookie2, "Max-Age=-1", "cookie should be cleared even on auth failure")
}

// TestPKCE_MISSING_PKCE_StatusCode verifies the registered HTTP status for MISSING_PKCE.
func TestPKCE_MISSING_PKCE_StatusCode(t *testing.T) {
	require.Equal(t, http.StatusBadRequest, apperr.Status(apperr.MISSING_PKCE))
}

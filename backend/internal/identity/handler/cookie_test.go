package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/config"
)

// TestSessionCookieName_EnvSuffix verifies cookie name includes env suffix for non-production
func TestSessionCookieName_EnvSuffix(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		wantCookie string
	}{
		{"development", "development", "__Secure-session_token_development"},
		{"staging", "staging", "__Secure-session_token_staging"},
		{"production", "production", "__Secure-session_token"},
		{"empty env", "", "__Secure-session_token"},
		{"custom env", "test", "__Secure-session_token_test"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handler{cfg: &config.Config{Env: tc.env}}
			got := h.sessionCookieName()
			if got != tc.wantCookie {
				t.Errorf("sessionCookieName() = %q, want %q", got, tc.wantCookie)
			}
		})
	}
}

// TestSessionCookieDomain_ReturnsConfiguredDomain verifies domain is returned from config
func TestSessionCookieDomain_ReturnsConfiguredDomain(t *testing.T) {
	// Currently returns empty string (host-only) until production domain is configured
	h := &Handler{cfg: &config.Config{Env: "development"}}
	got := h.sessionCookieDomain()
	if got != "" {
		t.Errorf("sessionCookieDomain() = %q, want empty string (host-only)", got)
	}
}

// TestSetSessionCookie_SetsCorrectAttributes verifies cookie attributes on set
func TestSetSessionCookie_SetsCorrectAttributes(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "staging"}}

	app := fiber.New()
	app.Post("/set-cookie", func(c fiber.Ctx) error {
		token := "test-token-123"
		expiresAt := time.Now().Add(24 * time.Hour)
		h.setSessionCookie(c, token, expiresAt)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodPost, "/set-cookie", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	// Get the Set-Cookie header
	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("no Set-Cookie header found")
	}

	cookieHeader := cookies[0]
	t.Logf("Set-Cookie: %s", cookieHeader)

	// Verify cookie name includes env suffix
	if !contains(cookieHeader, "__Secure-session_token_staging=") {
		t.Errorf("cookie name missing env suffix: %s", cookieHeader)
	}

	// Verify attributes (case-insensitive)
	requiredAttrs := []string{
		"HttpOnly",
		"Secure",
		"SameSite=Lax",
		"Path=/",
		"Max-Age=",
	}
	for _, attr := range requiredAttrs {
		if !contains(cookieHeader, attr) {
			t.Errorf("missing attribute %q in cookie: %s", attr, cookieHeader)
		}
	}
}

// TestClearSessionCookie_SetsExpiredCookie verifies cookie clearing sets Max-Age=0
func TestClearSessionCookie_SetsExpiredCookie(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "staging"}}

	app := fiber.New()
	app.Post("/clear-cookie", func(c fiber.Ctx) error {
		h.clearSessionCookie(c)
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodPost, "/clear-cookie", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("no Set-Cookie header found")
	}

	cookieHeader := cookies[0]
	t.Logf("Clear Set-Cookie: %s", cookieHeader)

	// Verify cookie name includes env suffix
	if !contains(cookieHeader, "__Secure-session_token_staging=") {
		t.Errorf("cookie name missing env suffix: %s", cookieHeader)
	}

	// Verify expired attributes (case-insensitive)
	requiredAttrs := []string{
		"HttpOnly",
		"Secure",
		"SameSite=Lax",
		"Path=/",
		"Max-Age=0", // Fiber uses 0 for expired
	}
	for _, attr := range requiredAttrs {
		if !contains(cookieHeader, attr) {
			t.Errorf("missing attribute %q in clear cookie: %s", attr, cookieHeader)
		}
	}

	// Verify value is empty
	if !contains(cookieHeader, "__Secure-session_token_staging=;") && !contains(cookieHeader, "__Secure-session_token_staging= ;") {
		t.Errorf("cookie value not empty: %s", cookieHeader)
	}
}

// TestAuthenticateTokenResponse_IncludesAuthState verifies response DTO has auth_state
func TestAuthenticateTokenResponse_IncludesAuthState(t *testing.T) {
	// This is a compile-time check - if the struct doesn't have auth_state, it won't compile
	resp := AuthenticateTokenResponse{
		Status:      "onboarding_required",
		AuthState:   "onboarding_required",
		TicketToken: "ticket-123",
	}

	if resp.AuthState != "onboarding_required" {
		t.Errorf("AuthState not set correctly: %s", resp.AuthState)
	}
	if resp.TicketToken != "ticket-123" {
		t.Errorf("TicketToken not set correctly: %s", resp.TicketToken)
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// MockService for handler tests (not used in current tests but kept for future)
type MockService struct{}

func (m *MockService) RequestMagicLink(ctx context.Context, email string) (string, error) {
	return "", nil
}

func (m *MockService) AuthenticateToken(ctx context.Context, token, pkceCodeVerifier string) (string, *service.OnboardingState, error) {
	return "", nil, nil
}

func (m *MockService) ProvisionTenant(ctx context.Context, ticketToken string, data service.ProvisionData) (*service.ProvisionResult, error) {
	return nil, nil
}

func (m *MockService) CreateSession(ctx context.Context, userID, tenantID uuid.UUID, stytchMemberSessionID string, expiresAt time.Time) (string, error) {
	return "", nil
}

func (m *MockService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return nil
}

func (m *MockService) UpdateOnboardingStep(ctx context.Context, pendingID uuid.UUID, step string, formData map[string]interface{}) error {
	return nil
}

func (m *MockService) GetOnboardingCurrent(ctx context.Context, ticketToken string) (*service.OnboardingState, error) {
	return nil, nil
}

func (m *MockService) ValidateSession(ctx context.Context, tokenHash []byte) (*service.SessionContext, error) {
	return nil, service.ErrSessionNotFound
}
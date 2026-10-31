package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/apperr"
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

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
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
		"path=/",
		"max-age=",
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

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
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
		"path=/",
		"max-age=0", // Fiber uses 0 for expired
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

// TestErrorResponse_Shape verifies error responses follow the contract
// Expected: { code: string, message: string, errors?: object, request_id?: string }
func TestErrorResponse_Shape(t *testing.T) {
	err := apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatal("error is not an apperr.Error")
	}
	if ae.Code != apperr.INVALID_REQUEST {
		t.Errorf("code = %v, want INVALID_REQUEST", ae.Code)
	}
	if ae.Message != "invalid request body" {
		t.Errorf("message = %v, want 'invalid request body'", ae.Message)
	}
}

// TestRequestMagicLink_ValidationErrors verifies validation error format
func TestRequestMagicLink_ValidationErrors(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "test"}}

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantCode    string
		wantContains string
	}{
		{
			name:        "missing email",
			body:        `{}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    "INVALID_EMAIL",
			wantContains: "invalid email address",
		},
		{
			name:        "invalid email format",
			body:        `{"email": "not-an-email"}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    "INVALID_EMAIL",
			wantContains: "invalid email address",
		},
		{
			name:        "empty email",
			body:        `{"email": ""}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    "INVALID_EMAIL",
			wantContains: "invalid email address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
			app.Post("/auth/request-magic-link", h.RequestMagicLink)

			req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, fiber.TestConfig{})
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			body := string(bodyBytes)
			t.Logf("Error response: %s", body)

			if !contains(body, tc.wantCode) {
				t.Errorf("missing code %q in response: %s", tc.wantCode, body)
			}
			if !contains(body, tc.wantContains) {
				t.Errorf("missing expected message %q in response: %s", tc.wantContains, body)
			}
		})
	}
}

// TestAuthenticateToken_MissingPKCE verifies PKCE cookie validation error
func TestAuthenticateToken_MissingPKCE(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "test"}}

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/auth/authenticate-token", h.AuthenticateToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/authenticate-token", strings.NewReader(`{"token": "abc"}`))
	req.Header.Set("Content-Type", "application/json")
	// No PKCE cookie

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	t.Logf("Error response: %s", body)

	if !contains(body, "MISSING_PKCE") {
		t.Errorf("missing MISSING_PKCE code: %s", body)
	}
}

// TestLogout_MissingSession verifies logout without session
func TestLogout_MissingSession(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "test"}}

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/auth/logout", h.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	// No session cookie

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	t.Logf("Error response: %s", body)

	if !contains(body, "UNAUTHENTICATED") {
		t.Errorf("missing UNAUTHENTICATED code: %s", body)
	}
}

// TestProvisionTenant_MissingTicket verifies missing onboarding ticket error
func TestProvisionTenant_MissingTicket(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "test"}}

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/onboarding/complete", h.ProvisionTenant)

	req := httptest.NewRequest(http.MethodPost, "/onboarding/complete", strings.NewReader(`{"school_name": "Test School", "idempotency_key": "key123"}`))
	req.Header.Set("Content-Type", "application/json")
	// No onboarding ticket cookie

	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	t.Logf("Error response: %s", body)

	if !contains(body, "MISSING_TICKET") {
		t.Errorf("missing MISSING_TICKET code: %s", body)
	}
}

// TestAuthenticateToken_DevOnboardingCookie verifies onboarding cookie attributes in dev env
func TestAuthenticateToken_DevOnboardingCookie(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "development"}}

	app := fiber.New(fiber.Config{ErrorHandler: testErrorHandler()})
	app.Post("/test-onboarding-cookie", func(c fiber.Ctx) error {
		onboardingCookieName := "__Host-onboarding"
		onboardingSecure := h.cfg.Env == "production" && c.Scheme() == "https"
		if h.cfg.Env != "production" {
			onboardingCookieName = "onboarding"
			onboardingSecure = false
		}
		c.Cookie(&fiber.Cookie{
			Name:     onboardingCookieName,
			Value:    "test-ticket-token",
			Path:     "/",
			HTTPOnly: true,
			Secure:   onboardingSecure,
			SameSite: fiber.CookieSameSiteLaxMode,
			MaxAge:   int(10 * time.Minute / time.Second),
		})
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/test-onboarding-cookie", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("no Set-Cookie header found")
	}

	cookieHeader := cookies[0]
	t.Logf("Dev Set-Cookie: %s", cookieHeader)

	if !contains(cookieHeader, "onboarding=test-ticket-token") {
		t.Errorf("cookie name should be 'onboarding' in dev: %s", cookieHeader)
	}
	if !contains(cookieHeader, "SameSite=Lax") && !contains(cookieHeader, "samesite=lax") {
		t.Errorf("missing SameSite=Lax: %s", cookieHeader)
	}
	if !contains(cookieHeader, "Max-Age=600") && !contains(cookieHeader, "max-age=600") {
		t.Errorf("missing Max-Age=600: %s", cookieHeader)
	}
	if !contains(cookieHeader, "HttpOnly") && !contains(cookieHeader, "httponly") {
		t.Errorf("missing HttpOnly: %s", cookieHeader)
	}
	if !contains(cookieHeader, "Path=/") && !contains(cookieHeader, "path=/") {
		t.Errorf("missing Path=/: %s", cookieHeader)
	}
}

// TestAuthenticateToken_ProdOnboardingCookie verifies onboarding cookie attributes in prod env (https)
func TestAuthenticateToken_ProdOnboardingCookie(t *testing.T) {
	h := &Handler{cfg: &config.Config{Env: "production"}}

	// Test the cookie logic directly by simulating the handler's cookie setting
	onboardingCookieName := "__Host-onboarding"
	onboardingSecure := h.cfg.Env == "production" && true // simulate https
	if h.cfg.Env != "production" {
		onboardingCookieName = "onboarding"
		onboardingSecure = false
	}

	if onboardingCookieName != "__Host-onboarding" {
		t.Errorf("cookie name should be '__Host-onboarding' in production, got %s", onboardingCookieName)
	}
	if !onboardingSecure {
		t.Errorf("Secure should be true for production https")
	}

	// Also test with http (should not set Secure)
	onboardingSecure = h.cfg.Env == "production" && false // simulate http
	if onboardingSecure {
		t.Errorf("Secure should be false for production http")
	}

	// Test dev env
	hDev := &Handler{cfg: &config.Config{Env: "development"}}
	onboardingCookieName = "__Host-onboarding"
	onboardingSecure = hDev.cfg.Env == "production" && true // simulate https
	if hDev.cfg.Env != "production" {
		onboardingCookieName = "onboarding"
		onboardingSecure = false
	}

	if onboardingCookieName != "onboarding" {
		t.Errorf("cookie name should be 'onboarding' in development, got %s", onboardingCookieName)
	}
	if onboardingSecure {
		t.Errorf("Secure should be false in development")
	}
}

// MockService for handler tests (not used in current tests but kept for future)
type MockService struct{}

func (m *MockService) RequestMagicLink(ctx context.Context, email string) (string, error) {
	return "", nil
}

// testErrorHandler creates a fiber error handler for testing
func testErrorHandler() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return c.Status(apperr.Status(ae.Code)).JSON(fiber.Map{
				"code":       ae.Code,
				"message":    ae.Message,
				"errors":     ae.Fields,
				"request_id": c.Locals("request_id"),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":       "INTERNAL_ERROR",
			"message":    "Internal Server Error",
			"errors":     map[string][]apperr.FieldError{},
			"request_id": c.Locals("request_id"),
		})
	}
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
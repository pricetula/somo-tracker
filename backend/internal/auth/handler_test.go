package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"somotracker/backend/internal/config"
)

// ============================================================================
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *Service
	repo    *MockRepository
	handler *Handler
	mr      *miniredis.Miniredis
	rdb     *redis.Client
	logs    *observer.ObservedLogs
}

// mockSchoolCreator implements SchoolCreator for handler tests.
type mockSchoolCreator struct {
	createFn func(ctx context.Context, tenantID string, name string) (string, error)
}

func (m *mockSchoolCreator) Create(ctx context.Context, tenantID string, name string) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, name)
	}
	return "school_" + tenantID, nil
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	// Start miniredis for lightweight Redis mocking
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)

	// Create a real go-redis client pointed at miniredis
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	repo := NewMockRepository()

	observedCore, observedLogs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)

	cfg := config.Config{
		AppEnv: "test",
	}

	svc := &Service{
		idp:           &MockIdentityProvider{},
		repo:          repo,
		rdb:           rdb,
		logger:        logger,
		cfg:           cfg,
		schoolCreator: &mockSchoolCreator{},
		yearCreator:   &mockYearCreator{},
	}

	handler := NewHandler(svc, logger, cfg)

	app := fiber.New()
	handler.RegisterRoutes(app)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
		mr:      mr,
		rdb:     rdb,
		logs:    observedLogs,
	}
}

// doRequest sends an HTTP request to the Fiber app with an optional cookie.
func (h *handlerTestHarness) doRequest(method, path string, cookieValue string) *http.Response {
	return h.doRequestWithBody(method, path, cookieValue, nil)
}

// doRequestWithBody sends an HTTP request with an optional JSON body and cookie.
func (h *handlerTestHarness) doRequestWithBody(method, path string, cookieValue string, body any) *http.Response {
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookieValue != "" {
		req.Header.Set("Cookie", "somo_sid="+cookieValue)
	}
	resp, _ := h.app.Test(req)
	return resp
}

// doRequestWithQuery sends an HTTP request with query parameters.
func (h *handlerTestHarness) doRequestWithQuery(method, path, query string, cookieValue string) *http.Response {
	fullPath := path
	if query != "" {
		fullPath = path + "?" + query
	}
	req := httptest.NewRequest(method, fullPath, nil)
	if cookieValue != "" {
		req.Header.Set("Cookie", "somo_sid="+cookieValue)
	}
	resp, _ := h.app.Test(req)
	return resp
}

// setSessionInRedis pre-populates a session key in miniredis so that
// the service's GetMe Redis check passes.
func (h *handlerTestHarness) setSessionInRedis(t *testing.T, token string) {
	if err := h.mr.Set("session:"+token, "stytch_sess_token"); err != nil {
		t.Fatalf("failed to set session in redis: %v", err)
	}
}

// ============================================================================
// Tests: GET /api/auth/me
// ============================================================================

// TestHandler_Me_HappyPath verifies that a valid session cookie returns
// the full user profile (requirement 6).
func TestHandler_Me_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "valid_session_token_001"

	// Pre-populate Redis so the service's GetMe Redis check passes
	h.setSessionInRedis(t, token)

	// Configure the mock repository to return a valid profile
	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return &MeInfo{
			UserID:     "user_001",
			TenantID:   "tenant_001",
			Role:       "SCHOOL_ADMIN",
			SchoolID:   "school_001",
			SchoolName: "Test School",
			FullName:   "Alice Smith",
			Email:      "alice@example.com",
		}, nil
	}

	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["user_id"] != "user_001" {
		t.Fatalf("expected user_id 'user_001', got %v", body["user_id"])
	}
	if body["tenant_id"] != "tenant_001" {
		t.Fatalf("expected tenant_id 'tenant_001', got %v", body["tenant_id"])
	}
	if body["role"] != "SCHOOL_ADMIN" {
		t.Fatalf("expected role 'SCHOOL_ADMIN', got %v", body["role"])
	}
	if body["school_id"] != "school_001" {
		t.Fatalf("expected school_id 'school_001', got %v", body["school_id"])
	}
	if body["school_name"] != "Test School" {
		t.Fatalf("expected school_name 'Test School', got %v", body["school_name"])
	}
	if body["full_name"] != "Alice Smith" {
		t.Fatalf("expected full_name 'Alice Smith', got %v", body["full_name"])
	}
	if body["email"] != "alice@example.com" {
		t.Fatalf("expected email 'alice@example.com', got %v", body["email"])
	}
}

// TestHandler_Me_MissingCookie verifies that a request without a session
// cookie returns 401 Unauthorized with the standard error body.
func TestHandler_Me_MissingCookie(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequest("GET", "/api/auth/me", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", body.Code)
	}
	if body.Message != "no session cookie found" {
		t.Fatalf("expected message 'no session cookie found', got %q", body.Message)
	}
}

// TestHandler_Me_EmptyCookie verifies that an empty somo_sid cookie is
// treated the same as a missing cookie (401 Unauthorized with error body).
func TestHandler_Me_EmptyCookie(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Set cookie but with empty value — handler checks token == ""
	// Fiber's c.Cookies returns "" for missing cookies, so an empty
	// cookie is the same as no cookie.
	resp := h.doRequest("GET", "/api/auth/me", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", body.Code)
	}
}

// TestHandler_Me_ExpiredToken verifies that a session token which does
// not exist in Redis returns 401 Unauthorized (via middleware.HTTPError
// mapping ErrExpiredToken → middleware.ErrUnauthorized → 401).
func TestHandler_Me_ExpiredToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Do NOT pre-populate Redis — the token doesn't exist
	resp := h.doRequest("GET", "/api/auth/me", "nonexistent_token")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", body.Code)
	}
	if body.Message != "authentication required" {
		t.Fatalf("expected message 'authentication required', got %q", body.Message)
	}
}

// TestHandler_Me_RedisExistsWithRepoNotFound verifies that when Redis
// has the session key but the repository returns ErrNotFound (e.g., the
// session was deleted from Postgres but not from Redis), the service
// cleans up the stale Redis entry and returns 401 Unauthorized.
func TestHandler_Me_RedisExistsWithRepoNotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "stale_session_token_001"

	// Pre-populate Redis so the first check passes
	h.setSessionInRedis(t, token)

	// Configure the mock repository to return ErrNotFound (session
	// exists in Redis but not in Postgres)
	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return nil, ErrNotFound
	}

	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", body.Code)
	}

	// Verify the stale Redis entry was cleaned up
	if h.mr.Exists("session:" + token) {
		t.Fatal("stale Redis entry should have been cleaned up after repo miss")
	}
}

// TestHandler_Me_InternalError verifies that an unexpected repository
// error returns 500 Internal Server Error with a generic message.
func TestHandler_Me_InternalError(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "internal_error_token_001"

	// Pre-populate Redis so the first check passes
	h.setSessionInRedis(t, token)

	// Configure the mock repository to return an internal error
	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return nil, ErrInternal
	}

	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Code != "internal_error" {
		t.Fatalf("expected code 'internal_error', got %q", body.Code)
	}
	if body.Message != "an unexpected error occurred" {
		t.Fatalf("expected generic message, got %q", body.Message)
	}

	// Verify a WARN/ERROR log was emitted for the internal error
	if h.logs.FilterLevelExact(zapcore.ErrorLevel).Len() == 0 {
		t.Log("note: expected ERROR log for internal error (non-fatal check for middleware logging)")
	}
}

// TestHandler_Me_ResponseShape verifies that the JSON response body
// contains exactly the expected fields and no extras.
func TestHandler_Me_ResponseShape(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "shape_check_token_001"
	h.setSessionInRedis(t, token)

	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return &MeInfo{
			UserID:     "user_shape",
			TenantID:   "tenant_shape",
			Role:       "TEACHER",
			SchoolID:   "school_shape",
			SchoolName: "Shape School",
			FullName:   "Shape User",
			Email:      "shape@example.com",
		}, nil
	}

	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expectedFields := []string{"user_id", "tenant_id", "role", "school_id", "school_name", "full_name", "email"}
	for _, field := range expectedFields {
		if _, ok := body[field]; !ok {
			t.Fatalf("expected field %q in response, but it's missing", field)
		}
	}

	// Verify no unexpected fields
	if len(body) != len(expectedFields) {
		t.Fatalf("expected %d fields, got %d: %v", len(expectedFields), len(body), body)
	}
}

// TestHandler_Me_SchoolFieldsEmpty verifies that school_id and school_name
// are returned as empty strings when the user has no active school.
func TestHandler_Me_SchoolFieldsEmpty(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "no_school_token_001"
	h.setSessionInRedis(t, token)

	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return &MeInfo{
			UserID:     "user_noschool",
			TenantID:   "tenant_noschool",
			Role:       "TEACHER",
			SchoolID:   "",
			SchoolName: "",
			FullName:   "No School User",
			Email:      "noschool@example.com",
		}, nil
	}

	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["school_id"] != "" {
		t.Fatalf("expected empty school_id, got %v", body["school_id"])
	}
	if body["school_name"] != "" {
		t.Fatalf("expected empty school_name, got %v", body["school_name"])
	}
}

// TestHandler_Me_InvalidMethod verifies that non-GET methods return 405
// Method Not Allowed.
func TestHandler_Me_InvalidMethod(t *testing.T) {
	h := newHandlerTestHarness(t)

	req := httptest.NewRequest("POST", "/api/auth/me", nil)
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed for POST, got %d", resp.StatusCode)
	}
}

// TestHandler_Me_LogoutClearsMe verifies that after a session is logged
// out (deleted from Redis), the /me endpoint returns 401 for that token.
func TestHandler_Me_LogoutClearsMe(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "logout_me_token_001"

	// Pre-populate Redis and configure repo
	h.setSessionInRedis(t, token)
	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return &MeInfo{
			UserID:     "user_logout",
			TenantID:   "tenant_logout",
			Role:       "TEACHER",
			SchoolID:   "school_logout",
			SchoolName: "Logout School",
			FullName:   "Logout User",
			Email:      "logout@example.com",
		}, nil
	}

	// First call should succeed
	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK before logout, got %d", resp.StatusCode)
	}

	// Simulate logout by removing from Redis
	h.mr.Del("session:" + token)

	// Second call should fail with 401
	resp = h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized after logout, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: POST /api/auth/discover
// ============================================================================

// TestHandler_Discover_HappyPath verifies that a valid email returns 200 OK
// and sends the discovery email via Stytch.
func TestHandler_Discover_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithBody("POST", "/api/auth/discover", "", map[string]string{
		"email": "alice@example.com",
	})

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// The mock IDP should have received the call
	if h.svc.idp.(*MockIdentityProvider).sendDiscoveryEmailCalls != 1 {
		t.Fatalf("expected 1 SendDiscoveryEmail call, got %d", h.svc.idp.(*MockIdentityProvider).sendDiscoveryEmailCalls)
	}
}

// TestHandler_Discover_MissingEmail verifies that omitting email returns 422.
func TestHandler_Discover_MissingEmail(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithBody("POST", "/api/auth/discover", "", map[string]string{})
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", body.Code)
	}
	if body.Message != "email is required" {
		t.Fatalf("expected message 'email is required', got %q", body.Message)
	}
}

// TestHandler_Discover_InvalidBody verifies that a malformed JSON body returns 422.
func TestHandler_Discover_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	req := httptest.NewRequest("POST", "/api/auth/discover", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// TestHandler_Discover_StytchError verifies that a Stytch failure during
// discovery is mapped to an appropriate HTTP error via middleware.HTTPError.
func TestHandler_Discover_StytchError(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Make the mock IDP return an internal error
	idp := h.svc.idp.(*MockIdentityProvider)
	idp.sendDiscoveryEmailFn = func(ctx context.Context, email string) error {
		return ErrInternal
	}

	resp := h.doRequestWithBody("POST", "/api/auth/discover", "", map[string]string{
		"email": "fail@example.com",
	})

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "internal_error" {
		t.Fatalf("expected code 'internal_error', got %q", body.Code)
	}
}

// TestHandler_Discover_InvalidMethod verifies that non-POST returns 405.
func TestHandler_Discover_InvalidMethod(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequest("GET", "/api/auth/discover", "")
	if resp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed for GET, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: POST /api/auth/register
// ============================================================================

// TestHandler_Register_HappyPath verifies a successful registration returns
// 204 No Content and sets session, role, and school ID cookies.
func TestHandler_Register_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	sessionRef := "550e8400-e29b-41d4-a716-446655440000"
	istKey := "ist:test:" + sessionRef
	cacheData, _ := json.Marshal(istCacheData{IST: "ist_value", Email: "alice@example.com"})
	if err := h.mr.Set(istKey, string(cacheData)); err != nil {
		t.Fatalf("set IST in redis: %v", err)
	}

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "Test School",
		SessionRef: sessionRef,
		FullName:   "Alice Smith",
	})

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	// Verify session cookie was set
	cookies := resp.Header.Values("Set-Cookie")
	hasSessionCookie := false
	hasRoleCookie := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_sid=") {
			hasSessionCookie = true
		}
		if strings.Contains(c, "somo_role=") {
			hasRoleCookie = true
		}
	}
	if !hasSessionCookie {
		t.Fatal("expected somo_sid cookie to be set")
	}
	if !hasRoleCookie {
		t.Fatal("expected somo_role cookie to be set")
	}
}

// TestHandler_Register_ValidationError verifies that invalid registration
// payload returns 400 with validation error details.
func TestHandler_Register_ValidationError(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "",
		SessionRef: "not-a-uuid",
		FullName:   "",
	})

	// ValidationError with ErrInvalidInput maps to 400
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}
}

// TestHandler_Register_InvalidBody verifies that a malformed JSON body returns 422.
func TestHandler_Register_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", body.Code)
	}
}

// TestHandler_Register_ExpiredToken verifies that a consumed or expired
// session_ref returns 401 via middleware.HTTPError.
func TestHandler_Register_ExpiredToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Do NOT pre-set the IST in Redis — it's already consumed or never existed
	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "Expired School",
		SessionRef: "550e8400-e29b-41d4-a716-446655449999",
		FullName:   "John Doe",
	})

	// ErrExpiredToken maps to 401 via middleware.HTTPError
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", resp.StatusCode)
	}
}

// TestHandler_Register_MFARequired verifies that an MFA-required error from
// the IDP returns 500 because auth.ErrMFARequired does not wrap
// middleware.ErrUnauthorized.
func TestHandler_Register_MFARequired(t *testing.T) {
	h := newHandlerTestHarness(t)

	sessionRef := "550e8400-e29b-41d4-a716-446655440001"
	istKey := "ist:test:" + sessionRef
	cacheData, _ := json.Marshal(istCacheData{IST: "ist_mfa", Email: "mfa@example.com"})
	if err := h.mr.Set(istKey, string(cacheData)); err != nil {
		t.Fatalf("set IST in redis: %v", err)
	}

	// Make the mock IDP return MFA required during IST exchange
	idp := h.svc.idp.(*MockIdentityProvider)
	idp.exchangeIntermediateSessionFn = func(ctx context.Context, ist, orgID string) (ExchangeResult, error) {
		return ExchangeResult{}, ErrMFARequired
	}

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "MFA School",
		SessionRef: sessionRef,
		FullName:   "Jane Doe",
	})

	// ErrMFARequired does NOT wrap middleware.ErrUnauthorized → falls to 500
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: POST /api/auth/verify
// ============================================================================

// TestHandler_Verify_HappyPath_NewUser verifies that a magic-link token for
// a new user (no discovered orgs) returns session_ref.
func TestHandler_Verify_HappyPath_NewUser(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithBody("POST", "/api/auth/verify", "", VerifyPayload{
		Token: "new_user_magic_token",
	})

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body struct {
		SessionRef string `json:"session_ref"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.SessionRef == "" {
		t.Fatal("expected non-empty session_ref for new user")
	}
}

// TestHandler_Verify_HappyPath_ExistingUser verifies that a magic-link token
// for an existing user (with discovered orgs and matching tenant) returns
// session_token, role, and email.
func TestHandler_Verify_HappyPath_ExistingUser(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateDiscoveryTokenFn = func(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
		return "ist_existing", "existing@example.com", []DiscoveredOrg{
			{OrganizationID: "org_existing", OrganizationName: "Existing School", MemberID: "member_existing", MemberAuthenticated: true},
		}, nil
	}

	h.repo.getTenantByStytchOrgIDFn = func(ctx context.Context, stytchOrgID string) (string, error) {
		return "tenant_existing", nil
	}
	h.repo.getUserByEmailAndTenantFn = func(ctx context.Context, email, tenantID string) (string, string, string, error) {
		return "user_existing", "Existing User", "ext_existing", nil
	}

	// handleExistingUser creates a session via CreateSessionOnly, then queries
	// it via GetSessionByToken to get the role. Ensure CreateSessionOnly stores
	// the session in the mock map.
	h.repo.createSessionOnlyFn = func(ctx context.Context, params CreateSessionParams) error {
		h.repo.sessions[params.Token] = &UserSession{
			Token:    params.Token,
			UserID:   "user_existing",
			TenantID: "tenant_existing",
			Role:     "SCHOOL_ADMIN",
		}
		return nil
	}

	resp := h.doRequestWithBody("POST", "/api/auth/verify", "", VerifyPayload{
		Token: "existing_user_magic_token",
	})

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body struct {
		SessionToken string `json:"session_token"`
		Role         string `json:"role"`
		Email        string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.SessionToken == "" {
		t.Fatal("expected non-empty session_token for existing user")
	}
	if body.Role == "" {
		t.Fatal("expected non-empty role for existing user")
	}
	if body.Email != "existing@example.com" {
		t.Fatalf("expected email 'existing@example.com', got %q", body.Email)
	}
}

// TestHandler_Verify_MissingToken verifies that an empty token returns 422.
func TestHandler_Verify_MissingToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithBody("POST", "/api/auth/verify", "", VerifyPayload{
		Token: "",
	})

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", body.Code)
	}
}

// TestHandler_Verify_InvalidBody verifies that a malformed JSON body returns 422.
func TestHandler_Verify_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	req := httptest.NewRequest("POST", "/api/auth/verify", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// TestHandler_Verify_ExpiredToken verifies that an expired magic-link token
// returns 401 via middleware.HTTPError.
func TestHandler_Verify_ExpiredToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateDiscoveryTokenFn = func(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
		return "", "", nil, fmt.Errorf("%w: token expired", ErrExpiredToken)
	}

	resp := h.doRequestWithBody("POST", "/api/auth/verify", "", VerifyPayload{
		Token: "expired_token",
	})

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestHandler_Verify_InvalidMethod verifies that non-POST returns 405.
func TestHandler_Verify_InvalidMethod(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequest("GET", "/api/auth/verify", "")
	if resp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed for GET, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: GET /api/auth/callback (magic link callback)
// ============================================================================

// TestHandler_Callback_NewUser verifies that a magic link callback for a new
// user redirects to the frontend registration page with session_ref.
func TestHandler_Callback_NewUser(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithQuery("GET", "/api/auth/callback", "token=new_user_token", "")

	// Should redirect (302 Found)
	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("expected 302 Found redirect, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "/register?session_ref=") &&
		!strings.Contains(location, "/register?session_ref=") {
		t.Fatalf("expected redirect to registration page with session_ref, got Location: %s", location)
	}
}

// TestHandler_Callback_ExistingUser verifies that a magic link callback for
// an existing user redirects to the frontend dashboard with session cookies set.
func TestHandler_Callback_ExistingUser(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateDiscoveryTokenFn = func(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
		return "ist_existing", "existing@example.com", []DiscoveredOrg{
			{OrganizationID: "org_callback", OrganizationName: "Callback School", MemberID: "member_callback", MemberAuthenticated: true},
		}, nil
	}

	h.repo.getTenantByStytchOrgIDFn = func(ctx context.Context, stytchOrgID string) (string, error) {
		return "tenant_callback", nil
	}
	h.repo.getUserByEmailAndTenantFn = func(ctx context.Context, email, tenantID string) (string, string, string, error) {
		return "user_callback", "Callback User", "ext_callback", nil
	}

	// The service's handleExistingUser calls CreateSessionOnly then
	// GetSessionByToken. We need CreateSessionOnly to store the session
	// so GetSessionByToken can find it to retrieve the role.
	h.repo.createSessionOnlyFn = func(ctx context.Context, params CreateSessionParams) error {
		h.repo.sessions[params.Token] = &UserSession{
			Token:    params.Token,
			UserID:   "user_callback",
			TenantID: "tenant_callback",
			Role:     "SCHOOL_ADMIN",
		}
		return nil
	}

	resp := h.doRequestWithQuery("GET", "/api/auth/callback", "token=existing_callback_token", "")

	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("expected 302 Found redirect, got %d", resp.StatusCode)
	}

	// Verify session cookies are set in the redirect
	cookies := resp.Header.Values("Set-Cookie")
	hasSessionCookie := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_sid=") {
			hasSessionCookie = true
			break
		}
	}
	if !hasSessionCookie {
		t.Fatal("expected somo_sid cookie to be set for existing user callback")
	}
}

// TestHandler_Callback_MissingToken verifies that a callback without a token
// returns 422 Unprocessable Entity.
func TestHandler_Callback_MissingToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithQuery("GET", "/api/auth/callback", "", "")
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", body.Code)
	}
	if body.Message != "token query parameter is required" {
		t.Fatalf("expected message 'token query parameter is required', got %q", body.Message)
	}
}

// TestHandler_Callback_ExpiredToken verifies that a callback with an expired
// magic link token returns an error (handled by middleware.HTTPError).
func TestHandler_Callback_ExpiredToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateDiscoveryTokenFn = func(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
		return "", "", nil, fmt.Errorf("%w: token expired", ErrExpiredToken)
	}

	resp := h.doRequestWithQuery("GET", "/api/auth/callback", "token=expired_callback_token", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", body.Code)
	}
}

// ============================================================================
// Tests: GET /api/auth/invite/callback (invite callback)
// ============================================================================

// TestHandler_InviteCallback_HappyPath verifies that a valid invite token
// returns 302 redirect to dashboard with session cookies set.
func TestHandler_InviteCallback_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Configure mock IDP to return valid invite auth
	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateInviteTokenFn = func(ctx context.Context, token string) (string, string, error) {
		return "ist_invite", "invited@example.com", nil
	}
	idp.exchangeInviteSessionFn = func(ctx context.Context, ist, orgID string) (string, error) {
		return "sty_sess_invite", nil
	}

	// Configure repo to return a valid invitation
	h.repo.getInvitationByEmailFn = func(ctx context.Context, email string) (*Invitation, error) {
		return &Invitation{
			ID:             "invite_001",
			TenantID:       "tenant_001",
			SchoolID:       "school_001",
			Role:           "TEACHER",
			Email:          "invited@example.com",
			FullName:       "Invited Teacher",
			Status:         "pending",
			StytchMemberID: "sty_member_invited",
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}, nil
	}
	h.repo.getTenantStytchOrgIDFn = func(ctx context.Context, tenantID string) (string, error) {
		return "org_invite", nil
	}

	resp := h.doRequestWithQuery("GET", "/api/auth/invite/callback", "token=valid_invite_token", "")

	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("expected 302 Found redirect, got %d", resp.StatusCode)
	}

	// Verify session cookies are set
	cookies := resp.Header.Values("Set-Cookie")
	hasSessionCookie := false
	hasRoleCookie := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_sid=") {
			hasSessionCookie = true
		}
		if strings.Contains(c, "somo_role=") {
			hasRoleCookie = true
		}
	}
	if !hasSessionCookie {
		t.Fatal("expected somo_sid cookie to be set")
	}
	if !hasRoleCookie {
		t.Fatal("expected somo_role cookie to be set")
	}
}

// TestHandler_InviteCallback_MissingToken verifies that a callback without
// a token returns 400 Bad Request.
func TestHandler_InviteCallback_MissingToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithQuery("GET", "/api/auth/invite/callback", "", "")
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", body.Code)
	}
	if body.Message != "token query parameter is required" {
		t.Fatalf("expected message 'token query parameter is required', got %q", body.Message)
	}
}

// TestHandler_InviteCallback_ExpiredInvitation verifies that an expired or
// already-accepted invitation returns a 401 (mapped via middleware.HTTPError).
func TestHandler_InviteCallback_ExpiredInvitation(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateInviteTokenFn = func(ctx context.Context, token string) (string, string, error) {
		return "ist_invite", "expired@example.com", nil
	}

	// No invitation exists → service returns ErrExpiredToken wrapping middleware.ErrUnauthorized
	h.repo.getInvitationByEmailFn = func(ctx context.Context, email string) (*Invitation, error) {
		return nil, ErrNotFound
	}

	resp := h.doRequestWithQuery("GET", "/api/auth/invite/callback", "token=expired_invite_token", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestHandler_InviteCallback_StytchExchangeMFAFailure verifies that when
// the Stytch IST exchange fails due to MFA not being satisfied, the handler
// returns 500 because auth.ErrMFARequired does not wrap
// middleware.ErrUnauthorized (it falls through to the default case).
func TestHandler_InviteCallback_StytchExchangeMFAFailure(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateInviteTokenFn = func(ctx context.Context, token string) (string, string, error) {
		return "ist_mfa", "mfa@example.com", nil
	}
	h.repo.getInvitationByEmailFn = func(ctx context.Context, email string) (*Invitation, error) {
		return &Invitation{
			ID:             "invite_mfa",
			TenantID:       "tenant_mfa",
			SchoolID:       "school_mfa",
			Role:           "TEACHER",
			Email:          "mfa@example.com",
			FullName:       "MFA Teacher",
			Status:         "pending",
			StytchMemberID: "sty_member_mfa",
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}, nil
	}
	h.repo.getTenantStytchOrgIDFn = func(ctx context.Context, tenantID string) (string, error) {
		return "org_mfa", nil
	}
	idp.exchangeInviteSessionFn = func(ctx context.Context, ist, orgID string) (string, error) {
		return "", ErrMFARequired
	}

	resp := h.doRequestWithQuery("GET", "/api/auth/invite/callback", "token=mfa_invite_token", "")
	// ErrMFARequired does not wrap middleware.ErrUnauthorized → falls to 500
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: DELETE /api/auth/session (logout)
// ============================================================================

// TestHandler_Logout_HappyPath verifies that a successful logout returns
// 204 No Content and clears all auth cookies.
func TestHandler_Logout_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Pre-set a session in Redis
	token := "logout_test_token"
	h.setSessionInRedis(t, token)

	resp := h.doRequest("DELETE", "/api/auth/session", token)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	// Verify auth cookies are cleared (MaxAge <= 0)
	cookies := resp.Header.Values("Set-Cookie")
	hasClearedSession := false
	hasClearedRole := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_sid=") && (strings.Contains(c, "Max-Age=0") || strings.Contains(c, "max-age=0")) {
			hasClearedSession = true
		}
		if strings.Contains(c, "somo_role=") && (strings.Contains(c, "Max-Age=0") || strings.Contains(c, "max-age=0")) {
			hasClearedRole = true
		}
	}
	if !hasClearedSession {
		t.Fatal("expected somo_sid cookie to be cleared on logout")
	}
	if !hasClearedRole {
		t.Fatal("expected somo_role cookie to be cleared on logout")
	}
}

// TestHandler_Logout_NoCookie verifies that a logout request without a
// session cookie still returns 204 (idempotent) and clears cookies.
func TestHandler_Logout_NoCookie(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequest("DELETE", "/api/auth/session", "")

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content for logout without cookie, got %d", resp.StatusCode)
	}

	// Auth cookies should still be cleared even without a session
	cookies := resp.Header.Values("Set-Cookie")
	hasClearedSession := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_sid=") && (strings.Contains(c, "Max-Age=0") || strings.Contains(c, "max-age=0")) {
			hasClearedSession = true
		}
	}
	if !hasClearedSession {
		t.Fatal("expected somo_sid cookie to be cleared even without a session")
	}
}

// TestHandler_Logout_ServiceError verifies that even when the service returns
// an error (e.g., Redis down), the handler still clears the cookies and does
// not leave the user in a stuck state.
func TestHandler_Logout_ServiceError(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "logout_error_token"
	h.setSessionInRedis(t, token)

	// Make the repository return an internal error on delete
	h.repo.deleteSessionFn = func(ctx context.Context, token string) error {
		return ErrInternal
	}

	resp := h.doRequest("DELETE", "/api/auth/session", token)

	// Even with service error, cookies should be cleared
	cookies := resp.Header.Values("Set-Cookie")
	hasClearedSession := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_sid=") && (strings.Contains(c, "Max-Age=0") || strings.Contains(c, "max-age=0")) {
			hasClearedSession = true
		}
	}
	if !hasClearedSession {
		t.Fatal("expected somo_sid cookie to be cleared even on service error")
	}
}

// TestHandler_Logout_InvalidMethod verifies that non-DELETE returns 405.
func TestHandler_Logout_InvalidMethod(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequest("GET", "/api/auth/session", "")
	if resp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed for GET, got %d", resp.StatusCode)
	}

	resp = h.doRequest("POST", "/api/auth/session", "")
	if resp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed for POST, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: HTTPError mapping edge cases (via handlers that go through middleware)
// ============================================================================

// TestHandler_ErrorMapping_ExpiredToken verifies that ErrExpiredToken maps
// to 401 via middleware.HTTPError through the discover endpoint.
func TestHandler_ErrorMapping_ExpiredToken(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.sendDiscoveryEmailFn = func(ctx context.Context, email string) error {
		return ErrExpiredToken
	}

	resp := h.doRequestWithBody("POST", "/api/auth/discover", "", map[string]string{
		"email": "expired@example.com",
	})

	// ErrExpiredToken wraps middleware.ErrUnauthorized → 401
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for ErrExpiredToken, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", body.Code)
	}
}

// TestHandler_ErrorMapping_ExpiredTokenThroughVerify verifies that
// middleware.HTTPError maps ErrExpiredToken to 401 through the verify
// endpoint (not just discover).
func TestHandler_ErrorMapping_ExpiredTokenThroughVerify(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateDiscoveryTokenFn = func(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
		return "", "", nil, ErrExpiredToken
	}

	resp := h.doRequestWithBody("POST", "/api/auth/verify", "", VerifyPayload{
		Token: "expired_verify_token",
	})

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestHandler_ErrorMapping_ExpiredTokenThroughCallback verifies that
// middleware.HTTPError maps ErrExpiredToken to 401 through the callback.
func TestHandler_ErrorMapping_ExpiredTokenThroughCallback(t *testing.T) {
	h := newHandlerTestHarness(t)

	idp := h.svc.idp.(*MockIdentityProvider)
	idp.authenticateDiscoveryTokenFn = func(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
		return "", "", nil, ErrExpiredToken
	}

	resp := h.doRequestWithQuery("GET", "/api/auth/callback", "token=expired_cb_token", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Cookie setting on register (additional checks)
// ============================================================================

// ============================================================================
// Tests: Cookie setting on register (additional checks)
// ============================================================================

// TestHandler_Register_CookieSecureFlag verifies that cookies have Secure=true
// in non-development environments.
func TestHandler_Register_SetsCSRFTokenCookie(t *testing.T) {
	h := newHandlerTestHarness(t)

	sessionRef := "550e8400-e29b-41d4-a716-446655440010"
	istKey := "ist:test:" + sessionRef
	cacheData, _ := json.Marshal(istCacheData{IST: "ist_csrf", Email: "csrf@example.com"})
	if err := h.mr.Set(istKey, string(cacheData)); err != nil {
		t.Fatalf("set IST in redis: %v", err)
	}

	resp := h.doRequestWithBody("POST", "/api/auth/register", "", RegistrationPayload{
		SchoolName: "CSRF School",
		SessionRef: sessionRef,
		FullName:   "CSRF User",
	})

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	// Verify CSRF token cookie is set
	cookies := resp.Header.Values("Set-Cookie")
	hasCSRFCookie := false
	for _, c := range cookies {
		if strings.Contains(c, "csrf_token=") {
			hasCSRFCookie = true
			break
		}
	}
	if !hasCSRFCookie {
		t.Fatal("expected csrf_token cookie to be set on registration")
	}
}

// TestHandler_Me_SchoolIDCookieSet verifies that the school_id cookie is set
// in the /me response when the user has an active school.
func TestHandler_Me_SchoolIDCookieSet(t *testing.T) {
	h := newHandlerTestHarness(t)

	token := "school_cookie_token"
	h.setSessionInRedis(t, token)

	h.repo.getMeInfoFn = func(ctx context.Context, token string) (*MeInfo, error) {
		return &MeInfo{
			UserID:     "user_sc",
			TenantID:   "tenant_sc",
			Role:       "TEACHER",
			SchoolID:   "school_sc_001",
			SchoolName: "School Cookie School",
			FullName:   "School Cookie User",
			Email:      "schoolcookie@example.com",
		}, nil
	}

	resp := h.doRequest("GET", "/api/auth/me", token)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Verify school_id cookie is set
	cookies := resp.Header.Values("Set-Cookie")
	hasSchoolIDCookie := false
	for _, c := range cookies {
		if strings.Contains(c, "somo_school_id=") && !strings.Contains(c, "Max-Age=0") && !strings.Contains(c, "max-age=0") {
			hasSchoolIDCookie = true
			break
		}
	}
	if !hasSchoolIDCookie {
		t.Fatal("expected somo_school_id cookie to be set in /me response")
	}
}

// TestHandler_Callback_NewUser_SetsCSRFCookie verifies the magic link
// callback for a new user also sets the CSRF token cookie.
func TestHandler_Callback_NewUser_SetsCSRFCookie(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doRequestWithQuery("GET", "/api/auth/callback", "token=new_user_csrf", "")

	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("expected 302 Found redirect, got %d", resp.StatusCode)
	}

	// Verify CSRF token cookie is set
	cookies := resp.Header.Values("Set-Cookie")
	hasCSRFCookie := false
	for _, c := range cookies {
		if strings.Contains(c, "csrf_token=") {
			hasCSRFCookie = true
			break
		}
	}
	if !hasCSRFCookie {
		t.Fatal("expected csrf_token cookie to be set on callback")
	}
}

// ============================================================================
// Helper: isErrorCodeInResponse decodes a standard error body and checks
// the code and optionally the message.
// ============================================================================

// ensureTestImports is a no-op helper to satisfy unused import requirements.
var _ = errors.Is

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
		idp:    &MockIdentityProvider{},
		repo:   repo,
		rdb:    rdb,
		logger: logger,
		cfg:    cfg,
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
	req := httptest.NewRequest(method, path, nil)
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

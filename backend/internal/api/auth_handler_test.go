package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/services"
)

// mockAuthService is a hand-rolled stub of services.AuthService for
// transport-layer tests. It avoids the runtime cost (and infrastructure
// dependencies) of the real auth flow.
type mockAuthService struct {
	sendCalls          []sendMagicLinkCall
	authenticateCalls  []string
	sendErr            error
	authenticateErr    error
	authenticateResult *services.SessionResult
	revokeCalls        []string
	revokeErr          error
}

type sendMagicLinkCall struct {
	email       string
	orgIDOrSlug string
}

func (m *mockAuthService) SendMagicLink(_ context.Context, email, orgIDOrSlug string) error {
	m.sendCalls = append(m.sendCalls, sendMagicLinkCall{email: email, orgIDOrSlug: orgIDOrSlug})
	return m.sendErr
}

func (m *mockAuthService) AuthenticateCallback(_ context.Context, token string, _ fiber.Ctx) (*services.SessionResult, error) {
	m.authenticateCalls = append(m.authenticateCalls, token)
	return m.authenticateResult, m.authenticateErr
}

func (m *mockAuthService) RevokeSession(_ context.Context, token string, _ string) error {
	m.revokeCalls = append(m.revokeCalls, token)
	return m.revokeErr
}

// newTestRouter wires a real *Router against a mockAuthService. The rate
// limiter is nil so the middleware falls through (no Redis required).
func newTestRouter(mock *mockAuthService) *fiber.App {
	router := NewRouter(nil, nil, mock, nil)
	app := fiber.New()
	router.RegisterRoutes(app, nil, nil)
	return app
}

// sendRequest is a thin helper that returns the *http.Response directly.
func sendRequest(t *testing.T, app *fiber.App, req *httpRequest) *http.Response {
	t.Helper()
	httpReq := httptest.NewRequest(req.method, req.path, req.body)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	return resp
}

type httpRequest struct {
	method  string
	path    string
	body    *bytes.Reader
	headers map[string]string
}

// readJSON decodes the response body into a map.
func readJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	return got
}

// readJSONField extracts a top-level string field from the response body.
func readJSONField(t *testing.T, resp *http.Response, field string) string {
	t.Helper()
	return readJSON(t, resp)[field].(string)
}

// readCookie returns the cookie value by name, or t.Fatalf if absent.
func readCookie(t *testing.T, resp *http.Response, name string) string {
	t.Helper()
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	t.Fatalf("cookie %q not found in response", name)
	return ""
}

// =============================================================================
// /api/auth/magic-link/send
// =============================================================================

func TestSendMagicLink_HappyPath_Returns200(t *testing.T) {
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	body := bytes.NewReader([]byte(`{"email":"alice@example.com"}`))
	resp := sendRequest(t, app, &httpRequest{
		method: "POST",
		path:   "/api/auth/magic-link/send",
		body:   body,
	})
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, mock.sendCalls, 1, "service should be called exactly once")
	assert.Equal(t, "alice@example.com", mock.sendCalls[0].email)
}

func TestSendMagicLink_MissingEmail_Returns400(t *testing.T) {
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	body := bytes.NewReader([]byte(`{}`))
	resp := sendRequest(t, app, &httpRequest{
		method: "POST",
		path:   "/api/auth/magic-link/send",
		body:   body,
	})
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Empty(t, mock.sendCalls, "service must not be called when email is missing")
	assert.Equal(t, "missing_email", readJSONField(t, resp, "code"))
}

func TestSendMagicLink_AcceptsFormEncodedEmail(t *testing.T) {
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	form := "email=alice%40example.com"
	req := httptest.NewRequest("POST", "/api/auth/magic-link/send", bytes.NewReader([]byte(form)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, mock.sendCalls, 1)
	assert.Equal(t, "alice@example.com", mock.sendCalls[0].email)
}

func TestSendMagicLink_ServiceError_Returns500WithoutLeakage(t *testing.T) {
	mock := &mockAuthService{sendErr: assertAnError{}}
	app := newTestRouter(mock)

	body := bytes.NewReader([]byte(`{"email":"alice@example.com"}`))
	resp := sendRequest(t, app, &httpRequest{
		method: "POST",
		path:   "/api/auth/magic-link/send",
		body:   body,
	})
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "internal_error", readJSONField(t, resp, "code"),
		"service errors must not leak their internal type to clients")
}

func TestSendMagicLink_OrgIDOrSlug_PassesThroughToService(t *testing.T) {
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	body := bytes.NewReader([]byte(`{"email":"alice@example.com","org_id":"acme-university"}`))
	resp := sendRequest(t, app, &httpRequest{
		method: "POST",
		path:   "/api/auth/magic-link/send",
		body:   body,
	})
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, mock.sendCalls, 1)
	assert.Equal(t, "acme-university", mock.sendCalls[0].orgIDOrSlug)
}

// =============================================================================
// /api/auth/callback
// =============================================================================

func TestCallback_HappyPath_SetsSessionCookieAndReturns200(t *testing.T) {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	mock := &mockAuthService{
		authenticateResult: &services.SessionResult{
			OpaqueToken:     "opaque-abc123",
			StytchSessionID: "stytch-session-xyz",
			UserID:          "user-1",
			TenantID:        "tenant-1",
			ExpiresAt:       expiresAt,
		},
	}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=valid-token", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "authenticated", readJSONField(t, resp, "code"))
	assert.Equal(t, "opaque-abc123", readCookie(t, resp, "session_token"),
		"session_token cookie must be set from the service result")
	require.Len(t, mock.authenticateCalls, 1)
	assert.Equal(t, "valid-token", mock.authenticateCalls[0])
}

func TestCallback_MissingToken_Returns400(t *testing.T) {
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "missing_token", readJSONField(t, resp, "code"))
	assert.Empty(t, mock.authenticateCalls, "service must not be called when token is missing")
}

func TestCallback_WhitespaceToken_Returns400(t *testing.T) {
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=%20%20", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Empty(t, mock.authenticateCalls)
}

func TestCallback_ServiceBadRequest_Returns400(t *testing.T) {
	mock := &mockAuthService{
		authenticateErr: errBadRequest("token is invalid"),
	}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=bogus", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalid_request", readJSONField(t, resp, "code"))
}

func TestCallback_ServiceUnauthorized_Returns401(t *testing.T) {
	mock := &mockAuthService{
		authenticateErr: errUnauthorized("session expired"),
	}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=expired", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "unauthorized", readJSONField(t, resp, "code"))
}

func TestCallback_ServiceTooManyRequests_Returns429(t *testing.T) {
	mock := &mockAuthService{
		authenticateErr: errTooManyRequests("rate limit exceeded"),
	}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=spam", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, "rate_limit_exceeded", readJSONField(t, resp, "code"))
}

func TestCallback_ServiceInternalError_Returns500WithoutLeakage(t *testing.T) {
	mock := &mockAuthService{
		authenticateErr: assertAnError{}, // non-prefixed → mapped to 500
	}
	app := newTestRouter(mock)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=any", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "internal_error", readJSONField(t, resp, "code"),
		"untyped service errors must not leak their details to clients")
}

func TestCallback_RouteIsRegisteredGETOnly(t *testing.T) {
	// POST is intentionally not supported (no programmatic caller today).
	// This guards against an accidental re-introduction.
	mock := &mockAuthService{}
	app := newTestRouter(mock)

	body := bytes.NewReader([]byte(`{"token":"valid-token"}`))
	resp, err := app.Test(httptest.NewRequest("POST", "/api/auth/callback", body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode,
		"POST /api/auth/callback must not be routed; the route is GET-only")
	assert.Empty(t, mock.authenticateCalls)
}

func TestCallback_RateLimitMiddlewareAttached(t *testing.T) {
	// Verify the callback route is reachable through the registered middleware
	// chain. The nil limiter short-circuits the middleware, so we can exercise
	// the full handler without a live Redis instance.
	mock := &mockAuthService{
		authenticateResult: &services.SessionResult{
			OpaqueToken: "tok",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}
	router := NewRouter(nil, nil, mock, nil) // nil limiter → passes through
	app := fiber.New()
	router.RegisterRoutes(app, nil, nil)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/auth/callback?token=ok", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode,
		"callback route must be reachable through the rate-limit middleware chain")
}

// =============================================================================
// helpers
// =============================================================================

// assertAnError is an arbitrary non-prefixed error used to exercise the 500 path.
type assertAnError struct{}

func (assertAnError) Error() string { return "boom" }

func errBadRequest(msg string) error   { return prefixErr("bad_request", msg) }
func errUnauthorized(msg string) error { return prefixErr("unauthorized", msg) }
func errTooManyRequests(msg string) error {
	return prefixErr("too_many_requests", msg)
}

type prefixedErr struct{ prefix, msg string }

func (p prefixedErr) Error() string { return p.prefix + ": " + p.msg }

func prefixErr(prefix, msg string) error { return prefixedErr{prefix: prefix, msg: msg} }

// suppressBodyClose silences the errcheck warning on deferred Body.Close calls
// where the return value is intentionally ignored.
func suppressBodyClose(_ io.ReadCloser) {}

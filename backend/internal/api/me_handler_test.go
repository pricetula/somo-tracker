package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/services"
)

// mockMeService stubs services.MeService for transport-layer tests.
type mockMeService struct {
	calls  []meCall
	result services.MeResult
	err    error
}

type meCall struct {
	token    string
	tenantID string
}

func (m *mockMeService) GetCurrentUser(ctx context.Context, token string, tenantID string) (services.MeResult, error) {
	m.calls = append(m.calls, meCall{token: token, tenantID: tenantID})
	return m.result, m.err
}

// injectTenantMW is a test middleware that injects tenant_id into locals.
func injectTenantMW(tenantID string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("tenant_id", tenantID)
		return c.Next()
	}
}

func newMeTestApp(mock *mockMeService, injectTenant bool) *fiber.App {
	app := fiber.New()
	h := newMeHandler(mock)
	if injectTenant {
		app.Use(injectTenantMW("tenant-42"))
	}
	app.Get("/api/me", h.getMe)
	return app
}

func sendMeRequest(t *testing.T, app *fiber.App, cookie string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/me", nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "session_token", Value: cookie})
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func readJSONMap(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	return got
}

// =============================================================================
// Happy path
// =============================================================================

func TestGetMe_HappyPath_Returns200(t *testing.T) {
	mock := &mockMeService{
		result: services.MeResult{
			UserName:         "Alice",
			Email:            "alice@example.com",
			ActiveSchoolID:   "school-1",
			SchoolName:       "North High",
			ActiveSchoolRole: "teacher",
			TenantID:         "tenant-42",
		},
	}
	app := newMeTestApp(mock, true)

	resp := sendMeRequest(t, app, "token-abc")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "token-abc", mock.calls[0].token)
	assert.Equal(t, "tenant-42", mock.calls[0].tenantID)

	body := readJSONMap(t, resp)
	assert.Equal(t, "Alice", body["user_name"])
	assert.Equal(t, "alice@example.com", body["email"])
}

// =============================================================================
// Missing session token
// =============================================================================

func TestGetMe_MissingCookie_Returns401(t *testing.T) {
	mock := &mockMeService{}
	app := newMeTestApp(mock, true)

	resp := sendMeRequest(t, app, "")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.Empty(t, mock.calls, "service must not be called when cookie is missing")
	body := readJSONMap(t, resp)
	assert.Equal(t, "unauthorized", body["code"])
	assert.Equal(t, "session token missing", body["message"])
}

// =============================================================================
// Missing tenant context
// =============================================================================

func TestGetMe_MissingTenantContext_Returns401(t *testing.T) {
	mock := &mockMeService{}
	// No injectTenant middleware — locals have no tenant_id.
	app := newMeTestApp(mock, false)

	resp := sendMeRequest(t, app, "token-xyz")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.Empty(t, mock.calls)
	body := readJSONMap(t, resp)
	assert.Equal(t, "unauthorized", body["code"])
	assert.Equal(t, "tenant context missing", body["message"])
}

// =============================================================================
// Service error mapping
// =============================================================================

func TestGetMe_ServiceBadRequest_Returns400(t *testing.T) {
	mock := &mockMeService{err: errors.New("bad_request: missing session token")}
	app := newMeTestApp(mock, true)

	resp := sendMeRequest(t, app, "bad-token")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	body := readJSONMap(t, resp)
	assert.Equal(t, "bad_request", body["code"])
	assert.Equal(t, "missing session token", body["message"])
}

func TestGetMe_ServiceNotFound_Returns404(t *testing.T) {
	mock := &mockMeService{err: errors.New("not_found: session expired or invalid")}
	app := newMeTestApp(mock, true)

	resp := sendMeRequest(t, app, "expired")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	body := readJSONMap(t, resp)
	assert.Equal(t, "not_found", body["code"])
	assert.Equal(t, "session expired or invalid", body["message"])
}

func TestGetMe_ServiceUnknownError_Returns500WithoutLeakage(t *testing.T) {
	mock := &mockMeService{err: errors.New("db connection lost")}
	app := newMeTestApp(mock, true)

	resp := sendMeRequest(t, app, "any")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	body := readJSONMap(t, resp)
	assert.Equal(t, "internal_error", body["code"])
	assert.Equal(t, "An unexpected error occurred", body["message"])
}

// =============================================================================
// Cookie presence verification (service receives token)
// =============================================================================

func TestGetMe_CookieValuePassedToService(t *testing.T) {
	mock := &mockMeService{result: services.MeResult{UserName: "Bob"}}
	app := newMeTestApp(mock, true)

	resp := sendMeRequest(t, app, "secret-cookie-value")
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "secret-cookie-value", mock.calls[0].token)
}

// =============================================================================
// Empty cookie string explicitly set (cookie name exists but value empty)
// =============================================================================

func TestGetMe_EmptyCookieValue_Returns401(t *testing.T) {
	mock := &mockMeService{}
	app := newMeTestApp(mock, true)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ""})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.Empty(t, mock.calls)
}

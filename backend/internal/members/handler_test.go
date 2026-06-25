package members

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *Service
	repo    *MockRepository
	handler *Handler
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}

	svc := &Service{
		repo: repo,
	}

	handler := &Handler{
		svc:  svc,
		repo: repo,
	}

	app := fiber.New()

	// Test middleware that sets tenant_id and user_id (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	}

	// Register routes manually (bypassing RegisterRoutes which embeds requireAuth)
	members := app.Group("/api/v1/members", testAuth)
	members.Get("/", handler.List)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
	}
}

func doRequest(app *fiber.App, method, path string, body []byte) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Cookie", "somo_sid=valid_session_token")
	resp, _ := app.Test(req)
	return resp
}

// ============================================================================
// Tests: List Members Handler (GET /api/v1/members)
// ============================================================================

func TestHandler_ListMembers_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		return []Member{
			{ID: "user_001", Email: "alice@school.com", FullName: "Alice Smith", Role: "TEACHER", IsActive: true},
		}, 1, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/members/?role=TEACHER", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(result.Members))
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestHandler_ListMembers_WithSearch(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		if search != "Alice" {
			t.Errorf("expected search 'Alice', got %q", search)
		}
		return []Member{
			{ID: "user_001", Email: "alice@school.com", FullName: "Alice Smith", Role: "TEACHER", IsActive: true},
		}, 1, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/members/?role=TEACHER&search=Alice", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_ListMembers_MissingRole(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "GET", "/api/v1/members/", nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var errBody struct {
		Error   string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Error != "invalid_input" {
		t.Fatalf("expected error 'invalid_input', got %q", errBody.Error)
	}
}

func TestHandler_ListMembers_InvalidRole(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "GET", "/api/v1/members/?role=PRINCIPAL", nil)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_ListMembers_EmptyResults(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		return []Member{}, 0, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/members/?role=NURSE", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Members) != 0 {
		t.Fatalf("expected 0 members, got %d", len(result.Members))
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

func TestHandler_ListMembers_Pagination(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByRoleFn = func(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
		// page=2, per_page=20 → offset = (2-1)*20 = 20, limit = 20
		if offset != 20 {
			t.Errorf("expected offset 20, got %d", offset)
		}
		if limit != 20 {
			t.Errorf("expected limit 20, got %d", limit)
		}
		return []Member{}, 0, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/members/?role=TEACHER&page=2&per_page=20", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

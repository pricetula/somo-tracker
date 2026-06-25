package invitations

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
	app      *fiber.App
	svc      *Service
	repo     *MockRepository
	resolver *MockSchoolResolver
	handler  *Handler
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	resolver := &MockSchoolResolver{}
	svc := &Service{repo: repo}
	handler := &Handler{svc: svc, repo: repo, resolver: resolver}

	app := fiber.New()

	// Test middleware that sets tenant_id and user_id (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	}

	invitations := app.Group("/api/v1/invitations", testAuth)
	invitations.Get("/", handler.ListInvitations)

	return &handlerTestHarness{
		app:      app,
		svc:      svc,
		repo:     repo,
		resolver: resolver,
		handler:  handler,
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
// Tests: ListInvitations Handler (GET /api/v1/invitations)
// ============================================================================

func TestHandler_ListInvitations_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	fullName := "Alice"
	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		return []Invitation{
			{ID: "inv_001", Email: "alice@school.com", Role: "TEACHER", Status: "pending", FullName: &fullName},
		}, 1, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/invitations/", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListInvitationsResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(result.Invitations))
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
}

func TestHandler_ListInvitations_WithFilters(t *testing.T) {
	h := newHandlerTestHarness(t)

	fullName := "Bob"
	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		if filter.Status != "pending" {
			t.Errorf("expected status 'pending', got %q", filter.Status)
		}
		if filter.Role != "NURSE" {
			t.Errorf("expected role 'NURSE', got %q", filter.Role)
		}
		return []Invitation{
			{ID: "inv_002", Email: "bob@school.com", Role: "NURSE", Status: "pending", FullName: &fullName},
		}, 1, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/invitations/?status=pending&role=NURSE", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_ListInvitations_EmptyResults(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		return []Invitation{}, 0, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/invitations/?role=FINANCE", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

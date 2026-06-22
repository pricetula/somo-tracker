package members

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

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
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	idp := &MockIDP{}
	logger := zap.NewNop()
	cfg := config.Config{AppEnv: "test", FrontendURL: "http://localhost:3000"}

	svc := &Service{
		repo:   repo,
		idp:    idp,
		cfg:    cfg,
		logger: logger,
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
	members.Post("/invite", handler.BulkInvite)

	invitations := app.Group("/api/v1/invitations", testAuth)
	invitations.Get("/", handler.ListInvitations)
	invitations.Post("/", handler.CreateInvitations)

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
			{ID: "user_001", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith", Role: "TEACHER", IsActive: true},
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
			{ID: "user_001", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith", Role: "TEACHER", IsActive: true},
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

	var errBody ErrorBody
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

// ============================================================================
// Tests: BulkInvite Handler (POST /api/v1/members/invite)
// ============================================================================

func TestHandler_BulkInvite_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := BulkInviteRequest{
		Role: "TEACHER",
		Invites: []InviteItem{
			{Email: "teacher@school.com", FirstName: "New", LastName: "Teacher"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/members/invite", bodyBytes)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result BulkInviteResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Sent != 1 {
		t.Fatalf("expected 1 sent, got %d", result.Sent)
	}
}

func TestHandler_BulkInvite_MissingRole(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := BulkInviteRequest{
		Role: "",
		Invites: []InviteItem{
			{Email: "teacher@school.com", FirstName: "New", LastName: "Teacher"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/members/invite", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_BulkInvite_InvalidRole(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := BulkInviteRequest{
		Role: "PRINCIPAL",
		Invites: []InviteItem{
			{Email: "teacher@school.com", FirstName: "New", LastName: "Teacher"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/members/invite", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_BulkInvite_EmptyInvites(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := BulkInviteRequest{
		Role:    "TEACHER",
		Invites: []InviteItem{},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/members/invite", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_BulkInvite_InvalidJSON(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/members/invite", []byte("{invalid}"))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: ListInvitations Handler (GET /api/v1/invitations)
// ============================================================================

func TestHandler_ListInvitations_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	firstName := "Alice"
	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		return []Invitation{
			{ID: "inv_001", Email: "alice@school.com", Role: "TEACHER", Status: "pending", FirstName: &firstName},
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

	firstName := "Bob"
	h.repo.listInvitationsFn = func(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
		if filter.Status != "pending" {
			t.Errorf("expected status 'pending', got %q", filter.Status)
		}
		if filter.Role != "NURSE" {
			t.Errorf("expected role 'NURSE', got %q", filter.Role)
		}
		return []Invitation{
			{ID: "inv_002", Email: "bob@school.com", Role: "NURSE", Status: "pending", FirstName: &firstName},
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

// ============================================================================
// Tests: CreateInvitations Handler (POST /api/v1/invitations)
// ============================================================================

func TestHandler_CreateInvitations_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "teacher@school.com", FirstName: "New", LastName: "Teacher", Role: "TEACHER"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/invitations/", bodyBytes)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result BulkInviteResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Sent != 1 {
		t.Fatalf("expected 1 sent, got %d", result.Sent)
	}
}

func TestHandler_CreateInvitations_EmptyInvites(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := CreateInvitationsRequest{
		Invites: []CreateInviteItem{},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/invitations/", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateInvitations_InvalidJSON(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/invitations/", []byte("{bad}"))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateInvitations_PartialFailures(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Individual invite failures return in the response body, not as HTTP errors
	h.repo.getMemberByEmailFn = func(ctx context.Context, schoolID, email string) (*Member, error) {
		return nil, errors.New("db error")
	}

	body := CreateInvitationsRequest{
		Invites: []CreateInviteItem{
			{Email: "fail@school.com", FirstName: "Fail", LastName: "User", Role: "TEACHER"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/invitations/", bodyBytes)
	// Handler returns 200 even when individual invites fail — errors are in the body
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK (errors in body), got %d", resp.StatusCode)
	}

	var result BulkInviteResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", result.Failed)
	}
	if result.Sent != 0 {
		t.Fatalf("expected 0 sent, got %d", result.Sent)
	}
}

package imports

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
	client := &MockAsynqClient{}
	logger := zap.NewNop()
	cfg := config.Config{AppEnv: "test", FrontendURL: "http://localhost:3000"}

	svc := &Service{
		repo:   repo,
		client: client,
		logger: logger,
		cfg:    cfg,
	}

	handler := &Handler{
		svc:    svc,
		repo:   repo,
		rdb:    nil,
		logger: logger,
	}

	app := fiber.New()

	// Register routes with a test auth middleware (bypasses real requireAuth)
	// that sets the tenant_id and user_id locals
	imports := app.Group("/api/v1/imports/staff", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	})

	imports.Post("/", handler.StartImport)
	imports.Get("/track/:id", handler.TrackImport)
	imports.Get("/track/:id/sse", handler.SSETrackImport)
	imports.Get("/:id/failures", handler.ListFailedInvitations)

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
// Tests: StartImport Handler
// ============================================================================

func TestHandler_StartImport_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := StartImportRequest{
		Role: "SCHOOL_ADMIN",
		Records: []ImportStaffRecord{
			{Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", bodyBytes)
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}

	var result StartImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.ImportJobID == "" {
		t.Fatal("expected non-empty import_job_id")
	}
	if result.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", result.Status)
	}
}

func TestHandler_StartImport_MissingRole(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := StartImportRequest{
		Role: "",
		Records: []ImportStaffRecord{
			{Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var errBody ErrorBody
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Error != "invalid_input" {
		t.Fatalf("expected error 'invalid_input', got %q", errBody.Error)
	}
}

func TestHandler_StartImport_InvalidJSON(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", []byte("{invalid json"))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_StartImport_ServiceError(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getTenantStytchOrgIDFn = func(ctx context.Context, tenantID string) (string, error) {
		return "", errors.New("tenant not found")
	}

	body := StartImportRequest{
		Role: "NURSE",
		Records: []ImportStaffRecord{
			{Email: "nurse@school.com", FirstName: "Nurse", LastName: "Betty"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: TrackImport Handler
// ============================================================================

func TestHandler_TrackImport_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return &ImportJob{
			ID:           jobID,
			TenantID:     "tenant_001",
			SchoolID:     "school_001",
			Role:         "SCHOOL_ADMIN",
			Status:       "completed",
			TotalRecords: 10,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/track/job_001", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result TrackImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Job.ID != "job_001" {
		t.Fatalf("expected job ID 'job_001', got %q", result.Job.ID)
	}
}

func TestHandler_TrackImport_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return nil, errors.New("import job not found")
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/track/nonexistent", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: ListFailedInvitations Handler
// ============================================================================

func TestHandler_ListFailedInvitations_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	firstName := "Alice"
	errMsg := "stytch invite failed"

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return []FailedInvitation{
			{ID: "inv_001", Email: "alice@bad.com", FirstName: &firstName, ErrorMessage: &errMsg},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/job_001/failures", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListFailedInvitationsResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Invitations) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(result.Invitations))
	}
}

func TestHandler_ListFailedInvitations_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return nil, errors.New("import job not found")
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/nonexistent/failures", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

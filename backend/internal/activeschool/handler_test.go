package activeschool

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/middleware"
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
	svc := NewService(repo)
	handler := NewHandler(svc, config.Config{AppEnv: "test"})

	app := fiber.New()

	// Test middleware that sets session (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("session", &middleware.SessionInfo{
			UserID:   "user_001",
			TenantID: "tenant_001",
			Role:     "TEACHER",
		})
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	}

	// Register routes manually with test auth
	as := app.Group("/api/v1/active-school", testAuth)
	as.Put("/", handler.Switch)
	as.Get("/", handler.Get)

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
	resp, _ := app.Test(req)
	return resp
}

// ============================================================================
// Tests: Switch Active School (PUT /api/v1/active-school)
// ============================================================================

func TestHandler_Switch_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.upsertFn = func(ctx context.Context, tenantID, userID, schoolID string) error {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		if schoolID != "school_002" {
			t.Errorf("expected schoolID 'school_002', got %q", schoolID)
		}
		return nil
	}

	body, _ := json.Marshal(SwitchActiveSchoolPayload{SchoolID: "school_002"})
	resp := doRequest(h.app, "PUT", "/api/v1/active-school", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["message"] != "active school updated" {
		t.Fatalf("expected message 'active school updated', got %v", result["message"])
	}
}

func TestHandler_Switch_MissingSchoolID(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(SwitchActiveSchoolPayload{SchoolID: ""})
	resp := doRequest(h.app, "PUT", "/api/v1/active-school", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_Switch_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "PUT", "/api/v1/active-school", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Get Active School (GET /api/v1/active-school)
// ============================================================================

func TestHandler_Get_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getActiveSchoolIDFn = func(ctx context.Context, tenantID, userID string) (string, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		return "school_001", nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/active-school", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["school_id"] != "school_001" {
		t.Fatalf("expected school_id 'school_001', got %v", result["school_id"])
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getActiveSchoolIDFn = func(ctx context.Context, tenantID, userID string) (string, error) {
		return "", ErrNotFound
	}

	resp := doRequest(h.app, "GET", "/api/v1/active-school", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

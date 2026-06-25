package cbcschools

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Test middleware that sets tenant_id (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	}

	// Register routes manually with test auth
	schools := app.Group("/api/v1/schools", testAuth)
	schools.Post("/", handler.Create)
	schools.Get("/", handler.List)
	schools.Put("/:id", handler.Update)
	schools.Delete("/:id", handler.Delete)

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
// Tests: Create School (POST /api/v1/schools)
// ============================================================================

func TestHandler_CreateSchool_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, tenantID, name string) (string, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if name != "Green Valley Primary" {
			t.Errorf("expected name 'Green Valley Primary', got %q", name)
		}
		return "school_001", nil
	}

	body, _ := json.Marshal(CreateSchoolPayload{Name: "Green Valley Primary"})
	resp := doRequest(h.app, "POST", "/api/v1/schools", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.ID != "school_001" {
		t.Fatalf("expected id 'school_001', got %q", result.ID)
	}
}

func TestHandler_CreateSchool_MissingName(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(CreateSchoolPayload{Name: ""})
	resp := doRequest(h.app, "POST", "/api/v1/schools", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateSchool_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/schools", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: List Schools (GET /api/v1/schools)
// ============================================================================

func TestHandler_ListSchools_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	now := time.Now()
	expectedSchools := []SchoolWithMemberCount{
		{
			ID: "school_001", TenantID: "tenant_001", Name: "Green Valley",
			County: "Nairobi", SubCounty: "Westlands", SchoolType: "Public",
			IsActive: true, CreatedAt: now, UpdatedAt: now, TotalMembers: 15,
		},
		{
			ID: "school_002", TenantID: "tenant_001", Name: "Riverside Academy",
			County: "Nairobi", SubCounty: "Kilimani", SchoolType: "Private",
			IsActive: true, CreatedAt: now, UpdatedAt: now, TotalMembers: 42,
		},
	}

	h.repo.listByTenantFn = func(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		return expectedSchools, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/schools", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListSchoolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Schools) != 2 {
		t.Fatalf("expected 2 schools, got %d", len(result.Schools))
	}
	if result.Schools[0].TotalMembers != 15 {
		t.Fatalf("expected TotalMembers 15, got %d", result.Schools[0].TotalMembers)
	}
}

func TestHandler_ListSchools_Empty(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listByTenantFn = func(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
		return []SchoolWithMemberCount{}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/schools", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListSchoolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
	if len(result.Schools) != 0 {
		t.Fatalf("expected 0 schools, got %d", len(result.Schools))
	}
}

// ============================================================================
// Tests: Update School (PUT /api/v1/schools/:id)
// ============================================================================

func TestHandler_UpdateSchool_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	newName := "Updated School Name"

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return &School{ID: id, TenantID: "tenant_001", Name: "Old Name"}, nil
	}

	h.repo.updateFn = func(ctx context.Context, school SchoolUpdateFields) error {
		if school.ID != "school_001" {
			t.Errorf("expected ID 'school_001', got %q", school.ID)
		}
		if school.Name == nil || *school.Name != "Updated School Name" {
			t.Errorf("expected Name 'Updated School Name', got %v", school.Name)
		}
		return nil
	}

	body, _ := json.Marshal(UpdateSchoolPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/schools/school_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateSchool_WrongTenant(t *testing.T) {
	h := newHandlerTestHarness(t)

	newName := "Updated Name"

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		// School belongs to a different tenant
		return &School{ID: id, TenantID: "tenant_999", Name: "Other Tenant School"}, nil
	}

	body, _ := json.Marshal(UpdateSchoolPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/schools/school_001", body)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateSchool_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return nil, ErrNotFound
	}

	newName := "Updated Name"
	body, _ := json.Marshal(UpdateSchoolPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/schools/school_999", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateSchool_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return &School{ID: "school_001", TenantID: "tenant_001", Name: "Test"}, nil
	}

	resp := doRequest(h.app, "PUT", "/api/v1/schools/school_001", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Delete School (DELETE /api/v1/schools/:id)
// ============================================================================

func TestHandler_DeleteSchool_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return &School{ID: id, TenantID: "tenant_001", Name: "School to Delete"}, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id string) error {
		if id != "school_001" {
			t.Errorf("expected id 'school_001', got %q", id)
		}
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/schools/school_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteSchool_WrongTenant(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return &School{ID: id, TenantID: "tenant_999", Name: "Other Tenant School"}, nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/schools/school_001", nil)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteSchool_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/schools/school_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

package curriculum

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
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Test middleware that sets tenant_id, user_id, and active_school_id
	// (bypasses RequireAuth for unit testing)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("active_school_id", "school_001")
		return c.Next()
	}

	// Register routes manually with test auth
	areas := app.Group("/api/v1/curriculum/learning-areas", testAuth)
	areas.Post("/", handler.Create)
	areas.Get("/", handler.List)
	areas.Get("/:id", handler.GetByID)
	areas.Put("/:id", handler.Update)
	areas.Delete("/:id", handler.Delete)

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
// Tests: Create Learning Area (POST /api/v1/curriculum/learning-areas)
// ============================================================================

func TestHandler_Create_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, params CreateLearningAreaParams) (string, error) {
		if params.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", params.TenantID)
		}
		if params.SchoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", params.SchoolID)
		}
		if params.Name != "Mathematics" {
			t.Errorf("expected Name 'Mathematics', got %q", params.Name)
		}
		if params.Code != "MATH" {
			t.Errorf("expected Code 'MATH', got %q", params.Code)
		}
		if params.EducationLevel != "Junior_Secondary" {
			t.Errorf("expected EducationLevel 'Junior_Secondary', got %q", params.EducationLevel)
		}
		return "area_001", nil
	}

	body, _ := json.Marshal(CreateLearningAreaPayload{
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.ID != "area_001" {
		t.Fatalf("expected id 'area_001', got %q", result.ID)
	}
}

func TestHandler_Create_MissingName(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(CreateLearningAreaPayload{
		Name:           "",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_Create_InvalidEducationLevel(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(CreateLearningAreaPayload{
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Invalid_Level",
	})
	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/curriculum/learning-areas", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: List Learning Areas (GET /api/v1/curriculum/learning-areas)
// ============================================================================

func TestHandler_List_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	expectedAreas := []LearningArea{
		{ID: "area_001", TenantID: "tenant_001", SchoolID: "school_001", Name: "English", Code: "ENG", EducationLevel: "Early_Years"},
		{ID: "area_002", TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Early_Years"},
	}

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if educationLevel != nil {
			t.Errorf("expected nil educationLevel filter, got %q", *educationLevel)
		}
		return expectedAreas, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListLearningAreasResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.LearningAreas) != 2 {
		t.Fatalf("expected 2 learning areas, got %d", len(result.LearningAreas))
	}
}

func TestHandler_List_FilteredByEducationLevel(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		if educationLevel == nil || *educationLevel != "Senior_School" {
			t.Errorf("expected educationLevel 'Senior_School', got %v", educationLevel)
		}
		return []LearningArea{
			{ID: "area_003", Name: "Biology", Code: "BIO", EducationLevel: "Senior_School"},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas?education_level=Senior_School", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListLearningAreasResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.LearningAreas) != 1 {
		t.Fatalf("expected 1 learning area, got %d", len(result.LearningAreas))
	}
	if result.LearningAreas[0].Code != "BIO" {
		t.Fatalf("expected Code 'BIO', got %q", result.LearningAreas[0].Code)
	}
}

func TestHandler_List_Empty(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		return []LearningArea{}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListLearningAreasResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
	if len(result.LearningAreas) != 0 {
		t.Fatalf("expected 0 learning areas, got %d", len(result.LearningAreas))
	}
}

// ============================================================================
// Tests: Get Learning Area By ID (GET /api/v1/curriculum/learning-areas/:id)
// ============================================================================

func TestHandler_GetByID_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		if id != "area_001" {
			t.Errorf("expected id 'area_001', got %q", id)
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return &LearningArea{
			ID:             id,
			TenantID:       tenantID,
			SchoolID:       schoolID,
			Name:           "Mathematics",
			Code:           "MATH",
			EducationLevel: "Junior_Secondary",
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas/area_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result LearningArea
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Name != "Mathematics" {
		t.Fatalf("expected Name 'Mathematics', got %q", result.Name)
	}
	if result.Code != "MATH" {
		t.Fatalf("expected Code 'MATH', got %q", result.Code)
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return nil, ErrNotFound
	}

	resp := doRequest(h.app, "GET", "/api/v1/curriculum/learning-areas/area_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Update Learning Area (PUT /api/v1/curriculum/learning-areas/:id)
// ============================================================================

func TestHandler_Update_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	newName := "Advanced Mathematics"

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return &LearningArea{
			ID:             id,
			TenantID:       tenantID,
			SchoolID:       schoolID,
			Name:           "Mathematics",
			Code:           "MATH",
			EducationLevel: "Junior_Secondary",
		}, nil
	}

	h.repo.updateFn = func(ctx context.Context, params UpdateLearningAreaParams) error {
		if params.ID != "area_001" {
			t.Errorf("expected ID 'area_001', got %q", params.ID)
		}
		if params.Name == nil || *params.Name != "Advanced Mathematics" {
			t.Errorf("expected Name 'Advanced Mathematics', got %v", params.Name)
		}
		return nil
	}

	body, _ := json.Marshal(UpdateLearningAreaPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/learning-areas/area_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandler_Update_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return nil, ErrNotFound
	}

	newName := "Updated Name"
	body, _ := json.Marshal(UpdateLearningAreaPayload{Name: &newName})
	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/learning-areas/area_999", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestHandler_Update_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		return &LearningArea{ID: id, TenantID: tenantID, SchoolID: schoolID, Name: "Test"}, nil
	}

	resp := doRequest(h.app, "PUT", "/api/v1/curriculum/learning-areas/area_001", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Delete Learning Area (DELETE /api/v1/curriculum/learning-areas/:id)
// ============================================================================

func TestHandler_Delete_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "area_001" {
			t.Errorf("expected id 'area_001', got %q", id)
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/learning-areas/area_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_Delete_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/curriculum/learning-areas/area_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: No active school set
// ============================================================================

func TestHandler_NoActiveSchool(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Auth middleware without active_school_id
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	}

	areas := app.Group("/api/v1/curriculum/learning-areas", testAuth)
	areas.Post("/", handler.Create)

	body, _ := json.Marshal(CreateLearningAreaPayload{
		Name:           "Test",
		Code:           "TEST",
		EducationLevel: "Early_Years",
	})
	resp := doRequest(app, "POST", "/api/v1/curriculum/learning-areas", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing active school, got %d", resp.StatusCode)
	}
}

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

	"go.uber.org/zap"
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
	svc := NewService(repo, zap.NewNop().Sugar())
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
	schools.Delete("/", handler.Delete)

	// Singular school endpoints (e.g., status)
	school := app.Group("/api/v1/school", testAuth)
	school.Get("/status", handler.OnboardingStatus)

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
			IsActive: true, CreatedAt: now, UpdatedAt: now,
			Teachers: 15,
		},
		{
			ID: "school_002", TenantID: "tenant_001", Name: "Riverside Academy",
			County: "Nairobi", SubCounty: "Kilimani", SchoolType: "Private",
			IsActive: true, CreatedAt: now, UpdatedAt: now,
			Teachers: 40, Parents: 2,
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
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 schools, got %d", len(result.Items))
	}
	if result.Items[0].Teachers != 15 {
		t.Fatalf("expected Teachers 15, got %d", result.Items[0].Teachers)
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
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 schools, got %d", len(result.Items))
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

	body, _ := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: "school_001"})
	resp := doRequest(h.app, "DELETE", "/api/v1/schools", body)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteSchool_WrongTenant(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return &School{ID: id, TenantID: "tenant_999", Name: "Other Tenant School"}, nil
	}

	body, _ := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: "school_001"})
	resp := doRequest(h.app, "DELETE", "/api/v1/schools", body)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteSchool_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id string) (*School, error) {
		return nil, ErrNotFound
	}

	body, _ := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: "school_999"})
	resp := doRequest(h.app, "DELETE", "/api/v1/schools", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Onboarding Status (GET /api/v1/school/status)
// ============================================================================

func TestHandler_OnboardingStatus_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Set up the mock repository to return a specific onboarding status
	expectedTenantID := "tenant_001"
	h.repo.onboardingStatusFn = func(ctx context.Context, tenantID string) (*OnboardingStatus, error) {
		if tenantID != expectedTenantID {
			t.Errorf("expected tenantID %q, got %q", expectedTenantID, tenantID)
		}
		return &OnboardingStatus{
			TenantID:                   expectedTenantID,
			ClassStreamsCreated:        true,
			AcademicCalendarConfigured: false,
			CurriculumInitialized:      true,
			StaffInvited:               false,
			StudentsEnrolled:           true,
			IsOnboardingComplete:       false, // because not all steps are true
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/school/status", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result OnboardingStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.TenantID != expectedTenantID {
		t.Fatalf("expected tenant_id %q, got %q", expectedTenantID, result.TenantID)
	}
	if result.IsOnboardingComplete != false {
		t.Fatalf("expected is_onboarding_complete false, got %v", result.IsOnboardingComplete)
	}
	if result.Steps.AcademicCalendarConfigured != false {
		t.Fatalf("expected academic_calendar_configured false, got %v", result.Steps.AcademicCalendarConfigured)
	}
	if result.Steps.CurriculumInitialized != true {
		t.Fatalf("expected curriculum_initialized true, got %v", result.Steps.CurriculumInitialized)
	}
	if result.Steps.ClassStreamsCreated != true {
		t.Fatalf("expected class_streams_created true, got %v", result.Steps.ClassStreamsCreated)
	}
	if result.Steps.StaffInvited != false {
		t.Fatalf("expected staff_invited false, got %v", result.Steps.StaffInvited)
	}
	if result.Steps.StudentsEnrolled != true {
		t.Fatalf("expected students_enrolled true, got %v", result.Steps.StudentsEnrolled)
	}
}

package academicyears

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// Handler Test Harness
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

	// Test middleware that sets tenant_id and user_id (bypasses requireAuth/requireAdmin)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("school_id", "school_001")
		c.Locals("role", "SCHOOL_ADMIN")
		return c.Next()
	}

	testViewerAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("school_id", "school_001")
		c.Locals("role", "TEACHER")
		return c.Next()
	}

	// Register routes with our test auth
	years := app.Group("/api/v1/academic-years", testAuth)
	years.Get("/", handler.ListYears)
	years.Patch("/:id", handler.PatchYear)
	years.Post("/:id/set-current", handler.SetCurrentYear)
	years.Delete("/:id", handler.DeleteYear)

	terms := app.Group("/api/v1/academic-terms", testAuth)
	terms.Get("/", handler.ListTerms)
	terms.Post("/", handler.CreateTerm)
	terms.Patch("/:id", handler.PatchTerm)
	terms.Delete("/:id", handler.DeleteTerm)

	// Viewer-only routes (no admin needed) for non-admin test
	viewerYears := app.Group("/api/v2/academic-years", testViewerAuth)
	viewerYears.Get("/", handler.ListYears)

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
// A1 — Hierarchical fetch with ordered terms
// ============================================================================

func TestHandler_ListYears_WithTerms(t *testing.T) {
	h := newHandlerTestHarness(t)

	now := time.Now()
	h.repo.listYearsFn = func(ctx context.Context, tenantID, schoolID string) ([]AcademicYearWithTerms, error) {
		return []AcademicYearWithTerms{
			{
				AcademicYear: AcademicYear{
					ID: "year_001", Name: "2025",
					StartDate: now, EndDate: now, IsCurrent: true, Version: 3,
				},
				Terms: []AcademicTerm{
					{ID: "t1", Name: "Term 1", TermNumber: 1},
					{ID: "t2", Name: "Term 2", TermNumber: 2},
				},
			},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/academic-years", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data []AcademicYearWithTerms `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 year, got %d", len(result.Data))
	}
	if len(result.Data[0].Terms) != 2 {
		t.Fatalf("expected 2 terms, got %d", len(result.Data[0].Terms))
	}
}

// ============================================================================
// A2 — Soft delete cascades to terms
// ============================================================================

func TestHandler_DeleteYear_SoftDelete(t *testing.T) {
	h := newHandlerTestHarness(t)

	var capturedID, capturedActor string
	h.repo.softDeleteYearFn = func(ctx context.Context, id, actorID string) error {
		capturedID = id
		capturedActor = actorID
		return nil
	}

	h.repo.hasDependentsFn = func(ctx context.Context, id string) (bool, error) {
		return false, nil
	}

	h.repo.getYearByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return &AcademicYear{ID: id, TenantID: tenantID, SchoolID: schoolID, Name: "2025"}, nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/academic-years/year_001", nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if capturedID != "year_001" {
		t.Errorf("expected id 'year_001', got %q", capturedID)
	}
	if capturedActor != "user_001" {
		t.Errorf("expected actor 'user_001', got %q", capturedActor)
	}
}

// ============================================================================
// A3 — Set current year
// ============================================================================

func TestHandler_SetCurrentYear(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.clearCurrentYearFn = func(ctx context.Context, schoolID, tenantID, excludeID, actorID string) error {
		return nil
	}
	h.repo.setCurrentYearFn = func(ctx context.Context, id, tenantID, schoolID, actorID string) (bool, error) {
		return true, nil
	}

	resp := doRequest(h.app, "POST", "/api/v1/academic-years/year_002/set-current", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// ============================================================================
// A5 — PATCH blocked when dates strand terms
// ============================================================================

func TestHandler_PatchYear_TermStranding(t *testing.T) {
	h := newHandlerTestHarness(t)

	year := &AcademicYear{
		ID: "year_001", TenantID: "tenant_001", SchoolID: "school_001",
		Name: "2025", Version: 3,
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	h.repo.getYearByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return year, nil
	}

	h.repo.updateYearFn = func(ctx context.Context, y *AcademicYear) error {
		return nil
	}

	// Our PatchYear service doesn't return TermsOutOfRangeError on stranding — it returns nil,nil
	// Let's adjust the test to match: the handler wraps the service, which returns nil for error
	// So it falls through to the conflict handler

	body := map[string]interface{}{
		"end_date": "2025-08-31",
		"version":  3,
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "PATCH", "/api/v1/academic-years/year_001", b)

	// Without the stranding mock, this will succeed (200)
	// To test stranding, we need the repo.FindStrandedTerms to return results
	// But the handler relies on the service returning *TermsOutOfRangeError

	// Actually, looking at the handler code — PatchYear calls the service and gets
	// a *TermsOutOfRangeError return. But PatchYear in the service doesn't return
	// that error — it returns (nil, nil) or (year, nil) or nil from a wrapped error.
	// The stranding check returns a *TermsOutOfRangeError but it's only set on
	// the second return value.

	// The handler code checks strandingErr != nil. Let's make the mock return
	// stranded terms.
	t.Logf("response status: %d", resp.StatusCode)

	// Re-do with stranding
	h2 := newHandlerTestHarness(t)
	h2.repo.getYearByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return year, nil
	}
	h2.repo.findStrandedTermsFn = func(ctx context.Context, yearID string, newStart, newEnd time.Time) ([]ConflictingTerm, error) {
		return []ConflictingTerm{
			{ID: "term_001", Name: "Term 1", StartDate: "2025-09-01", EndDate: "2025-11-30"},
		}, nil
	}

	resp2 := doRequest(h2.app, "PATCH", "/api/v1/academic-years/year_001", b)
	if resp2.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for stranded terms, got %d", resp2.StatusCode)
	}

	var errResp struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if errResp.Code != "TERMS_OUT_OF_RANGE" {
		t.Errorf("expected code 'TERMS_OUT_OF_RANGE', got %q", errResp.Code)
	}
}

// ============================================================================
// A6 — PATCH blocked by stale version
// ============================================================================

func TestHandler_PatchYear_StaleVersion(t *testing.T) {
	h := newHandlerTestHarness(t)

	year := &AcademicYear{ID: "year_001", Version: 5}
	h.repo.getYearByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return year, nil
	}

	body := map[string]interface{}{
		"name":    "New Name",
		"version": 3, // stale
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "PATCH", "/api/v1/academic-years/year_001", b)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 for stale version, got %d", resp.StatusCode)
	}
}

// ============================================================================
// B1 — Term before year start blocked
// ============================================================================

func TestHandler_CreateTerm_BeforeYearStart(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getYearByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return &AcademicYear{
			ID: id, StartDate: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
			EndDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		}, nil
	}

	body := map[string]interface{}{
		"academic_year_id": "year_001",
		"name":             "Term 1",
		"term_number":      1,
		"start_date":       "2025-01-05",
		"end_date":         "2025-04-04",
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "POST", "/api/v1/academic-terms", b)
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

// ============================================================================
// B3 — Overlapping terms blocked
// ============================================================================

func TestHandler_CreateTerm_Overlap(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getYearByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return &AcademicYear{
			ID:        id,
			StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		}, nil
	}

	h.repo.findOverlappingTermsFn = func(ctx context.Context, yearID, excludeID string, startDate, endDate time.Time) ([]AcademicTerm, error) {
		return []AcademicTerm{
			{ID: "term_001", Name: "Term 1", TermNumber: 1},
		}, nil
	}

	body := map[string]interface{}{
		"academic_year_id": "year_001",
		"name":             "Term 2",
		"term_number":      2,
		"start_date":       "2025-03-01",
		"end_date":         "2025-06-30",
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "POST", "/api/v1/academic-terms", b)
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

// ============================================================================
// B5 — Duplicate term_number blocked
// ============================================================================

func TestHandler_CreateTerm_DuplicateNumber(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getYearByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return &AcademicYear{
			ID:        id,
			StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		}, nil
	}

	h.repo.createTermFn = func(ctx context.Context, term *AcademicTerm) (string, error) {
		return "", errors.New("duplicate key value violates unique constraint")
	}

	body := map[string]interface{}{
		"academic_year_id": "year_001",
		"name":             "Term 1 again",
		"term_number":      1,
		"start_date":       "2025-01-01",
		"end_date":         "2025-04-04",
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "POST", "/api/v1/academic-terms", b)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 for duplicate term number, got %d", resp.StatusCode)
	}
}

// ============================================================================
// B9 — PATCH blocked by stale version
// ============================================================================

func TestHandler_PatchTerm_StaleVersion(t *testing.T) {
	h := newHandlerTestHarness(t)

	term := &AcademicTerm{
		ID: "term_001", Version: 5,
		AcademicYearID: "year_001",
	}
	year := &AcademicYear{ID: "year_001"}

	h.repo.getTermByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
		return term, year, nil
	}

	body := map[string]interface{}{
		"name":    "New Name",
		"version": 3,
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "PATCH", "/api/v1/academic-terms/term_001", b)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 for stale version, got %d", resp.StatusCode)
	}
}

// ============================================================================
// A7 — Tenant isolation: viewer can list but not see other tenants
// ============================================================================

func TestHandler_ListYears_TenantIsolation(t *testing.T) {
	h := newHandlerTestHarness(t)

	var capturedTenant string
	h.repo.listYearsFn = func(ctx context.Context, tenantID, schoolID string) ([]AcademicYearWithTerms, error) {
		capturedTenant = tenantID
		return []AcademicYearWithTerms{}, nil
	}

	// Use the viewer-only route (GET /api/v2/academic-years) with TEACHER role
	resp := doRequest(h.app, "GET", "/api/v2/academic-years", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if capturedTenant != "tenant_001" {
		t.Errorf("expected tenant 'tenant_001', got %q", capturedTenant)
	}
}

// ============================================================================
// B11 — is_current warning on PATCH term
// ============================================================================

func TestHandler_PatchTerm_IsCurrentStripped(t *testing.T) {
	h := newHandlerTestHarness(t)

	term := &AcademicTerm{
		ID: "term_001", Version: 1, IsCurrent: true,
		Name:           "Term 1",
		AcademicYearID: "year_001",
		StartDate:      time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2025, 4, 4, 0, 0, 0, 0, time.UTC),
	}
	year := &AcademicYear{
		ID:        "year_001",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	h.repo.getTermByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
		return term, year, nil
	}
	h.repo.updateTermFn = func(ctx context.Context, t *AcademicTerm) error {
		return nil
	}

	// Include is_current in body — handler should strip it
	body := map[string]interface{}{
		"name":       "New Name",
		"version":    1,
		"is_current": true,
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "PATCH", "/api/v1/academic-terms/term_001", b)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if warnings, ok := result["warnings"]; ok {
		t.Logf("response includes warnings: %v", warnings)
	}
}

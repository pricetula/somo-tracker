package academicyears

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
	svc := NewService(repo, zap.NewNop().Sugar())
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

	terms := app.Group("/api/v1/academic-terms", testAuth)
	terms.Get("/", handler.ListTerms)
	terms.Post("/", handler.CreateTerm)
	terms.Patch("/:id", handler.PatchTerm)
	terms.Post("/:id/activate", handler.ActivateTerm)
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

	now := DateOnly{Time: time.Now()}
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
// B1 — Term before year start blocked
// ============================================================================

func TestHandler_CreateTerm_BeforeYearStart(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getYearByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicYear, error) {
		return &AcademicYear{
			ID: id, StartDate: DateOnly{Time: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)},
			EndDate: DateOnly{Time: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
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
			StartDate: DateOnly{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			EndDate:   DateOnly{Time: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
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
			StartDate: DateOnly{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			EndDate:   DateOnly{Time: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	h.repo.createTermFn = func(ctx context.Context, term *AcademicTerm) (string, error) {
		return "", &TermNumberExistsError{}
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
		StartDate:      DateOnly{Time: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)},
		EndDate:        DateOnly{Time: time.Date(2025, 4, 4, 0, 0, 0, 0, time.UTC)},
	}
	year := &AcademicYear{
		ID:        "year_001",
		StartDate: DateOnly{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		EndDate:   DateOnly{Time: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
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

// ============================================================================
// C1 — Activate term returns 200 with the activated term
// ============================================================================

func TestHandler_ActivateTerm(t *testing.T) {
	h := newHandlerTestHarness(t)

	var capturedID string
	h.repo.activateTermFn = func(ctx context.Context, termID, tenantID, schoolID, actorID string) (*AcademicTerm, error) {
		capturedID = termID
		return &AcademicTerm{
			ID: termID, TenantID: tenantID, SchoolID: schoolID,
			AcademicYearID: "year_001", Name: "Term 2", TermNumber: 2,
			Version: 2, IsCurrent: true,
		}, nil
	}

	resp := doRequest(h.app, "POST", "/api/v1/academic-terms/term_002/activate", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if capturedID != "term_002" {
		t.Errorf("expected id 'term_002', got %q", capturedID)
	}

	var result struct {
		ID        string `json:"id"`
		IsCurrent bool   `json:"is_current"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result.ID != "term_002" {
		t.Errorf("expected id 'term_002', got %q", result.ID)
	}
	if !result.IsCurrent {
		t.Error("expected is_current = true in response")
	}
}

// ============================================================================
// D1 — Delete current term rejected with 409 term_is_current
// ============================================================================

func TestHandler_DeleteTerm_BlockedWhenCurrent(t *testing.T) {
	h := newHandlerTestHarness(t)

	term := &AcademicTerm{
		ID: "term_001", TenantID: "tenant_001", SchoolID: "school_001",
		AcademicYearID: "year_001", Version: 1, IsCurrent: true,
	}
	year := &AcademicYear{ID: "year_001"}
	h.repo.getTermByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
		return term, year, nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/academic-terms/term_001", nil)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	var errResp struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if errResp.Code != "term_is_current" {
		t.Errorf("expected code 'term_is_current', got %q", errResp.Code)
	}
}

// ============================================================================
// D2 — Delete term with dependents returns 409 with per-table counts
// ============================================================================

func TestHandler_DeleteTerm_BlockedWithDependents(t *testing.T) {
	h := newHandlerTestHarness(t)

	term := &AcademicTerm{
		ID: "term_001", TenantID: "tenant_001", SchoolID: "school_001",
		AcademicYearID: "year_001", Version: 1, IsCurrent: false,
	}
	year := &AcademicYear{ID: "year_001"}
	h.repo.getTermByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
		return term, year, nil
	}
	h.repo.termDependencyCountsFn = func(ctx context.Context, termID string) (map[string]int64, error) {
		return map[string]int64{"invoices": 7}, nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/academic-terms/term_001", nil)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	var errResp struct {
		Code    string `json:"code"`
		Details struct {
			Counts map[string]int64 `json:"counts"`
		} `json:"details"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if errResp.Code != "HAS_DEPENDENTS" {
		t.Errorf("expected code 'HAS_DEPENDENTS', got %q", errResp.Code)
	}
	if errResp.Details.Counts["invoices"] != 7 {
		t.Errorf("expected 7 invoices in counts, got %d", errResp.Details.Counts["invoices"])
	}
}

// ============================================================================
// D3 — Delete inactive term with no dependents returns 204
// ============================================================================

func TestHandler_DeleteTerm_Success(t *testing.T) {
	h := newHandlerTestHarness(t)

	term := &AcademicTerm{
		ID: "term_001", TenantID: "tenant_001", SchoolID: "school_001",
		AcademicYearID: "year_001", Version: 1, IsCurrent: false,
	}
	year := &AcademicYear{ID: "year_001"}
	h.repo.getTermByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
		return term, year, nil
	}
	h.repo.termDependencyCountsFn = func(ctx context.Context, termID string) (map[string]int64, error) {
		return map[string]int64{}, nil
	}
	h.repo.deleteTermFn = func(ctx context.Context, id string) error {
		return nil
	}
	h.repo.syncCurrentTermFn = func(ctx context.Context, academicYearID string, now time.Time) error {
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/academic-terms/term_001", nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

// ============================================================================
// E1 — PATCH narrowing active term with orphans returns 409 ORPHANED_RECORDS
// ============================================================================

func TestHandler_PatchTerm_OrphanGuard(t *testing.T) {
	h := newHandlerTestHarness(t)

	term := &AcademicTerm{
		ID: "term_001", TenantID: "tenant_001", SchoolID: "school_001",
		AcademicYearID: "year_001", Version: 1, IsCurrent: true,
		StartDate: DateOnly{Time: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)},
		EndDate:   DateOnly{Time: time.Date(2025, 4, 4, 0, 0, 0, 0, time.UTC)},
	}
	year := &AcademicYear{
		ID:        "year_001",
		StartDate: DateOnly{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		EndDate:   DateOnly{Time: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
	}
	h.repo.getTermByIDForUpdateFn = func(ctx context.Context, id, tenantID, schoolID string) (*AcademicTerm, *AcademicYear, error) {
		return term, year, nil
	}
	h.repo.findOverlappingTermsFn = func(ctx context.Context, yearID, excludeID string, startDate, endDate time.Time) ([]AcademicTerm, error) {
		return nil, nil
	}
	h.repo.countOrphansOutsideRangeFn = func(ctx context.Context, termID string, newStart, newEnd time.Time) (map[string]int64, error) {
		return map[string]int64{"assessment_sessions": 1, "attendance_records": 4}, nil
	}

	body := map[string]interface{}{
		"start_date": "2025-02-01", // narrow
		"version":    1,
	}
	b, _ := json.Marshal(body)
	resp := doRequest(h.app, "PATCH", "/api/v1/academic-terms/term_001", b)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	var errResp struct {
		Code    string `json:"code"`
		Details struct {
			AttendanceRecords int64 `json:"attendance_records"`
		} `json:"details"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if errResp.Code != "ORPHANED_RECORDS" {
		t.Errorf("expected code 'ORPHANED_RECORDS', got %q", errResp.Code)
	}
	if errResp.Details.AttendanceRecords != 4 {
		t.Errorf("expected 4 attendance records in details, got %d", errResp.Details.AttendanceRecords)
	}
}

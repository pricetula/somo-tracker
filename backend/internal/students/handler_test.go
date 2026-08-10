package students

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// Mock Import Service Adapter
// ============================================================================

type mockImportServiceAdapter struct {
	createJobFn func(ctx context.Context, req imports.CreateJobRequest) (*imports.CreateJobResponse, error)
}

func (m *mockImportServiceAdapter) CreateJob(ctx context.Context, req imports.CreateJobRequest) (*imports.CreateJobResponse, error) {
	if m.createJobFn != nil {
		return m.createJobFn(ctx, req)
	}
	return &imports.CreateJobResponse{
		JobID:        uuid.New(),
		TotalRecords: int64(len(req.Rows)),
		TotalChunks:  1,
		Status:       imports.ImportJobStatusProcessing,
	}, nil
}

// ============================================================================
// Mock Academic Years Service Adapter
// ============================================================================

type mockAcademicYearsAdapter struct {
	getCurrentAcademicYearIDFn func(ctx context.Context, tenantID, schoolID string) (string, error)
	getCurrentAcademicTermIDFn func(ctx context.Context, academicYearID string) (string, error)
}

func (m *mockAcademicYearsAdapter) GetCurrentAcademicYearID(ctx context.Context, tenantID, schoolID string) (string, error) {
	if m.getCurrentAcademicYearIDFn != nil {
		return m.getCurrentAcademicYearIDFn(ctx, tenantID, schoolID)
	}
	return "year_001", nil
}

func (m *mockAcademicYearsAdapter) GetCurrentAcademicTermID(ctx context.Context, academicYearID string) (string, error) {
	if m.getCurrentAcademicTermIDFn != nil {
		return m.getCurrentAcademicTermIDFn(ctx, academicYearID)
	}
	return "term_001", nil
}

// ============================================================================
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app               *fiber.App
	handler           *Handler
	impMock           *mockImportServiceAdapter
	academicYearsMock *mockAcademicYearsAdapter
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	svc := &Service{}
	handler := NewHandler(svc)

	impMock := &mockImportServiceAdapter{}
	handler.SetImportService(impMock)

	academicYearsMock := &mockAcademicYearsAdapter{}
	handler.SetAcademicYearsService(academicYearsMock)

	app := fiber.New()

	// Test middleware that sets tenant_id, user_id, and active_school_id
	// (bypasses requireAuth for testing)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("tenant_id", uuid.New().String())
		c.Locals("user_id", uuid.New().String())
		c.Locals("active_school_id", uuid.New().String())
		return c.Next()
	})

	// Register only the import route (the one we're testing)
	students := app.Group("/api/v1/students")
	students.Post("/import", handler.BulkImport)

	return &handlerTestHarness{
		app:               app,
		handler:           handler,
		impMock:           impMock,
		academicYearsMock: academicYearsMock,
	}
}

// doImportRequest is a helper to POST to /api/v1/students/import.
func (h *handlerTestHarness) doImportRequest(t *testing.T, rows []ImportRow) *http.Response {
	t.Helper()

	body := ImportRequest{Rows: rows}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/students/import", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.app.Test(req, 5000) // 5s timeout
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

// decodeError decodes the canonical error response.
func decodeError(t *testing.T, resp *http.Response) struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
} {
	t.Helper()
	var result struct {
		Code    string              `json:"code"`
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return result
}

// ============================================================================
// Tests: Row limit enforcement (BulkImport)
// ============================================================================

func TestBulkImport_ZeroRows_ReturnsInvalidInput(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := h.doImportRequest(t, []ImportRow{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty rows, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody.Code != "invalid_input" {
		t.Fatalf("expected code 'invalid_input', got %q", errBody.Code)
	}
	if !strings.Contains(errBody.Message, "empty") {
		t.Fatalf("expected message about empty rows, got %q", errBody.Message)
	}
}

func TestBulkImport_ExactlyMaxImportRows_Succeeds(t *testing.T) {
	h := newHandlerTestHarness(t)

	rows := make([]ImportRow, imports.MaxImportRows)
	for i := 0; i < imports.MaxImportRows; i++ {
		rows[i] = ImportRow{
			FullName: fmt.Sprintf("Student %d", i),
			Gender:   "M",
		}
	}

	resp := h.doImportRequest(t, rows)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for MaxImportRows rows, got %d", resp.StatusCode)
	}
}

func TestBulkImport_ExceedsMaxImportRows_ReturnsRowLimitExceeded(t *testing.T) {
	h := newHandlerTestHarness(t)

	rows := make([]ImportRow, imports.MaxImportRows+1)
	for i := 0; i < imports.MaxImportRows+1; i++ {
		rows[i] = ImportRow{
			FullName: fmt.Sprintf("Student %d", i),
			Gender:   "M",
		}
	}

	resp := h.doImportRequest(t, rows)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for exceeding MaxImportRows, got %d", resp.StatusCode)
	}

	errBody := decodeError(t, resp)
	if errBody.Code != "import_row_limit_exceeded" {
		t.Fatalf("expected code 'import_row_limit_exceeded', got %q", errBody.Code)
	}

	expectedMsg := fmt.Sprintf("Import contains %d rows; the maximum is %d",
		imports.MaxImportRows+1, imports.MaxImportRows)
	if !strings.Contains(errBody.Message, expectedMsg) {
		t.Fatalf("expected message containing %q, got %q", expectedMsg, errBody.Message)
	}
}

func TestBulkImport_WellBelowLimit_Succeeds(t *testing.T) {
	h := newHandlerTestHarness(t)

	rows := []ImportRow{
		{FullName: "Alice Wanjiku", Gender: "F"},
		{FullName: "Bob Kiplagat", Gender: "M"},
	}

	resp := h.doImportRequest(t, rows)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for small import, got %d", resp.StatusCode)
	}
}

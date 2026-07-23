package cbcclasses

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Test Harness
// ============================================================================

// mockAcademicYearsAdapter implements academicYearsAdapter for testing.
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

type handlerTestHarness struct {
	app               *fiber.App
	svc               *Service
	repo              *MockRepository
	handler           *Handler
	academicYearsMock *mockAcademicYearsAdapter
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	academicYearsMock := &mockAcademicYearsAdapter{}
	handler.SetAcademicYearsService(academicYearsMock)

	app := fiber.New()

	// Test middleware that sets tenant_id and active_school_id (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("active_school_id", "school_001")
		return c.Next()
	}

	// Register routes manually with test auth
	classes := app.Group("/api/v1/classes", testAuth)
	classes.Get("/", handler.List)
	classes.Get("/:id", handler.Get)
	classes.Post("/", handler.Create)
	classes.Put("/:id", handler.Update)
	classes.Delete("/", handler.BulkDelete)

	// Enrollment routes
	classes.Get("/:id/roster", handler.GetRoster)
	classes.Post("/:id/enroll", handler.BatchEnroll)
	classes.Post("/:id/unenroll/:studentId", handler.UnenrollStudent)
	classes.Get("/:id/available-students", handler.GetAvailableStudents)

	return &handlerTestHarness{
		app:               app,
		svc:               svc,
		repo:              repo,
		handler:           handler,
		academicYearsMock: academicYearsMock,
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
// CL: List Classes (GET /api/v1/classes)
// ============================================================================

func TestHandler_ListClasses_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	expectedResult := &ClassListResult{
		Items: []Class{
			{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001", StudentCount: 32},
			{ID: "class_002", GradeLevel: "G4", StreamName: "Red", DisplayLabel: "G4 Red", StreamID: "stream_002", StudentCount: 28},
		},
		Total: 2,
		Page:  1,
		Limit: 50,
	}

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if filter.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", filter.TenantID)
		}
		if filter.SchoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", filter.SchoolID)
		}
		if filter.AcademicYearID != "year_001" {
			t.Errorf("expected AcademicYearID 'year_001', got %q", filter.AcademicYearID)
		}
		if filter.AcademicTermID != "term_001" {
			t.Errorf("expected AcademicTermID 'term_001', got %q", filter.AcademicTermID)
		}
		return expectedResult, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL1: expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL1: failed to decode response: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("CL1: expected 2 classes, got %d", len(result.Items))
	}
	if result.Items[0].DisplayLabel != "G4 Blue" {
		t.Fatalf("CL1: expected display_label 'G4 Blue', got %q", result.Items[0].DisplayLabel)
	}
	if result.Items[0].StudentCount != 32 {
		t.Fatalf("CL1: expected student_count 32, got %d", result.Items[0].StudentCount)
	}
}

func TestHandler_ListClasses_Metadata(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if filter.Page != 2 {
			t.Errorf("expected page 2, got %d", filter.Page)
		}
		if filter.Limit != 10 {
			t.Errorf("expected limit 10, got %d", filter.Limit)
		}
		return &ClassListResult{
			Items: []Class{{ID: "class_003", GradeLevel: "G5", StreamName: "Green", DisplayLabel: "G5 Green", StreamID: "stream_003"}},
			Total: 25,
			Page:  2,
			Limit: 10,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001&page=2&limit=10", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL2: expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL2: failed to decode response: %v", err)
	}
	if result.Total != 25 {
		t.Fatalf("CL2: expected total 25, got %d", result.Total)
	}
	if result.Page != 2 {
		t.Fatalf("CL2: expected page 2, got %d", result.Page)
	}
	if result.Limit != 10 {
		t.Fatalf("CL2: expected limit 10, got %d", result.Limit)
	}
}

func TestHandler_ListClasses_WithSingleGradeLevelFilter(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if len(filter.GradeLevels) != 1 || filter.GradeLevels[0] != "G4" {
			t.Errorf("expected grade_levels ['G4'], got %v", filter.GradeLevels)
		}
		return &ClassListResult{
			Items: []Class{
				{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"},
			},
			Total: 1,
			Page:  1,
			Limit: 50,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001&grade_level=G4", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL6: expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL6: failed to decode response: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("CL6: expected 1 class, got %d", len(result.Items))
	}
}

func TestHandler_ListClasses_WithMultipleGradeLevelsFilter(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if len(filter.GradeLevels) != 2 || filter.GradeLevels[0] != "G7" || filter.GradeLevels[1] != "G8" {
			t.Errorf("expected grade_levels ['G7', 'G8'], got %v", filter.GradeLevels)
		}
		return &ClassListResult{
			Items: []Class{
				{ID: "class_001", GradeLevel: "G7", StreamName: "Blue", DisplayLabel: "G7 Blue", StreamID: "stream_001"},
				{ID: "class_002", GradeLevel: "G8", StreamName: "Red", DisplayLabel: "G8 Red", StreamID: "stream_002"},
			},
			Total: 2,
			Page:  1,
			Limit: 50,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001&grade_level=G7&grade_level=G8", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(result.Items))
	}
}

func TestHandler_ListClasses_WithSingleStreamIDFilter(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if len(filter.StreamIDs) != 1 || filter.StreamIDs[0] != "stream_001" {
			t.Errorf("expected stream_ids ['stream_001'], got %v", filter.StreamIDs)
		}
		return &ClassListResult{
			Items: []Class{
				{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"},
			},
			Total: 1,
			Page:  1,
			Limit: 50,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001&stream_id=stream_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL7: expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL7: failed to decode response: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("CL7: expected 1 class, got %d", len(result.Items))
	}
}

func TestHandler_ListClasses_WithMultipleStreamIDsFilter(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if len(filter.StreamIDs) != 2 || filter.StreamIDs[0] != "stream_001" || filter.StreamIDs[1] != "stream_002" {
			t.Errorf("expected stream_ids ['stream_001', 'stream_002'], got %v", filter.StreamIDs)
		}
		return &ClassListResult{
			Items: []Class{
				{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"},
				{ID: "class_002", GradeLevel: "G4", StreamName: "Red", DisplayLabel: "G4 Red", StreamID: "stream_002"},
			},
			Total: 2,
			Page:  1,
			Limit: 50,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001&stream_id=stream_001&stream_id=stream_002", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(result.Items))
	}
}

func TestHandler_ListClasses_EmptyResults(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		return &ClassListResult{
			Items: []Class{},
			Total: 0,
			Page:  1,
			Limit: 50,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL15: expected 200 OK, got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL15: failed to decode response: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("CL15: expected empty items array, got %d items", len(result.Items))
	}
}

func TestHandler_ListClasses_MissingAcademicYearID(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Simulate no current academic year configured
	h.academicYearsMock.getCurrentAcademicYearIDFn = func(ctx context.Context, tenantID, schoolID string) (string, error) {
		return "", nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_term_id=term_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL11: expected 200 OK (empty result), got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL11: failed to decode response: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("CL11: expected empty items, got %d items", len(result.Items))
	}
	if result.Total != 0 {
		t.Fatalf("CL11: expected total 0, got %d", result.Total)
	}
}

func TestHandler_ListClasses_MissingAcademicTermID(t *testing.T) {
	h := newHandlerTestHarness(t)

	// Simulate no current academic term configured
	h.academicYearsMock.getCurrentAcademicTermIDFn = func(ctx context.Context, academicYearID string) (string, error) {
		return "", nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CL12: expected 200 OK (empty result), got %d", resp.StatusCode)
	}

	var result ClassListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CL12: failed to decode response: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("CL12: expected empty items, got %d items", len(result.Items))
	}
	if result.Total != 0 {
		t.Fatalf("CL12: expected total 0, got %d", result.Total)
	}
}

// ============================================================================
// CC: Create Class (POST /api/v1/classes)
// ============================================================================

func TestHandler_CreateClass_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, params CreateClassParams) (*Class, error) {
		if params.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", params.TenantID)
		}
		if params.GradeLevel != "G4" {
			t.Errorf("expected GradeLevel 'G4', got %q", params.GradeLevel)
		}
		if len(params.StudentIDs) != 2 {
			t.Errorf("expected 2 student IDs, got %d", len(params.StudentIDs))
		}
		return &Class{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"}, nil
	}

	body, _ := json.Marshal(CreateClassPayload{
		GradeLevel:     "G4",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		StreamID:       "stream_001",
		StudentIDs:     []string{"student_001", "student_002"},
	})

	resp := doRequest(h.app, "POST", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("CC1/CC2: expected 201 Created, got %d", resp.StatusCode)
	}

	var result Class
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CC1/CC2: failed to decode response: %v", err)
	}
	if result.DisplayLabel != "G4 Blue" {
		t.Fatalf("CC2: expected display_label 'G4 Blue', got %q", result.DisplayLabel)
	}
}

func TestHandler_CreateClass_MissingRequiredField(t *testing.T) {
	h := newHandlerTestHarness(t)

	tests := []struct {
		name    string
		payload map[string]interface{}
		field   string
	}{
		{"grade_level", map[string]interface{}{"academic_year_id": "year_001", "academic_term_id": "term_001", "stream_id": "stream_001"}, "grade_level"},
		{"stream_id", map[string]interface{}{"grade_level": "G4", "academic_year_id": "year_001", "academic_term_id": "term_001"}, "stream_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			resp := doRequest(h.app, "POST", "/api/v1/classes", body)

			if resp.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("CC8: expected 422 for missing %s, got %d", tt.field, resp.StatusCode)
			}
		})
	}
}

func TestHandler_CreateClass_EmptyStudentIDs(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, params CreateClassParams) (*Class, error) {
		if len(params.StudentIDs) != 0 {
			t.Errorf("expected empty student IDs, got %d", len(params.StudentIDs))
		}
		return &Class{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"}, nil
	}

	body, _ := json.Marshal(CreateClassPayload{
		GradeLevel:     "G4",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		StreamID:       "stream_001",
		StudentIDs:     []string{},
	})

	resp := doRequest(h.app, "POST", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("CC9: expected 201 Created, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateClass_InvalidAcademicYear(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.validateAcademicYearFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	body, _ := json.Marshal(CreateClassPayload{
		GradeLevel:     "G4",
		AcademicYearID: "invalid_year",
		AcademicTermID: "term_001",
		StreamID:       "stream_001",
	})

	resp := doRequest(h.app, "POST", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("CC3: expected 422 for invalid academic_year_id, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateClass_InvalidStream(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.validateStreamFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	body, _ := json.Marshal(CreateClassPayload{
		GradeLevel:     "G4",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		StreamID:       "invalid_stream",
	})

	resp := doRequest(h.app, "POST", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("CC5/CC6: expected 422 for invalid stream, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateClass_DuplicateEntry(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, params CreateClassParams) (*Class, error) {
		return nil, ErrAlreadyExists
	}

	body, _ := json.Marshal(CreateClassPayload{
		GradeLevel:     "G4",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		StreamID:       "stream_001",
	})

	resp := doRequest(h.app, "POST", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("CC7: expected 409 Conflict, got %d", resp.StatusCode)
	}
}

// ============================================================================
// CU: Update Class (PUT /api/v1/classes/:id)
// ============================================================================

func TestHandler_UpdateClass_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFn = func(ctx context.Context, params UpdateClassParams) (*Class, error) {
		if params.ClassID != "class_001" {
			t.Errorf("expected ClassID 'class_001', got %q", params.ClassID)
		}
		if params.GradeLevel != "G5" {
			t.Errorf("expected GradeLevel 'G5', got %q", params.GradeLevel)
		}
		return &Class{ID: "class_001", GradeLevel: "G5", StreamName: "Green", DisplayLabel: "G5 Green", StreamID: "stream_002"}, nil
	}

	body, _ := json.Marshal(UpdateClassPayload{
		GradeLevel:     "G5",
		StreamID:       "stream_002",
		AcademicTermID: "term_001",
		StudentIDs:     []string{"student_001", "student_003"},
	})

	resp := doRequest(h.app, "PUT", "/api/v1/classes/class_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CU1/CU2: expected 200 OK, got %d", resp.StatusCode)
	}

	var result Class
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CU1/CU2: failed to decode response: %v", err)
	}
	if result.DisplayLabel != "G5 Green" {
		t.Fatalf("CU2: expected display_label 'G5 Green', got %q", result.DisplayLabel)
	}
}

func TestHandler_UpdateClass_MissingFields(t *testing.T) {
	h := newHandlerTestHarness(t)

	tests := []struct {
		name    string
		payload map[string]interface{}
		field   string
	}{
		{"grade_level", map[string]interface{}{"stream_id": "stream_001", "academic_term_id": "term_001"}, "grade_level"},
		{"stream_id", map[string]interface{}{"grade_level": "G4", "academic_term_id": "term_001"}, "stream_id"},
		{"academic_term_id", map[string]interface{}{"grade_level": "G4", "stream_id": "stream_001"}, "academic_term_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			resp := doRequest(h.app, "PUT", "/api/v1/classes/class_001", body)

			if resp.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("expected 422 for missing %s, got %d", tt.field, resp.StatusCode)
			}
		})
	}
}

func TestHandler_UpdateClass_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Class, error) {
		return nil, ErrNotFound
	}

	body, _ := json.Marshal(UpdateClassPayload{
		GradeLevel:     "G4",
		StreamID:       "stream_999",
		AcademicTermID: "term_001",
	})

	resp := doRequest(h.app, "PUT", "/api/v1/classes/class_999", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("CU8: expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateClass_DifferentialSync(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFn = func(ctx context.Context, params UpdateClassParams) (*Class, error) {
		// Verify differential sync params
		if len(params.StudentIDs) != 2 {
			t.Errorf("expected 2 student IDs for differential sync, got %d", len(params.StudentIDs))
		}
		return &Class{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"}, nil
	}

	body, _ := json.Marshal(UpdateClassPayload{
		GradeLevel:     "G4",
		StreamID:       "stream_001",
		AcademicTermID: "term_001",
		StudentIDs:     []string{"student_003", "student_004"},
	})

	resp := doRequest(h.app, "PUT", "/api/v1/classes/class_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("CU3-CU6: expected 200 OK, got %d", resp.StatusCode)
	}
}

// ============================================================================
// CD: Bulk Delete Classes (DELETE /api/v1/classes)
// ============================================================================

func TestHandler_BulkDeleteClasses_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.bulkDeleteFn = func(ctx context.Context, ids []string, tenantID, schoolID string) error {
		if len(ids) != 2 {
			t.Errorf("expected 2 ids, got %d", len(ids))
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		return nil
	}

	body, _ := json.Marshal(BulkDeletePayload{
		ClassIDs: []string{"class_001", "class_002"},
	})

	resp := doRequest(h.app, "DELETE", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("CD1: expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_BulkDeleteClasses_EmptyIDs(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(BulkDeletePayload{
		ClassIDs: []string{},
	})

	resp := doRequest(h.app, "DELETE", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("CD8: expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_BulkDeleteClasses_OverLimit(t *testing.T) {
	h := newHandlerTestHarness(t)

	ids := make([]string, 101)
	for i := 0; i < 101; i++ {
		ids[i] = "id"
	}

	body, _ := json.Marshal(BulkDeletePayload{
		ClassIDs: ids,
	})

	resp := doRequest(h.app, "DELETE", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("CD7: expected 400 Bad Request (LIMIT_EXCEEDED), got %d", resp.StatusCode)
	}

	var result struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("CD7: failed to decode response: %v", err)
	}
	if result.Error != "LIMIT_EXCEEDED" {
		t.Fatalf("CD7: expected error 'LIMIT_EXCEEDED', got %q", result.Error)
	}
}

func TestHandler_BulkDeleteClasses_NonExistentIDs(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.bulkDeleteFn = func(ctx context.Context, ids []string, tenantID, schoolID string) error {
		// Even with non-existent IDs, the service returns no error
		// (the WHERE clause simply matches nothing)
		return nil
	}

	body, _ := json.Marshal(BulkDeletePayload{
		ClassIDs: []string{"class_999", "class_998"},
	})

	resp := doRequest(h.app, "DELETE", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("CD9: expected 204 No Content (no-op), got %d", resp.StatusCode)
	}
}

func TestHandler_BulkDeleteClasses_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "DELETE", "/api/v1/classes", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid body, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Auth rejection tests (unauthenticated requests)
// ============================================================================

func testHandlerWithoutAuth(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New(fiber.Config{
		// Route middleware-returned sentinel errors (ErrUnauthorized etc.)
		// through the same mapping logic used in production.
		ErrorHandler: middleware.HTTPError,
	})

	// Register routes WITHOUT the test auth middleware — the real RequireAuth
	// will reject because there's no session in context.
	classes := app.Group("/api/v1/classes")
	classes.Get("/", middleware.RequireAuth, handler.List)
	classes.Post("/", middleware.RequireAuth, handler.Create)
	classes.Put("/:id", middleware.RequireAuth, handler.Update)
	classes.Delete("/", middleware.RequireAuth, handler.BulkDelete)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
	}
}

func TestHandler_ListClasses_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/classes?academic_year_id=year_001&academic_term_id=term_001", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("CL16: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateClass_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(CreateClassPayload{
		GradeLevel:     "G4",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		StreamID:       "stream_001",
	})

	resp := doRequest(h.app, "POST", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("CC12: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateClass_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(UpdateClassPayload{
		GradeLevel:     "G4",
		StreamID:       "stream_001",
		AcademicTermID: "term_001",
	})

	resp := doRequest(h.app, "PUT", "/api/v1/classes/class_001", body)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("CU13: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_BulkDeleteClasses_Unauthenticated(t *testing.T) {
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(BulkDeletePayload{
		ClassIDs: []string{"class_001"},
	})

	resp := doRequest(h.app, "DELETE", "/api/v1/classes", body)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("CD10: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

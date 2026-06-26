package cbcclasses

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listFn                  func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error)
	getByIDFn               func(ctx context.Context, id, tenantID, schoolID string) (*Class, error)
	createFn                func(ctx context.Context, params CreateClassParams) (*Class, error)
	updateFn                func(ctx context.Context, params UpdateClassParams) (*Class, error)
	bulkDeleteFn            func(ctx context.Context, ids []string, tenantID, schoolID string) error
	hasAssessmentSessionsFn func(ctx context.Context, classID, tenantID string) (bool, error)
	hasAnyAssessmentFn      func(ctx context.Context, classIDs []string, tenantID string) (bool, error)
	validateAcademicYearFn  func(ctx context.Context, id, tenantID, schoolID string) (bool, error)
	validateAcademicTermFn  func(ctx context.Context, id, academicYearID string) (bool, error)
	validateStreamFn        func(ctx context.Context, id, tenantID, schoolID string) (bool, error)
}

func (m *MockRepository) List(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &ClassListResult{Data: []Class{}, TotalRecords: 0, CurrentPage: 1, Limit: 50, TotalPages: 1}, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Class, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID, schoolID)
	}
	return &Class{ID: id, GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"}, nil
}

func (m *MockRepository) Create(ctx context.Context, params CreateClassParams) (*Class, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return &Class{ID: "class_001", GradeLevel: params.GradeLevel, StreamName: "Blue", DisplayLabel: params.GradeLevel + " Blue", StreamID: params.StreamID}, nil
}

func (m *MockRepository) Update(ctx context.Context, params UpdateClassParams) (*Class, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, params)
	}
	return &Class{ID: params.ClassID, GradeLevel: params.GradeLevel, StreamName: "Green", DisplayLabel: params.GradeLevel + " Green", StreamID: params.StreamID}, nil
}

func (m *MockRepository) BulkDelete(ctx context.Context, ids []string, tenantID, schoolID string) error {
	if m.bulkDeleteFn != nil {
		return m.bulkDeleteFn(ctx, ids, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) HasAssessmentSessions(ctx context.Context, classID, tenantID string) (bool, error) {
	if m.hasAssessmentSessionsFn != nil {
		return m.hasAssessmentSessionsFn(ctx, classID, tenantID)
	}
	return false, nil
}

func (m *MockRepository) HasAnyAssessmentSessions(ctx context.Context, classIDs []string, tenantID string) (bool, error) {
	if m.hasAnyAssessmentFn != nil {
		return m.hasAnyAssessmentFn(ctx, classIDs, tenantID)
	}
	return false, nil
}

func (m *MockRepository) ValidateAcademicYear(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	if m.validateAcademicYearFn != nil {
		return m.validateAcademicYearFn(ctx, id, tenantID, schoolID)
	}
	return true, nil
}

func (m *MockRepository) ValidateAcademicTerm(ctx context.Context, id, academicYearID string) (bool, error) {
	if m.validateAcademicTermFn != nil {
		return m.validateAcademicTermFn(ctx, id, academicYearID)
	}
	return true, nil
}

func (m *MockRepository) ValidateStream(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	if m.validateStreamFn != nil {
		return m.validateStreamFn(ctx, id, tenantID, schoolID)
	}
	return true, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc  *Service
	repo *MockRepository
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	svc := NewService(repo)
	return &testHarness{
		svc:  svc,
		repo: repo,
	}
}

// ============================================================================
// Tests: ListClasses
// ============================================================================

func TestListClasses_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedResult := &ClassListResult{
		Data: []Class{
			{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001", StudentCount: 32},
			{ID: "class_002", GradeLevel: "G4", StreamName: "Red", DisplayLabel: "G4 Red", StreamID: "stream_002", StudentCount: 28},
		},
		TotalRecords: 2,
		CurrentPage:  1,
		Limit:        50,
		TotalPages:   1,
	}

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		if filter.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", filter.TenantID)
		}
		if filter.AcademicYearID != "year_001" {
			t.Errorf("expected AcademicYearID 'year_001', got %q", filter.AcademicYearID)
		}
		if filter.AcademicTermID != "term_001" {
			t.Errorf("expected AcademicTermID 'term_001', got %q", filter.AcademicTermID)
		}
		return expectedResult, nil
	}

	result, err := h.svc.ListClasses(context.Background(), ClassListFilter{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(result.Data))
	}
	if result.Data[0].DisplayLabel != "G4 Blue" {
		t.Fatalf("expected display_label 'G4 Blue', got %q", result.Data[0].DisplayLabel)
	}
	if result.Data[0].StudentCount != 32 {
		t.Fatalf("expected student_count 32, got %d", result.Data[0].StudentCount)
	}
}

func TestListClasses_MissingTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListClasses(context.Background(), ClassListFilter{
		TenantID:       "",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListClasses_MissingSchoolID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListClasses(context.Background(), ClassListFilter{
		TenantID:       "tenant_001",
		SchoolID:       "",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListClasses_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listFn = func(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
		return &ClassListResult{Data: []Class{}, TotalRecords: 0, CurrentPage: 1, Limit: 50, TotalPages: 1}, nil
	}

	result, err := h.svc.ListClasses(context.Background(), ClassListFilter{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 0 {
		t.Fatalf("expected 0 classes, got %d", len(result.Data))
	}
}

// ============================================================================
// Tests: CreateClass
// ============================================================================

func TestCreateClass_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, params CreateClassParams) (*Class, error) {
		if params.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", params.TenantID)
		}
		if params.GradeLevel != "G4" {
			t.Errorf("expected GradeLevel 'G4', got %q", params.GradeLevel)
		}
		if params.StreamID != "stream_001" {
			t.Errorf("expected StreamID 'stream_001', got %q", params.StreamID)
		}
		return &Class{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"}, nil
	}

	class, err := h.svc.CreateClass(context.Background(), CreateClassParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if class.DisplayLabel != "G4 Blue" {
		t.Fatalf("expected DisplayLabel 'G4 Blue', got %q", class.DisplayLabel)
	}
}

func TestCreateClass_EmptyGradeLevel(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateClass(context.Background(), CreateClassParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		GradeLevel:     "",
		StreamID:       "stream_001",
	})
	if err == nil {
		t.Fatal("expected error for empty grade_level, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateClass_InvalidAcademicYear(t *testing.T) {
	h := newTestHarness()

	h.repo.validateAcademicYearFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	_, err := h.svc.CreateClass(context.Background(), CreateClassParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "invalid_year",
		AcademicTermID: "term_001",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
	})
	if err == nil {
		t.Fatal("expected error for invalid academic year, got nil")
	}
}

func TestCreateClass_InvalidAcademicTerm(t *testing.T) {
	h := newTestHarness()

	h.repo.validateAcademicTermFn = func(ctx context.Context, id, academicYearID string) (bool, error) {
		return false, nil
	}

	_, err := h.svc.CreateClass(context.Background(), CreateClassParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "invalid_term",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
	})
	if err == nil {
		t.Fatal("expected error for invalid academic term, got nil")
	}
}

func TestCreateClass_InvalidStream(t *testing.T) {
	h := newTestHarness()

	h.repo.validateStreamFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	_, err := h.svc.CreateClass(context.Background(), CreateClassParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		GradeLevel:     "G4",
		StreamID:       "invalid_stream",
	})
	if err == nil {
		t.Fatal("expected error for invalid stream, got nil")
	}
}

func TestCreateClass_WithStudents(t *testing.T) {
	h := newTestHarness()

	studentIDs := []string{"student_001", "student_002"}

	h.repo.createFn = func(ctx context.Context, params CreateClassParams) (*Class, error) {
		if len(params.StudentIDs) != 2 {
			t.Errorf("expected 2 student IDs, got %d", len(params.StudentIDs))
		}
		return &Class{ID: "class_001", GradeLevel: "G4", StreamName: "Blue", DisplayLabel: "G4 Blue", StreamID: "stream_001"}, nil
	}

	class, err := h.svc.CreateClass(context.Background(), CreateClassParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		AcademicYearID: "year_001",
		AcademicTermID: "term_001",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
		StudentIDs:     studentIDs,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if class.ID != "class_001" {
		t.Fatalf("expected ID 'class_001', got %q", class.ID)
	}
}

// ============================================================================
// Tests: UpdateClass
// ============================================================================

func TestUpdateClass_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFn = func(ctx context.Context, params UpdateClassParams) (*Class, error) {
		if params.ClassID != "class_001" {
			t.Errorf("expected ClassID 'class_001', got %q", params.ClassID)
		}
		if params.GradeLevel != "G5" {
			t.Errorf("expected GradeLevel 'G5', got %q", params.GradeLevel)
		}
		return &Class{ID: "class_001", GradeLevel: "G5", StreamName: "Green", DisplayLabel: "G5 Green", StreamID: "stream_002"}, nil
	}

	class, err := h.svc.UpdateClass(context.Background(), UpdateClassParams{
		ClassID:        "class_001",
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		GradeLevel:     "G5",
		StreamID:       "stream_002",
		AcademicTermID: "term_001",
		StudentIDs:     []string{"student_001"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if class.DisplayLabel != "G5 Green" {
		t.Fatalf("expected DisplayLabel 'G5 Green', got %q", class.DisplayLabel)
	}
}

func TestUpdateClass_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateClass(context.Background(), UpdateClassParams{
		ClassID:        "",
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for empty class id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateClass_LockedByAssessments(t *testing.T) {
	h := newTestHarness()

	h.repo.hasAssessmentSessionsFn = func(ctx context.Context, classID, tenantID string) (bool, error) {
		return true, nil
	}

	_, err := h.svc.UpdateClass(context.Background(), UpdateClassParams{
		ClassID:        "class_001",
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for locked class, got nil")
	}
	if !errors.Is(err, ErrClassLocked) {
		t.Fatalf("expected ErrClassLocked, got %v", err)
	}
}

func TestUpdateClass_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Class, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.UpdateClass(context.Background(), UpdateClassParams{
		ClassID:        "class_999",
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		GradeLevel:     "G4",
		StreamID:       "stream_001",
		AcademicTermID: "term_001",
	})
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: BulkDeleteClasses
// ============================================================================

func TestBulkDeleteClasses_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.bulkDeleteFn = func(ctx context.Context, ids []string, tenantID, schoolID string) error {
		if len(ids) != 2 {
			t.Errorf("expected 2 ids, got %d", len(ids))
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		return nil
	}

	err := h.svc.BulkDeleteClasses(context.Background(), []string{"class_001", "class_002"}, "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBulkDeleteClasses_EmptyIDs(t *testing.T) {
	h := newTestHarness()

	err := h.svc.BulkDeleteClasses(context.Background(), []string{}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty IDs, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBulkDeleteClasses_OverLimit(t *testing.T) {
	h := newTestHarness()

	ids := make([]string, 101)
	for i := 0; i < 101; i++ {
		ids[i] = "id"
	}

	err := h.svc.BulkDeleteClasses(context.Background(), ids, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for over limit, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBulkDeleteClasses_BlockedByAssessments(t *testing.T) {
	h := newTestHarness()

	h.repo.hasAnyAssessmentFn = func(ctx context.Context, classIDs []string, tenantID string) (bool, error) {
		return true, nil
	}

	err := h.svc.BulkDeleteClasses(context.Background(), []string{"class_001"}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for classes with assessments, got nil")
	}
	if !errors.Is(err, ErrClassHasAssessments) {
		t.Fatalf("expected ErrClassHasAssessments, got %v", err)
	}
}

func TestBulkDeleteClasses_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.BulkDeleteClasses(context.Background(), []string{"class_001"}, "", "school_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

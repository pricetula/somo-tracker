package teacherworkloadsummaries

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	refreshComputationFn func(ctx context.Context, academicYearID string) error
	listByTeacherFn      func(ctx context.Context, tenantID, schoolID, userID, yearID string) (*WorkloadSummaryListResponse, error)
	listByYearFn         func(ctx context.Context, tenantID, schoolID, yearID string) (*WorkloadSummaryListResponse, error)
	getByTeacherYearFn   func(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error)
}

func (m *MockRepository) RefreshComputation(ctx context.Context, academicYearID string) error {
	if m.refreshComputationFn != nil {
		return m.refreshComputationFn(ctx, academicYearID)
	}
	return nil
}

func (m *MockRepository) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, yearID string) (*WorkloadSummaryListResponse, error) {
	if m.listByTeacherFn != nil {
		return m.listByTeacherFn(ctx, tenantID, schoolID, userID, yearID)
	}
	return &WorkloadSummaryListResponse{Items: []TeacherWorkloadSummary{}, Total: 0}, nil
}

func (m *MockRepository) ListByYear(ctx context.Context, tenantID, schoolID, yearID string) (*WorkloadSummaryListResponse, error) {
	if m.listByYearFn != nil {
		return m.listByYearFn(ctx, tenantID, schoolID, yearID)
	}
	return &WorkloadSummaryListResponse{Items: []TeacherWorkloadSummary{}, Total: 0}, nil
}

func (m *MockRepository) GetByTeacherYear(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error) {
	if m.getByTeacherYearFn != nil {
		return m.getByTeacherYearFn(ctx, userID, yearID)
	}
	return &TeacherWorkloadSummary{UserID: userID, AcademicYearID: yearID}, nil
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
// Tests: RefreshComputation
// ============================================================================

func TestRefreshComputation_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.refreshComputationFn = func(ctx context.Context, yearID string) error {
		if yearID != "year_001" {
			t.Errorf("expected year_001, got %q", yearID)
		}
		return nil
	}

	err := h.svc.RefreshComputation(context.Background(), "year_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshComputation_EmptyYearID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.RefreshComputation(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty academic_year_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ListByTeacher
// ============================================================================

func TestListByTeacher_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := &WorkloadSummaryListResponse{
		Items: []TeacherWorkloadSummary{
			{
				UserID:               "teacher_001",
				AcademicYearID:       "year_001",
				TotalAssignedPeriods: 24,
				UniqueSubjects:       3,
				ClassesTaught:        4,
			},
		},
		Total: 1,
	}

	h.repo.listByTeacherFn = func(ctx context.Context, tenantID, schoolID, userID, yearID string) (*WorkloadSummaryListResponse, error) {
		if userID != "teacher_001" {
			t.Errorf("expected teacher_001, got %q", userID)
		}
		if yearID != "year_001" {
			t.Errorf("expected year_001, got %q", yearID)
		}
		return expected, nil
	}

	result, err := h.svc.ListByTeacher(context.Background(), "tenant_001", "school_001", "teacher_001", "year_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 summary, got %d", result.Total)
	}
	if result.Items[0].TotalAssignedPeriods != 24 {
		t.Fatalf("expected 24 assigned periods, got %d", result.Items[0].TotalAssignedPeriods)
	}
	if result.Items[0].UniqueSubjects != 3 {
		t.Fatalf("expected 3 unique subjects, got %d", result.Items[0].UniqueSubjects)
	}
	if result.Items[0].ClassesTaught != 4 {
		t.Fatalf("expected 4 classes, got %d", result.Items[0].ClassesTaught)
	}
}

func TestListByTeacher_EmptyInputs(t *testing.T) {
	h := newTestHarness()

	tests := []struct {
		name     string
		tenantID string
		schoolID string
		userID   string
		yearID   string
	}{
		{"empty tenant_id", "", "school_001", "teacher_001", "year_001"},
		{"empty school_id", "tenant_001", "", "teacher_001", "year_001"},
		{"empty user_id", "tenant_001", "school_001", "", "year_001"},
		{"empty year_id", "tenant_001", "school_001", "teacher_001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.ListByTeacher(context.Background(), tt.tenantID, tt.schoolID, tt.userID, tt.yearID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ============================================================================
// Tests: ListByYear
// ============================================================================

func TestListByYear_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := &WorkloadSummaryListResponse{
		Items: []TeacherWorkloadSummary{
			{UserID: "teacher_001", AcademicYearID: "year_001", TotalAssignedPeriods: 24, UniqueSubjects: 3, ClassesTaught: 4},
			{UserID: "teacher_002", AcademicYearID: "year_001", TotalAssignedPeriods: 20, UniqueSubjects: 2, ClassesTaught: 3},
		},
		Total: 2,
	}

	h.repo.listByYearFn = func(ctx context.Context, tenantID, schoolID, yearID string) (*WorkloadSummaryListResponse, error) {
		if yearID != "year_001" {
			t.Errorf("expected year_001, got %q", yearID)
		}
		return expected, nil
	}

	result, err := h.svc.ListByYear(context.Background(), "tenant_001", "school_001", "year_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 summaries, got %d", result.Total)
	}
}

func TestListByYear_EmptyInputs(t *testing.T) {
	h := newTestHarness()

	tests := []struct {
		name     string
		tenantID string
		schoolID string
		yearID   string
	}{
		{"empty tenant_id", "", "school_001", "year_001"},
		{"empty school_id", "tenant_001", "", "year_001"},
		{"empty year_id", "tenant_001", "school_001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.ListByYear(context.Background(), tt.tenantID, tt.schoolID, tt.yearID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ============================================================================
// Tests: GetByTeacherYear
// ============================================================================

func TestGetByTeacherYear_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getByTeacherYearFn = func(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error) {
		if userID != "teacher_001" {
			t.Errorf("expected teacher_001, got %q", userID)
		}
		if yearID != "year_001" {
			t.Errorf("expected year_001, got %q", yearID)
		}
		return &TeacherWorkloadSummary{
			UserID:               userID,
			AcademicYearID:       yearID,
			TotalAssignedPeriods: 24,
			UniqueSubjects:       3,
			ClassesTaught:        4,
		}, nil
	}

	summary, err := h.svc.GetByTeacherYear(context.Background(), "teacher_001", "year_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalAssignedPeriods != 24 {
		t.Fatalf("expected 24 assigned periods, got %d", summary.TotalAssignedPeriods)
	}
	if summary.UniqueSubjects != 3 {
		t.Fatalf("expected 3 unique subjects, got %d", summary.UniqueSubjects)
	}
}

func TestGetByTeacherYear_EmptyInputs(t *testing.T) {
	h := newTestHarness()

	tests := []struct {
		name   string
		userID string
		yearID string
	}{
		{"empty user_id", "", "year_001"},
		{"empty year_id", "teacher_001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.GetByTeacherYear(context.Background(), tt.userID, tt.yearID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetByTeacherYear_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getByTeacherYearFn = func(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetByTeacherYear(context.Background(), "teacher_999", "year_999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

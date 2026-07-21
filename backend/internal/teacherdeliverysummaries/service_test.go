package teacherdeliverysummaries

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	refreshComputationFn func(ctx context.Context, termID string) error
	listByTeacherFn      func(ctx context.Context, tenantID, schoolID, userID, termID string) (*DeliverySummaryListResponse, error)
	listByTermFn         func(ctx context.Context, tenantID, schoolID, termID string) (*DeliverySummaryListResponse, error)
	getByTeacherTermFn   func(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error)
}

func (m *MockRepository) RefreshComputation(ctx context.Context, termID string) error {
	if m.refreshComputationFn != nil {
		return m.refreshComputationFn(ctx, termID)
	}
	return nil
}

func (m *MockRepository) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string) (*DeliverySummaryListResponse, error) {
	if m.listByTeacherFn != nil {
		return m.listByTeacherFn(ctx, tenantID, schoolID, userID, termID)
	}
	return &DeliverySummaryListResponse{Items: []TeacherDeliverySummary{}, Total: 0}, nil
}

func (m *MockRepository) ListByTerm(ctx context.Context, tenantID, schoolID, termID string) (*DeliverySummaryListResponse, error) {
	if m.listByTermFn != nil {
		return m.listByTermFn(ctx, tenantID, schoolID, termID)
	}
	return &DeliverySummaryListResponse{Items: []TeacherDeliverySummary{}, Total: 0}, nil
}

func (m *MockRepository) GetByTeacherTerm(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error) {
	if m.getByTeacherTermFn != nil {
		return m.getByTeacherTermFn(ctx, userID, termID)
	}
	return &TeacherDeliverySummary{UserID: userID, AcademicTermID: termID}, nil
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

	h.repo.refreshComputationFn = func(ctx context.Context, termID string) error {
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return nil
	}

	err := h.svc.RefreshComputation(context.Background(), "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshComputation_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.RefreshComputation(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty term_id, got nil")
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

	expected := &DeliverySummaryListResponse{
		Items: []TeacherDeliverySummary{
			{
				UserID:             "teacher_001",
				AcademicTermID:     "term_001",
				TotalAssignedSlots: 180,
				MarkedSlots:        172,
				MissedSlots:        8,
				SessionsCreated:    180,
				SessionsApproved:   172,
			},
		},
		Total: 1,
	}

	h.repo.listByTeacherFn = func(ctx context.Context, tenantID, schoolID, userID, termID string) (*DeliverySummaryListResponse, error) {
		if userID != "teacher_001" {
			t.Errorf("expected teacher_001, got %q", userID)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return expected, nil
	}

	result, err := h.svc.ListByTeacher(context.Background(), "tenant_001", "school_001", "teacher_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 summary, got %d", result.Total)
	}
	if result.Items[0].TotalAssignedSlots != 180 {
		t.Fatalf("expected 180 assigned slots, got %d", result.Items[0].TotalAssignedSlots)
	}
	if result.Items[0].MarkedSlots != 172 {
		t.Fatalf("expected 172 marked slots, got %d", result.Items[0].MarkedSlots)
	}
}

func TestListByTeacher_EmptyInputs(t *testing.T) {
	h := newTestHarness()

	tests := []struct {
		name     string
		tenantID string
		schoolID string
		userID   string
		termID   string
	}{
		{"empty tenant_id", "", "school_001", "teacher_001", "term_001"},
		{"empty school_id", "tenant_001", "", "teacher_001", "term_001"},
		{"empty user_id", "tenant_001", "school_001", "", "term_001"},
		{"empty term_id", "tenant_001", "school_001", "teacher_001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.ListByTeacher(context.Background(), tt.tenantID, tt.schoolID, tt.userID, tt.termID)
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
// Tests: ListByTerm
// ============================================================================

func TestListByTerm_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := &DeliverySummaryListResponse{
		Items: []TeacherDeliverySummary{
			{UserID: "teacher_001", AcademicTermID: "term_001", TotalAssignedSlots: 180, MarkedSlots: 172},
			{UserID: "teacher_002", AcademicTermID: "term_001", TotalAssignedSlots: 150, MarkedSlots: 145},
		},
		Total: 2,
	}

	h.repo.listByTermFn = func(ctx context.Context, tenantID, schoolID, termID string) (*DeliverySummaryListResponse, error) {
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return expected, nil
	}

	result, err := h.svc.ListByTerm(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 summaries, got %d", result.Total)
	}
}

func TestListByTerm_EmptyInputs(t *testing.T) {
	h := newTestHarness()

	tests := []struct {
		name     string
		tenantID string
		schoolID string
		termID   string
	}{
		{"empty tenant_id", "", "school_001", "term_001"},
		{"empty school_id", "tenant_001", "", "term_001"},
		{"empty term_id", "tenant_001", "school_001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.ListByTerm(context.Background(), tt.tenantID, tt.schoolID, tt.termID)
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
// Tests: GetByTeacherTerm
// ============================================================================

func TestGetByTeacherTerm_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getByTeacherTermFn = func(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error) {
		if userID != "teacher_001" {
			t.Errorf("expected teacher_001, got %q", userID)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return &TeacherDeliverySummary{UserID: userID, AcademicTermID: termID, TotalAssignedSlots: 180}, nil
	}

	summary, err := h.svc.GetByTeacherTerm(context.Background(), "teacher_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalAssignedSlots != 180 {
		t.Fatalf("expected 180 assigned slots, got %d", summary.TotalAssignedSlots)
	}
}

func TestGetByTeacherTerm_EmptyInputs(t *testing.T) {
	h := newTestHarness()

	tests := []struct {
		name   string
		userID string
		termID string
	}{
		{"empty user_id", "", "term_001"},
		{"empty term_id", "teacher_001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.GetByTeacherTerm(context.Background(), tt.userID, tt.termID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetByTeacherTerm_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getByTeacherTermFn = func(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetByTeacherTerm(context.Background(), "teacher_999", "term_999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

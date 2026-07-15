package teachers

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listBySchoolFn    func(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error)
	getByIDFn         func(ctx context.Context, userID, tenantID, schoolID string) (*Teacher, error)
	updateFn          func(ctx context.Context, userID, tenantID, schoolID string, payload UpdateTeacherPayload) error
	toggleActiveFn    func(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error
	deleteFn          func(ctx context.Context, tenantID, schoolID, userID string) error
	getActiveSchoolID func(ctx context.Context, tenantID, userID string) (string, error)
}

func (m *MockRepository) ListBySchool(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
	if m.listBySchoolFn != nil {
		return m.listBySchoolFn(ctx, tenantID, schoolID, includeInactive, offset, limit, search)
	}
	return []Teacher{}, 0, nil
}

func (m *MockRepository) ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
	if m.toggleActiveFn != nil {
		return m.toggleActiveFn(ctx, tenantID, schoolID, userID, isActive)
	}
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, userID, tenantID, schoolID string) (*Teacher, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, userID, tenantID, schoolID)
	}
	return &Teacher{ID: userID, FullName: "Test Teacher", IsActive: true}, nil
}

func (m *MockRepository) Update(ctx context.Context, userID, tenantID, schoolID string, payload UpdateTeacherPayload) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, userID, tenantID, schoolID, payload)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, schoolID, userID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, schoolID, userID)
	}
	return nil
}

func (m *MockRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if m.getActiveSchoolID != nil {
		return m.getActiveSchoolID(ctx, tenantID, userID)
	}
	return "", nil
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
// Tests: ListTeachers
// ============================================================================

func TestListTeachers_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedTeachers := []Teacher{
		{ID: "user_001", Email: "john@school.com", FullName: "John Doe", IsActive: true},
		{ID: "user_002", Email: "jane@school.com", FullName: "Jane Smith", IsActive: true},
	}

	h.repo.listBySchoolFn = func(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if includeInactive {
			t.Error("expected includeInactive false")
		}
		if offset != 0 {
			t.Errorf("expected offset 0, got %d", offset)
		}
		if limit != 50 {
			t.Errorf("expected limit 50, got %d", limit)
		}
		return expectedTeachers, 2, nil
	}

	teachers, total, err := h.svc.ListTeachers(context.Background(), "tenant_001", "school_001", false, 0, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(teachers) != 2 {
		t.Fatalf("expected 2 teachers, got %d", len(teachers))
	}
	if teachers[0].FullName != "John Doe" {
		t.Fatalf("expected first teacher 'John Doe', got %q", teachers[0].FullName)
	}
}

func TestListTeachers_DefaultPagination(t *testing.T) {
	h := newTestHarness()

	h.repo.listBySchoolFn = func(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
		if limit != 50 {
			t.Errorf("expected default limit 50, got %d", limit)
		}
		if offset != 0 {
			t.Errorf("expected offset 0, got %d", offset)
		}
		return []Teacher{}, 0, nil
	}

	// Passing 0 for both should apply defaults
	_, total, err := h.svc.ListTeachers(context.Background(), "tenant_001", "school_001", false, 0, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
}

func TestListTeachers_LimitCappedAt100(t *testing.T) {
	h := newTestHarness()

	h.repo.listBySchoolFn = func(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
		if limit > 100 {
			t.Errorf("expected limit capped at 100, got %d", limit)
		}
		return []Teacher{}, 0, nil
	}

	_, _, err := h.svc.ListTeachers(context.Background(), "tenant_001", "school_001", false, 0, 200, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTeachers_WithSearch(t *testing.T) {
	h := newTestHarness()

	h.repo.listBySchoolFn = func(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
		if search != "john" {
			t.Errorf("expected search 'john', got %q", search)
		}
		return []Teacher{
			{ID: "user_001", FullName: "John Doe"},
		}, 1, nil
	}

	teachers, total, err := h.svc.ListTeachers(context.Background(), "tenant_001", "school_001", false, 0, 50, "john")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(teachers) != 1 {
		t.Fatalf("expected 1 teacher, got %d", len(teachers))
	}
}

func TestListTeachers_IncludeInactive(t *testing.T) {
	h := newTestHarness()

	h.repo.listBySchoolFn = func(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
		if !includeInactive {
			t.Error("expected includeInactive true")
		}
		return []Teacher{
			{ID: "user_001", IsActive: false},
			{ID: "user_002", IsActive: true},
		}, 2, nil
	}

	teachers, total, err := h.svc.ListTeachers(context.Background(), "tenant_001", "school_001", true, 0, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(teachers) != 2 {
		t.Fatalf("expected 2 teachers, got %d", len(teachers))
	}
}

// ============================================================================
// Tests: ToggleActive
// ============================================================================

func TestToggleActive_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.toggleActiveFn = func(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		if isActive {
			t.Error("expected isActive false")
		}
		return nil
	}

	err := h.svc.ToggleActive(context.Background(), "tenant_001", "school_001", "user_001", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToggleActive_EmptyUserID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.ToggleActive(context.Background(), "tenant_001", "school_001", "", true)
	if err == nil {
		t.Fatal("expected error for empty userID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestToggleActive_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.toggleActiveFn = func(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
		return ErrNotFound
	}

	err := h.svc.ToggleActive(context.Background(), "tenant_001", "school_001", "user_999", true)
	if err == nil {
		t.Fatal("expected error for non-existent teacher, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestToggleActive_Reactivate(t *testing.T) {
	h := newTestHarness()

	h.repo.toggleActiveFn = func(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
		if !isActive {
			t.Error("expected isActive true for reactivation")
		}
		return nil
	}

	err := h.svc.ToggleActive(context.Background(), "tenant_001", "school_001", "user_001", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests: Delete
// ============================================================================

func TestDelete_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, tenantID, schoolID, userID string) error {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		return nil
	}

	err := h.svc.Delete(context.Background(), "tenant_001", "school_001", "user_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_EmptyUserID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.Delete(context.Background(), "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error for empty userID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, tenantID, schoolID, userID string) error {
		return ErrNotFound
	}

	err := h.svc.Delete(context.Background(), "tenant_001", "school_001", "user_999")
	if err == nil {
		t.Fatal("expected error for non-existent teacher, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

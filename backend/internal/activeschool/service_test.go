package activeschool

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	upsertFn            func(ctx context.Context, tenantID, userID, schoolID string) error
	getActiveSchoolIDFn func(ctx context.Context, tenantID, userID string) (string, error)
}

func (m *MockRepository) Upsert(ctx context.Context, tenantID, userID, schoolID string) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, tenantID, userID, schoolID)
	}
	return nil
}

func (m *MockRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if m.getActiveSchoolIDFn != nil {
		return m.getActiveSchoolIDFn(ctx, tenantID, userID)
	}
	return "school_001", nil
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
// Tests: SwitchActiveSchool
// ============================================================================

func TestSwitchActiveSchool_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.upsertFn = func(ctx context.Context, tenantID, userID, schoolID string) error {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		if schoolID != "school_002" {
			t.Errorf("expected schoolID 'school_002', got %q", schoolID)
		}
		return nil
	}

	err := h.svc.SwitchActiveSchool(context.Background(), "tenant_001", "user_001", "school_002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSwitchActiveSchool_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.SwitchActiveSchool(context.Background(), "", "user_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSwitchActiveSchool_EmptyUserID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.SwitchActiveSchool(context.Background(), "tenant_001", "", "school_001")
	if err == nil {
		t.Fatal("expected error for empty userID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSwitchActiveSchool_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.SwitchActiveSchool(context.Background(), "tenant_001", "user_001", "")
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: GetActiveSchoolID
// ============================================================================

func TestGetActiveSchoolID_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getActiveSchoolIDFn = func(ctx context.Context, tenantID, userID string) (string, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		return "school_001", nil
	}

	schoolID, err := h.svc.GetActiveSchoolID(context.Background(), "tenant_001", "user_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schoolID != "school_001" {
		t.Fatalf("expected schoolID 'school_001', got %q", schoolID)
	}
}

func TestGetActiveSchoolID_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetActiveSchoolID(context.Background(), "", "user_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetActiveSchoolID_EmptyUserID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetActiveSchoolID(context.Background(), "tenant_001", "")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetActiveSchoolID_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getActiveSchoolIDFn = func(ctx context.Context, tenantID, userID string) (string, error) {
		return "", ErrNotFound
	}

	_, err := h.svc.GetActiveSchoolID(context.Background(), "tenant_001", "user_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

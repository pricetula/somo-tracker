package cbcstreams

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listFn                  func(ctx context.Context, tenantID, schoolID string) ([]Stream, error)
	getByIDFn               func(ctx context.Context, id, tenantID, schoolID string) (*Stream, error)
	createFn                func(ctx context.Context, tenantID, schoolID, name string) (*Stream, error)
	updateFn                func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error)
	deleteFn                func(ctx context.Context, id, tenantID, schoolID string) error
	hasReferencingClassesFn func(ctx context.Context, id, tenantID, schoolID string) (bool, error)
}

func (m *MockRepository) List(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, schoolID)
	}
	return []Stream{}, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID, schoolID)
	}
	return &Stream{ID: id, Name: "Test Stream"}, nil
}

func (m *MockRepository) Create(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, schoolID, name)
	}
	return &Stream{ID: "stream_001", Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (m *MockRepository) Update(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, tenantID, schoolID, name)
	}
	return &Stream{ID: id, Name: name}, nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) HasReferencingClasses(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	if m.hasReferencingClassesFn != nil {
		return m.hasReferencingClassesFn(ctx, id, tenantID, schoolID)
	}
	return false, nil
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
// Tests: ListStreams
// ============================================================================

func TestListStreams_HappyPath(t *testing.T) {
	h := newTestHarness()

	now := time.Now()
	expected := []Stream{
		{ID: "stream_001", Name: "Blue", CreatedAt: now, UpdatedAt: now},
		{ID: "stream_002", Name: "Red", CreatedAt: now, UpdatedAt: now},
	}

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return expected, nil
	}

	streams, err := h.svc.ListStreams(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(streams))
	}
	if streams[0].Name != "Blue" {
		t.Fatalf("expected name 'Blue', got %q", streams[0].Name)
	}
}

func TestListStreams_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListStreams(context.Background(), "", "school_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListStreams_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListStreams(context.Background(), "tenant_001", "")
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListStreams_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
		return []Stream{}, nil
	}

	streams, err := h.svc.ListStreams(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 0 {
		t.Fatalf("expected 0 streams, got %d", len(streams))
	}
}

// ============================================================================
// Tests: CreateStream
// ============================================================================

func TestCreateStream_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if name != "Blue" {
			t.Errorf("expected name 'Blue', got %q", name)
		}
		return &Stream{ID: "stream_001", Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}

	stream, err := h.svc.CreateStream(context.Background(), "tenant_001", "school_001", "Blue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.Name != "Blue" {
		t.Fatalf("expected name 'Blue', got %q", stream.Name)
	}
}

func TestCreateStream_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateStream(context.Background(), "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateStream_NameTooLong(t *testing.T) {
	h := newTestHarness()

	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	_, err := h.svc.CreateStream(context.Background(), "tenant_001", "school_001", longName)
	if err == nil {
		t.Fatal("expected error for name > 100 chars, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateStream_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateStream(context.Background(), "", "school_001", "Blue")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: UpdateStream
// ============================================================================

func TestUpdateStream_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
		if id != "stream_001" {
			t.Errorf("expected id 'stream_001', got %q", id)
		}
		if name != "Green" {
			t.Errorf("expected name 'Green', got %q", name)
		}
		return &Stream{ID: id, Name: name}, nil
	}

	stream, err := h.svc.UpdateStream(context.Background(), "stream_001", "tenant_001", "school_001", "Green")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.Name != "Green" {
		t.Fatalf("expected name 'Green', got %q", stream.Name)
	}
}

func TestUpdateStream_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateStream(context.Background(), "", "tenant_001", "school_001", "Green")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateStream_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateStream(context.Background(), "stream_001", "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteStream
// ============================================================================

func TestDeleteStream_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "stream_001" {
			t.Errorf("expected id 'stream_001', got %q", id)
		}
		return nil
	}

	err := h.svc.DeleteStream(context.Background(), "stream_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStream_BlockedByClasses(t *testing.T) {
	h := newTestHarness()

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return true, nil
	}

	err := h.svc.DeleteStream(context.Background(), "stream_001", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for stream in use, got nil")
	}
	if !errors.Is(err, ErrStreamInUse) {
		t.Fatalf("expected ErrStreamInUse, got %v", err)
	}
}

func TestDeleteStream_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteStream(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteStream_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	err := h.svc.DeleteStream(context.Background(), "stream_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

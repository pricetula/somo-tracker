package cbcstreams

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listFn                 func(ctx context.Context, tenantID, schoolID string) ([]Stream, error)
	createFn               func(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error)
	getByIDFn              func(ctx context.Context, id, tenantID, schoolID string) (*Stream, error)
	updateFn               func(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error)
	deleteFn               func(ctx context.Context, id, tenantID, schoolID string) error
	hasActiveEnrollmentsFn func(ctx context.Context, id, tenantID, schoolID string) (bool, error)
}

func (m *MockRepository) List(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, schoolID)
	}
	return []Stream{}, nil
}

func (m *MockRepository) Create(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, schoolID, name, color)
	}
	return &Stream{ID: "stream_001", Name: name, Color: color}, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID, schoolID)
	}
	return &Stream{ID: id, Name: "Blue", Color: "#0000FF"}, nil
}

func (m *MockRepository) Update(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, tenantID, schoolID, name, color)
	}
	return &Stream{ID: id, Name: name, Color: color}, nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) HasActiveEnrollments(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	if m.hasActiveEnrollmentsFn != nil {
		return m.hasActiveEnrollmentsFn(ctx, id, tenantID, schoolID)
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

	expectedStreams := []Stream{
		{ID: "stream_001", Name: "Blue", Color: "#0000FF"},
		{ID: "stream_002", Name: "Red", Color: "#FF0000"},
	}

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return expectedStreams, nil
	}

	result, err := h.svc.ListStreams(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Blue" {
		t.Fatalf("expected first stream 'Blue', got %q", result.Items[0].Name)
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

	result, err := h.svc.ListStreams(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
}

// ============================================================================
// Tests: CreateStream
// ============================================================================

func TestCreateStream_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if name != "Blue" {
			t.Errorf("expected name 'Blue', got %q", name)
		}
		if color != "#0000FF" {
			t.Errorf("expected color '#0000FF', got %q", color)
		}
		return &Stream{ID: "stream_001", Name: "Blue", Color: "#0000FF"}, nil
	}

	s, err := h.svc.CreateStream(context.Background(), "tenant_001", "school_001", "Blue", "#0000FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != "stream_001" {
		t.Fatalf("expected id 'stream_001', got %q", s.ID)
	}
	if s.Name != "Blue" {
		t.Fatalf("expected name 'Blue', got %q", s.Name)
	}
}

func TestCreateStream_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateStream(context.Background(), "tenant_001", "school_001", "", "#0000FF")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateStream_AlreadyExists(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error) {
		return nil, ErrAlreadyExists
	}

	_, err := h.svc.CreateStream(context.Background(), "tenant_001", "school_001", "Blue", "#0000FF")
	if err == nil {
		t.Fatal("expected error for duplicate stream, got nil")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

// ============================================================================
// Tests: UpdateStream
// ============================================================================

func TestUpdateStream_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error) {
		if id != "stream_001" {
			t.Errorf("expected id 'stream_001', got %q", id)
		}
		if name != "Green" {
			t.Errorf("expected name 'Green', got %q", name)
		}
		if color != "#00FF00" {
			t.Errorf("expected color '#00FF00', got %q", color)
		}
		return &Stream{ID: id, Name: name, Color: color}, nil
	}

	s, err := h.svc.UpdateStream(context.Background(), "stream_001", "tenant_001", "school_001", "Green", "#00FF00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "Green" {
		t.Fatalf("expected name 'Green', got %q", s.Name)
	}
	if s.Color != "#00FF00" {
		t.Fatalf("expected color '#00FF00', got %q", s.Color)
	}
}

func TestUpdateStream_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateStream(context.Background(), "", "tenant_001", "school_001", "Green", "#00FF00")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateStream_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.UpdateStream(context.Background(), "stream_999", "tenant_001", "school_001", "Green", "#00FF00")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteStream
// ============================================================================

func TestDeleteStream_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
		return &Stream{ID: id, Name: "Blue", Color: "#0000FF"}, nil
	}

	called := false
	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		called = true
		if id != "stream_001" {
			t.Errorf("expected id 'stream_001', got %q", id)
		}
		return nil
	}

	err := h.svc.DeleteStream(context.Background(), "stream_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected deleteFn to be called")
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

func TestDeleteStream_HasActiveEnrollments(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
		return &Stream{ID: id, Name: "Blue", Color: "#0000FF"}, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrStreamHasActiveEnrollments
	}

	err := h.svc.DeleteStream(context.Background(), "stream_001", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for active enrollments, got nil")
	}
	if !errors.Is(err, ErrStreamHasActiveEnrollments) {
		t.Fatalf("expected ErrStreamHasActiveEnrollments, got %v", err)
	}
	// Verify the error message includes the stream name
	if !contains(err.Error(), "Blue") {
		t.Fatalf("expected error to contain stream name 'Blue', got: %v", err)
	}
}

func TestDeleteStream_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
		return nil, ErrNotFound
	}

	err := h.svc.DeleteStream(context.Background(), "stream_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for non-existent stream, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// contains checks if substr is in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

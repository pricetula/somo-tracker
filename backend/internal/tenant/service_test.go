package tenant

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockSqlcRepository
// ============================================================================

type MockSqlcRepository struct {
	existsByNameFn func(ctx context.Context, name string) (bool, error)
	existsBySlugFn func(ctx context.Context, slug string) (bool, error)
	createFn       func(ctx context.Context, name, slug string) (*Tenant, error)
	getByIDFn      func(ctx context.Context, id string) (*Tenant, error)
}

func (m *MockSqlcRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(ctx, name)
	}
	return false, nil
}

func (m *MockSqlcRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.existsBySlugFn != nil {
		return m.existsBySlugFn(ctx, slug)
	}
	return false, nil
}

func (m *MockSqlcRepository) Create(ctx context.Context, name, slug string) (*Tenant, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name, slug)
	}
	return &Tenant{ID: "tenant_001", Name: name, Slug: slug}, nil
}

func (m *MockSqlcRepository) GetByID(ctx context.Context, id string) (*Tenant, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc  *Service
	repo *MockSqlcRepository
}

func newTestHarness() *testHarness {
	repo := &MockSqlcRepository{}
	svc := &Service{repo: repo}
	return &testHarness{svc: svc, repo: repo}
}

// ============================================================================
// Tests: CreateTenant — Happy Path
// ============================================================================

func TestCreateTenant_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, name, slug string) (*Tenant, error) {
		return &Tenant{ID: "tenant_abc123", Name: name, Slug: slug}, nil
	}

	tenant, err := h.svc.CreateTenant(context.Background(), "Test School", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant.ID != "tenant_abc123" {
		t.Fatalf("expected ID 'tenant_abc123', got %q", tenant.ID)
	}
	if tenant.Slug == "" {
		t.Fatal("expected non-empty slug")
	}
}

func TestCreateTenant_WithCustomSlug(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, name, slug string) (*Tenant, error) {
		if slug != "custom-slug" {
			t.Fatalf("expected slug 'custom-slug', got %q", slug)
		}
		return &Tenant{ID: "tenant_001", Name: name, Slug: slug}, nil
	}

	tenant, err := h.svc.CreateTenant(context.Background(), "Test School", "custom-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant.Slug != "custom-slug" {
		t.Fatalf("expected slug 'custom-slug', got %q", tenant.Slug)
	}
}

// ============================================================================
// Tests: CreateTenant — Bad Paths
// ============================================================================

func TestCreateTenant_NameAlreadyExists(t *testing.T) {
	h := newTestHarness()

	h.repo.existsByNameFn = func(ctx context.Context, name string) (bool, error) {
		return true, nil
	}

	_, err := h.svc.CreateTenant(context.Background(), "Duplicate School", "")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestCreateTenant_ExistsByNameError(t *testing.T) {
	h := newTestHarness()

	h.repo.existsByNameFn = func(ctx context.Context, name string) (bool, error) {
		return false, errors.New("db connection error")
	}

	_, err := h.svc.CreateTenant(context.Background(), "Error School", "")
	if err == nil {
		t.Fatal("expected error for DB failure, got nil")
	}
}

func TestCreateTenant_SlugConflictHandling(t *testing.T) {
	h := newTestHarness()

	slugCheckCount := 0
	h.repo.existsBySlugFn = func(ctx context.Context, slug string) (bool, error) {
		slugCheckCount++
		// First two slugs exist (original + -2), third (-3) is free
		if slugCheckCount <= 2 {
			return true, nil
		}
		return false, nil
	}

	h.repo.createFn = func(ctx context.Context, name, slug string) (*Tenant, error) {
		if slug != "test-school-3" {
			t.Fatalf("expected slug 'test-school-3' after conflicts, got %q", slug)
		}
		return &Tenant{ID: "tenant_001", Name: name, Slug: slug}, nil
	}

	tenant, err := h.svc.CreateTenant(context.Background(), "Test School", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant.Slug != "test-school-3" {
		t.Fatalf("expected slug 'test-school-3', got %q", tenant.Slug)
	}
}

func TestCreateTenant_SlugCheckError(t *testing.T) {
	h := newTestHarness()

	h.repo.existsBySlugFn = func(ctx context.Context, slug string) (bool, error) {
		return false, errors.New("db error checking slug")
	}

	_, err := h.svc.CreateTenant(context.Background(), "Slug Error School", "")
	if err == nil {
		t.Fatal("expected error for slug check failure, got nil")
	}
}

func TestCreateTenant_CreateFails(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, name, slug string) (*Tenant, error) {
		return nil, errors.New("postgres insert error")
	}

	_, err := h.svc.CreateTenant(context.Background(), "Fail School", "")
	if err == nil {
		t.Fatal("expected error for insert failure, got nil")
	}
}

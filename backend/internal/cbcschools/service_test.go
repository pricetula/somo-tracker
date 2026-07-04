package cbcschools

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
	createFn       func(ctx context.Context, tenantID string, name string) (string, error)
	getByIDFn      func(ctx context.Context, id string) (*School, error)
	listByTenantFn func(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error)
	updateFn       func(ctx context.Context, school SchoolUpdateFields) error
	deleteFn       func(ctx context.Context, id string) error
}

func (m *MockRepository) Create(ctx context.Context, tenantID string, name string) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, name)
	}
	return "school_001", nil
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*School, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &School{ID: id, TenantID: "tenant_001", Name: "Test School"}, nil
}

func (m *MockRepository) ListByTenantID(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
	if m.listByTenantFn != nil {
		return m.listByTenantFn(ctx, tenantID, userID)
	}
	return []SchoolWithMemberCount{}, nil
}

func (m *MockRepository) Update(ctx context.Context, school SchoolUpdateFields) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, school)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
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

func ptr(s string) *string { return &s }

// ============================================================================
// Tests: CreateSchool
// ============================================================================

func TestCreateSchool_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, name string) (string, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if name != "Green Valley Primary" {
			t.Errorf("expected name 'Green Valley Primary', got %q", name)
		}
		return "school_001", nil
	}

	id, err := h.svc.CreateSchool(context.Background(), "tenant_001", "Green Valley Primary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "school_001" {
		t.Fatalf("expected id 'school_001', got %q", id)
	}
}

func TestCreateSchool_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSchool(context.Background(), "tenant_001", "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ListSchoolsByTenantID
// ============================================================================

func TestListSchools_HappyPath(t *testing.T) {
	h := newTestHarness()

	now := time.Now()
	expectedSchools := []SchoolWithMemberCount{
		{
			ID: "school_001", TenantID: "tenant_001", Name: "Green Valley",
			County: "Nairobi", SubCounty: "Westlands", SchoolType: "Public",
			IsActive: true, CreatedAt: now, UpdatedAt: now,
			Teachers: 15,
		},
		{
			ID: "school_002", TenantID: "tenant_001", Name: "Riverside Academy",
			County: "Nairobi", SubCounty: "Kilimani", SchoolType: "Private",
			IsActive: true, CreatedAt: now, UpdatedAt: now,
			Teachers: 40, Parents: 2,
		},
	}

	h.repo.listByTenantFn = func(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if userID != "user_001" {
			t.Errorf("expected userID 'user_001', got %q", userID)
		}
		return expectedSchools, nil
	}

	schools, err := h.svc.ListSchoolsByTenantID(context.Background(), "tenant_001", "user_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schools) != 2 {
		t.Fatalf("expected 2 schools, got %d", len(schools))
	}
	if schools[0].Teachers != 15 {
		t.Fatalf("expected Teachers 15, got %d", schools[0].Teachers)
	}
	if schools[1].Teachers != 40 {
		t.Fatalf("expected Teachers 40, got %d", schools[1].Teachers)
	}
	if schools[1].Parents != 2 {
		t.Fatalf("expected Parents 2, got %d", schools[1].Parents)
	}
}

func TestListSchools_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListSchoolsByTenantID(context.Background(), "", "user_001")
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListSchools_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listByTenantFn = func(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
		return []SchoolWithMemberCount{}, nil
	}

	schools, err := h.svc.ListSchoolsByTenantID(context.Background(), "tenant_001", "user_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schools) != 0 {
		t.Fatalf("expected 0 schools, got %d", len(schools))
	}
}

// ============================================================================
// Tests: UpdateSchool
// ============================================================================

func TestUpdateSchool_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFn = func(ctx context.Context, school SchoolUpdateFields) error {
		if school.ID != "school_001" {
			t.Errorf("expected ID 'school_001', got %q", school.ID)
		}
		if school.Name == nil || *school.Name != "New Name" {
			t.Errorf("expected Name 'New Name', got %v", school.Name)
		}
		if school.County == nil || *school.County != "Mombasa" {
			t.Errorf("expected County 'Mombasa', got %v", school.County)
		}
		return nil
	}

	err := h.svc.UpdateSchool(context.Background(), SchoolUpdateFields{
		ID:     "school_001",
		Name:   ptr("New Name"),
		County: ptr("Mombasa"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSchool_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateSchool(context.Background(), SchoolUpdateFields{
		ID:   "",
		Name: ptr("New Name"),
	})
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateSchool_NoFields(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateSchool(context.Background(), SchoolUpdateFields{
		ID: "school_001",
	})
	if err == nil {
		t.Fatal("expected error for no fields, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateSchool_IsActiveOnly(t *testing.T) {
	h := newTestHarness()

	active := false
	h.repo.updateFn = func(ctx context.Context, school SchoolUpdateFields) error {
		if school.IsActive == nil || *school.IsActive != false {
			t.Errorf("expected IsActive false, got %v", school.IsActive)
		}
		return nil
	}

	err := h.svc.UpdateSchool(context.Background(), SchoolUpdateFields{
		ID:       "school_001",
		IsActive: &active,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests: DeleteSchool
// ============================================================================

func TestDeleteSchool_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, id string) error {
		if id != "school_001" {
			t.Errorf("expected id 'school_001', got %q", id)
		}
		return nil
	}

	err := h.svc.DeleteSchool(context.Background(), "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSchool_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteSchool(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// MockCurriculumSeeder
// ============================================================================

// MockCurriculumSeeder records calls to SeedForSchool so tests can verify
// that the curriculum was seeded after school creation.
type MockCurriculumSeeder struct {
	calls   []SeedCall
	seedErr error // if set, SeedForSchool returns this error
}

type SeedCall struct {
	Ctx      context.Context
	TenantID string
	SchoolID string
}

func (m *MockCurriculumSeeder) SeedForSchool(ctx context.Context, tenantID, schoolID string) error {
	m.calls = append(m.calls, SeedCall{
		Ctx:      ctx,
		TenantID: tenantID,
		SchoolID: schoolID,
	})
	return m.seedErr
}

// ============================================================================
// Test Harness with Seeder
// ============================================================================

type testHarnessWithSeeder struct {
	svc    *Service
	repo   *MockRepository
	seeder *MockCurriculumSeeder
}

func newTestHarnessWithSeeder() *testHarnessWithSeeder {
	repo := &MockRepository{}
	seeder := &MockCurriculumSeeder{}
	svc := NewService(repo, WithCurriculumSeeder(seeder))
	return &testHarnessWithSeeder{
		svc:    svc,
		repo:   repo,
		seeder: seeder,
	}
}

// ============================================================================
// Tests: CreateSchool — Curriculum Seeding
// ============================================================================

func TestCreateSchool_SeedsCurriculum(t *testing.T) {
	h := newTestHarnessWithSeeder()

	const expectedTenantID = "tenant_abc"
	const expectedSchoolID = "school_xyz"

	h.repo.createFn = func(ctx context.Context, tenantID, name string) (string, error) {
		return expectedSchoolID, nil
	}

	id, err := h.svc.CreateSchool(context.Background(), expectedTenantID, "Test School")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != expectedSchoolID {
		t.Fatalf("expected school ID %q, got %q", expectedSchoolID, id)
	}

	// Verify seed was called exactly once
	if len(h.seeder.calls) != 1 {
		t.Fatalf("expected 1 call to SeedForSchool, got %d", len(h.seeder.calls))
	}

	// Verify the correct tenant ID and school ID were passed to the seeder
	call := h.seeder.calls[0]
	if call.TenantID != expectedTenantID {
		t.Errorf("expected TenantID %q, got %q", expectedTenantID, call.TenantID)
	}
	if call.SchoolID != expectedSchoolID {
		t.Errorf("expected SchoolID %q, got %q", expectedSchoolID, call.SchoolID)
	}
}

func TestCreateSchool_SeedFailure_DoesNotFailSchoolCreation(t *testing.T) {
	h := newTestHarnessWithSeeder()

	h.repo.createFn = func(ctx context.Context, tenantID, name string) (string, error) {
		return "school_001", nil
	}

	// Make the seeder return an error (simulates a transient DB issue)
	h.seeder.seedErr = errors.New("database timeout")

	// School creation should still succeed — seeding failure is logged, not fatal
	id, err := h.svc.CreateSchool(context.Background(), "tenant_001", "Test School")
	if err != nil {
		t.Fatalf("school creation should succeed even when seeding fails, got error: %v", err)
	}
	if id != "school_001" {
		t.Fatalf("expected school ID 'school_001', got %q", id)
	}

	// Verify the seeder was called (the error confirm the call was made)
	if len(h.seeder.calls) != 1 {
		t.Fatalf("expected 1 call to SeedForSchool, got %d", len(h.seeder.calls))
	}
}

func TestCreateSchool_WithoutSeeder_DoesNotSeed(t *testing.T) {
	// Use the standard harness which has NO seeder configured
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, name string) (string, error) {
		return "school_001", nil
	}

	id, err := h.svc.CreateSchool(context.Background(), "tenant_001", "Test School")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "school_001" {
		t.Fatalf("expected school ID 'school_001', got %q", id)
	}

	// The seeder field is nil, so CreateSchool should never call SeedForSchool.
	// There's no way to inspect this directly, but we verify the school was
	// created successfully, which is the expected behavior when no seeder is
	// wired.
}

package curriculum

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	createFn  func(ctx context.Context, params CreateLearningAreaParams) (string, error)
	getByIDFn func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error)
	listFn    func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error)
	updateFn  func(ctx context.Context, params UpdateLearningAreaParams) error
	deleteFn  func(ctx context.Context, id, tenantID, schoolID string) error
}

func (m *MockRepository) Create(ctx context.Context, params CreateLearningAreaParams) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return "area_001", nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID, schoolID)
	}
	return &LearningArea{
		ID:             id,
		TenantID:       tenantID,
		SchoolID:       schoolID,
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	}, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, schoolID, educationLevel)
	}
	return []LearningArea{}, nil
}

func (m *MockRepository) Update(ctx context.Context, params UpdateLearningAreaParams) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, params)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, tenantID, schoolID)
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

func ptrStr(s string) *string { return &s }

// ============================================================================
// Tests: CreateLearningArea
// ============================================================================

func TestCreateLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, params CreateLearningAreaParams) (string, error) {
		if params.TenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", params.TenantID)
		}
		if params.SchoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", params.SchoolID)
		}
		if params.Name != "Mathematics" {
			t.Errorf("expected Name 'Mathematics', got %q", params.Name)
		}
		if params.Code != "MATH" {
			t.Errorf("expected Code 'MATH', got %q", params.Code)
		}
		if params.EducationLevel != "Junior_Secondary" {
			t.Errorf("expected EducationLevel 'Junior_Secondary', got %q", params.EducationLevel)
		}
		return "area_001", nil
	}

	id, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "area_001" {
		t.Fatalf("expected id 'area_001', got %q", id)
	}
}

func TestCreateLearningArea_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           "",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_EmptyCode(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           "Mathematics",
		Code:           "",
		EducationLevel: "Junior_Secondary",
	})
	if err == nil {
		t.Fatal("expected error for empty code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_InvalidCodeFormat(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           "Mathematics",
		Code:           "math", // lowercase
		EducationLevel: "Junior_Secondary",
	})
	if err == nil {
		t.Fatal("expected error for lowercase code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_InvalidEducationLevel(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Invalid_Level",
	})
	if err == nil {
		t.Fatal("expected error for invalid education_level, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "",
		SchoolID:       "school_001",
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	})
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "",
		Name:           "Mathematics",
		Code:           "MATH",
		EducationLevel: "Junior_Secondary",
	})
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_CodeTooLong(t *testing.T) {
	h := newTestHarness()

	longCode := ""
	for i := 0; i < 51; i++ {
		longCode += "A"
	}

	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           "Test",
		Code:           longCode,
		EducationLevel: "Early_Years",
	})
	if err == nil {
		t.Fatal("expected error for too-long code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: GetLearningArea
// ============================================================================

func TestGetLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		if id != "area_001" {
			t.Errorf("expected id 'area_001', got %q", id)
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return &LearningArea{
			ID:             id,
			TenantID:       tenantID,
			SchoolID:       schoolID,
			Name:           "Mathematics",
			Code:           "MATH",
			EducationLevel: "Junior_Secondary",
		}, nil
	}

	area, err := h.svc.GetLearningArea(context.Background(), "area_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if area.Name != "Mathematics" {
		t.Fatalf("expected Name 'Mathematics', got %q", area.Name)
	}
	if area.Code != "MATH" {
		t.Fatalf("expected Code 'MATH', got %q", area.Code)
	}
}

func TestGetLearningArea_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetLearningArea(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ListLearningAreas
// ============================================================================

func TestListLearningAreas_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedAreas := []LearningArea{
		{ID: "area_001", TenantID: "tenant_001", SchoolID: "school_001", Name: "English", Code: "ENG", EducationLevel: "Early_Years"},
		{ID: "area_002", TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Early_Years"},
	}

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if educationLevel != nil {
			t.Errorf("expected nil educationLevel filter, got %q", *educationLevel)
		}
		return expectedAreas, nil
	}

	areas, err := h.svc.ListLearningAreas(context.Background(), "tenant_001", "school_001", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(areas) != 2 {
		t.Fatalf("expected 2 learning areas, got %d", len(areas))
	}
}

func TestListLearningAreas_FilteredByLevel(t *testing.T) {
	h := newTestHarness()

	level := "Senior_School"
	expectedAreas := []LearningArea{
		{ID: "area_003", Name: "Biology", Code: "BIO", EducationLevel: "Senior_School"},
	}

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		if educationLevel == nil || *educationLevel != "Senior_School" {
			t.Errorf("expected educationLevel 'Senior_School', got %v", educationLevel)
		}
		return expectedAreas, nil
	}

	areas, err := h.svc.ListLearningAreas(context.Background(), "tenant_001", "school_001", &level)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(areas) != 1 {
		t.Fatalf("expected 1 learning area, got %d", len(areas))
	}
	if areas[0].Code != "BIO" {
		t.Fatalf("expected Code 'BIO', got %q", areas[0].Code)
	}
}

func TestListLearningAreas_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListLearningAreas(context.Background(), "", "school_001", nil)
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListLearningAreas_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		return []LearningArea{}, nil
	}

	areas, err := h.svc.ListLearningAreas(context.Background(), "tenant_001", "school_001", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(areas) != 0 {
		t.Fatalf("expected 0 learning areas, got %d", len(areas))
	}
}

// ============================================================================
// Tests: UpdateLearningArea
// ============================================================================

func TestUpdateLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	newName := "Advanced Mathematics"

	h.repo.updateFn = func(ctx context.Context, params UpdateLearningAreaParams) error {
		if params.ID != "area_001" {
			t.Errorf("expected ID 'area_001', got %q", params.ID)
		}
		if params.Name == nil || *params.Name != "Advanced Mathematics" {
			t.Errorf("expected Name 'Advanced Mathematics', got %v", params.Name)
		}
		return nil
	}

	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:       "area_001",
		TenantID: "tenant_001",
		SchoolID: "school_001",
		Name:     ptrStr(newName),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateLearningArea_AllFields(t *testing.T) {
	h := newTestHarness()

	newName := "Integrated Science"
	newCode := "INT_SCI"
	newLevel := "Upper_Primary"

	h.repo.updateFn = func(ctx context.Context, params UpdateLearningAreaParams) error {
		if params.Name == nil || *params.Name != "Integrated Science" {
			t.Errorf("expected Name 'Integrated Science', got %v", params.Name)
		}
		if params.Code == nil || *params.Code != "INT_SCI" {
			t.Errorf("expected Code 'INT_SCI', got %v", params.Code)
		}
		if params.EducationLevel == nil || *params.EducationLevel != "Upper_Primary" {
			t.Errorf("expected EducationLevel 'Upper_Primary', got %v", params.EducationLevel)
		}
		return nil
	}

	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:             "area_001",
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		Name:           &newName,
		Code:           &newCode,
		EducationLevel: &newLevel,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateLearningArea_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:       "",
		TenantID: "tenant_001",
		SchoolID: "school_001",
		Name:     ptrStr("New Name"),
	})
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateLearningArea_NoFields(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:       "area_001",
		TenantID: "tenant_001",
		SchoolID: "school_001",
	})
	if err == nil {
		t.Fatal("expected error for no fields, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateLearningArea_InvalidName(t *testing.T) {
	h := newTestHarness()

	emptyName := ""
	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:       "area_001",
		TenantID: "tenant_001",
		SchoolID: "school_001",
		Name:     &emptyName,
	})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateLearningArea_InvalidLevel(t *testing.T) {
	h := newTestHarness()

	invalidLevel := "Bad_Level"
	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:             "area_001",
		TenantID:       "tenant_001",
		SchoolID:       "school_001",
		EducationLevel: &invalidLevel,
	})
	if err == nil {
		t.Fatal("expected error for invalid education_level, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateLearningArea_InvalidCode(t *testing.T) {
	h := newTestHarness()

	badCode := "bad code!"
	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID:       "area_001",
		TenantID: "tenant_001",
		SchoolID: "school_001",
		Code:     &badCode,
	})
	if err == nil {
		t.Fatal("expected error for invalid code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteLearningArea
// ============================================================================

func TestDeleteLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "area_001" {
			t.Errorf("expected id 'area_001', got %q", id)
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return nil
	}

	err := h.svc.DeleteLearningArea(context.Background(), "area_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteLearningArea_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteLearningArea(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

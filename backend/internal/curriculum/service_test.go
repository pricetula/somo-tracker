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
	createLearningAreaFn  func(ctx context.Context, params CreateLearningAreaParams) (string, error)
	getLearningAreaByIDFn func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error)
	listLearningAreasFn   func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error)
	updateLearningAreaFn  func(ctx context.Context, params UpdateLearningAreaParams) error
	deleteLearningAreaFn  func(ctx context.Context, id, tenantID, schoolID string) error

	createStrandFn  func(ctx context.Context, params CreateStrandParams) (string, error)
	getStrandByIDFn func(ctx context.Context, id string) (*Strand, error)
	listStrandsFn   func(ctx context.Context, learningAreaID string) ([]Strand, error)
	updateStrandFn  func(ctx context.Context, params UpdateStrandParams) error
	deleteStrandFn  func(ctx context.Context, id string) error

	createSubStrandFn  func(ctx context.Context, params CreateSubStrandParams) (string, error)
	getSubStrandByIDFn func(ctx context.Context, id string) (*SubStrand, error)
	listSubStrandsFn   func(ctx context.Context, strandID string) ([]SubStrand, error)
	updateSubStrandFn  func(ctx context.Context, params UpdateSubStrandParams) error
	deleteSubStrandFn  func(ctx context.Context, id string) error

	createPIFn  func(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error)
	getPIBYIDFn func(ctx context.Context, id string) (*PerformanceIndicator, error)
	listPIFn    func(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error)
	updatePIFn  func(ctx context.Context, params UpdatePerformanceIndicatorParams) error
	deletePIFn  func(ctx context.Context, id string) error
	getMaxSeqFn func(ctx context.Context, subStrandID string) (int, error)

	getTreeFn func(ctx context.Context, learningAreaID string) (*LearningAreaTree, error)

	verifyLearningAreaBelongsToTenantFn func(ctx context.Context, id, tenantID, schoolID string) error
	verifyStrandInTenantSchoolFn        func(ctx context.Context, strandID, tenantID, schoolID string) (string, error)
	verifySubStrandInTenantSchoolFn     func(ctx context.Context, subStrandID, tenantID, schoolID string) (string, error)

	getPerformanceIndicatorEducationLevelFn func(ctx context.Context, indicatorID string) (string, error)
}

func (m *MockRepository) GetPerformanceIndicatorEducationLevel(ctx context.Context, indicatorID string) (string, error) {
	if m.getPerformanceIndicatorEducationLevelFn != nil {
		return m.getPerformanceIndicatorEducationLevelFn(ctx, indicatorID)
	}
	return "Junior_Secondary", nil
}

func (m *MockRepository) CreateLearningArea(ctx context.Context, params CreateLearningAreaParams) (string, error) {
	if m.createLearningAreaFn != nil {
		return m.createLearningAreaFn(ctx, params)
	}
	return "area_001", nil
}

func (m *MockRepository) GetLearningAreaByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
	if m.getLearningAreaByIDFn != nil {
		return m.getLearningAreaByIDFn(ctx, id, tenantID, schoolID)
	}
	return &LearningArea{ID: id, TenantID: tenantID, SchoolID: schoolID, Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"}, nil
}

func (m *MockRepository) ListLearningAreas(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
	if m.listLearningAreasFn != nil {
		return m.listLearningAreasFn(ctx, tenantID, schoolID, educationLevel)
	}
	return []LearningArea{}, nil
}

func (m *MockRepository) UpdateLearningArea(ctx context.Context, params UpdateLearningAreaParams) error {
	if m.updateLearningAreaFn != nil {
		return m.updateLearningAreaFn(ctx, params)
	}
	return nil
}

func (m *MockRepository) DeleteLearningArea(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteLearningAreaFn != nil {
		return m.deleteLearningAreaFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) CreateStrand(ctx context.Context, params CreateStrandParams) (string, error) {
	if m.createStrandFn != nil {
		return m.createStrandFn(ctx, params)
	}
	return "strand_001", nil
}

func (m *MockRepository) GetStrandByID(ctx context.Context, id string) (*Strand, error) {
	if m.getStrandByIDFn != nil {
		return m.getStrandByIDFn(ctx, id)
	}
	return &Strand{ID: id, LearningAreaID: "area_001", Name: "Numbers"}, nil
}

func (m *MockRepository) ListStrandsByLearningArea(ctx context.Context, learningAreaID string) ([]Strand, error) {
	if m.listStrandsFn != nil {
		return m.listStrandsFn(ctx, learningAreaID)
	}
	return []Strand{}, nil
}

func (m *MockRepository) UpdateStrand(ctx context.Context, params UpdateStrandParams) error {
	if m.updateStrandFn != nil {
		return m.updateStrandFn(ctx, params)
	}
	return nil
}

func (m *MockRepository) DeleteStrand(ctx context.Context, id string) error {
	if m.deleteStrandFn != nil {
		return m.deleteStrandFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) CreateSubStrand(ctx context.Context, params CreateSubStrandParams) (string, error) {
	if m.createSubStrandFn != nil {
		return m.createSubStrandFn(ctx, params)
	}
	return "sub_001", nil
}

func (m *MockRepository) GetSubStrandByID(ctx context.Context, id string) (*SubStrand, error) {
	if m.getSubStrandByIDFn != nil {
		return m.getSubStrandByIDFn(ctx, id)
	}
	return &SubStrand{ID: id, StrandID: "strand_001", Name: "Addition"}, nil
}

func (m *MockRepository) ListSubStrandsByStrand(ctx context.Context, strandID string) ([]SubStrand, error) {
	if m.listSubStrandsFn != nil {
		return m.listSubStrandsFn(ctx, strandID)
	}
	return []SubStrand{}, nil
}

func (m *MockRepository) UpdateSubStrand(ctx context.Context, params UpdateSubStrandParams) error {
	if m.updateSubStrandFn != nil {
		return m.updateSubStrandFn(ctx, params)
	}
	return nil
}

func (m *MockRepository) DeleteSubStrand(ctx context.Context, id string) error {
	if m.deleteSubStrandFn != nil {
		return m.deleteSubStrandFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) CreatePerformanceIndicator(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error) {
	if m.createPIFn != nil {
		return m.createPIFn(ctx, params)
	}
	return "pi_001", nil
}

func (m *MockRepository) GetPerformanceIndicatorByID(ctx context.Context, id string) (*PerformanceIndicator, error) {
	if m.getPIBYIDFn != nil {
		return m.getPIBYIDFn(ctx, id)
	}
	return &PerformanceIndicator{ID: id, SubStrandID: "sub_001", Description: "Solve 1+1", SequenceOrder: 1}, nil
}

func (m *MockRepository) ListPerformanceIndicatorsBySubStrand(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error) {
	if m.listPIFn != nil {
		return m.listPIFn(ctx, subStrandID)
	}
	return []PerformanceIndicator{}, nil
}

func (m *MockRepository) UpdatePerformanceIndicator(ctx context.Context, params UpdatePerformanceIndicatorParams) error {
	if m.updatePIFn != nil {
		return m.updatePIFn(ctx, params)
	}
	return nil
}

func (m *MockRepository) DeletePerformanceIndicator(ctx context.Context, id string) error {
	if m.deletePIFn != nil {
		return m.deletePIFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) GetMaxSequenceOrder(ctx context.Context, subStrandID string) (int, error) {
	if m.getMaxSeqFn != nil {
		return m.getMaxSeqFn(ctx, subStrandID)
	}
	return 0, nil
}

func (m *MockRepository) GetTree(ctx context.Context, learningAreaID string) (*LearningAreaTree, error) {
	if m.getTreeFn != nil {
		return m.getTreeFn(ctx, learningAreaID)
	}
	return &LearningAreaTree{}, nil
}

func (m *MockRepository) VerifyLearningAreaBelongsToTenant(ctx context.Context, id, tenantID, schoolID string) error {
	if m.verifyLearningAreaBelongsToTenantFn != nil {
		return m.verifyLearningAreaBelongsToTenantFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) VerifyStrandInTenantSchool(ctx context.Context, strandID, tenantID, schoolID string) (string, error) {
	if m.verifyStrandInTenantSchoolFn != nil {
		return m.verifyStrandInTenantSchoolFn(ctx, strandID, tenantID, schoolID)
	}
	return "area_001", nil
}

func (m *MockRepository) VerifySubStrandInTenantSchool(ctx context.Context, subStrandID, tenantID, schoolID string) (string, error) {
	if m.verifySubStrandInTenantSchoolFn != nil {
		return m.verifySubStrandInTenantSchoolFn(ctx, subStrandID, tenantID, schoolID)
	}
	return "strand_001", nil
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
func intPtr(i int) *int       { return &i }

// ============================================================================
// Tests: CreateLearningArea
// ============================================================================

func TestCreateLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createLearningAreaFn = func(ctx context.Context, params CreateLearningAreaParams) (string, error) {
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
		TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary",
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
	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{TenantID: "tenant_001", SchoolID: "school_001", Name: "", Code: "MATH", EducationLevel: "Junior_Secondary"})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_EmptyCode(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "", EducationLevel: "Junior_Secondary"})
	if err == nil {
		t.Fatal("expected error for empty code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_InvalidCodeFormat(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "math", EducationLevel: "Junior_Secondary"})
	if err == nil {
		t.Fatal("expected error for lowercase code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateLearningArea_InvalidEducationLevel(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateLearningArea(context.Background(), CreateLearningAreaParams{TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Invalid_Level"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Strands
// ============================================================================

func TestCreateStrand_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createStrandFn = func(ctx context.Context, params CreateStrandParams) (string, error) {
		if params.LearningAreaID != "area_001" {
			t.Errorf("expected LearningAreaID 'area_001', got %q", params.LearningAreaID)
		}
		if params.Name != "Numbers" {
			t.Errorf("expected Name 'Numbers', got %q", params.Name)
		}
		return "strand_001", nil
	}

	id, err := h.svc.CreateStrand(context.Background(), CreateStrandParams{LearningAreaID: "area_001", Name: "Numbers"}, "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "strand_001" {
		t.Fatalf("expected id 'strand_001', got %q", id)
	}
}

func TestCreateStrand_EmptyName(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateStrand(context.Background(), CreateStrandParams{LearningAreaID: "area_001", Name: ""}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateStrand_EmptyLearningAreaID(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateStrand(context.Background(), CreateStrandParams{LearningAreaID: "", Name: "Numbers"}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty learning_area_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateStrand_LearningAreaNotInTenant(t *testing.T) {
	h := newTestHarness()

	h.repo.verifyLearningAreaBelongsToTenantFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	_, err := h.svc.CreateStrand(context.Background(), CreateStrandParams{LearningAreaID: "area_999", Name: "Numbers"}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for non-existent learning area, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListStrands_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []Strand{
		{ID: "strand_001", LearningAreaID: "area_001", Name: "Numbers"},
		{ID: "strand_002", LearningAreaID: "area_001", Name: "Algebra"},
	}

	h.repo.listStrandsFn = func(ctx context.Context, learningAreaID string) ([]Strand, error) {
		if learningAreaID != "area_001" {
			t.Errorf("expected learningAreaID 'area_001', got %q", learningAreaID)
		}
		return expected, nil
	}

	strands, err := h.svc.ListStrands(context.Background(), "area_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strands) != 2 {
		t.Fatalf("expected 2 strands, got %d", len(strands))
	}
}

func TestListStrands_EmptyLearningAreaID(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.ListStrands(context.Background(), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateStrand_HappyPath(t *testing.T) {
	h := newTestHarness()

	newName := "Advanced Numbers"
	h.repo.updateStrandFn = func(ctx context.Context, params UpdateStrandParams) error {
		if params.ID != "strand_001" {
			t.Errorf("expected ID 'strand_001', got %q", params.ID)
		}
		if params.Name == nil || *params.Name != "Advanced Numbers" {
			t.Errorf("expected Name 'Advanced Numbers', got %v", params.Name)
		}
		return nil
	}

	err := h.svc.UpdateStrand(context.Background(), UpdateStrandParams{ID: "strand_001", Name: ptrStr(newName)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStrand_EmptyID(t *testing.T) {
	h := newTestHarness()
	err := h.svc.UpdateStrand(context.Background(), UpdateStrandParams{ID: "", Name: ptrStr("Test")})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteStrand_HappyPath(t *testing.T) {
	h := newTestHarness()
	h.repo.deleteStrandFn = func(ctx context.Context, id string) error {
		if id != "strand_001" {
			t.Errorf("expected id 'strand_001', got %q", id)
		}
		return nil
	}
	err := h.svc.DeleteStrand(context.Background(), "strand_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStrand_EmptyID(t *testing.T) {
	h := newTestHarness()
	err := h.svc.DeleteStrand(context.Background(), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Sub-Strands
// ============================================================================

func TestCreateSubStrand_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createSubStrandFn = func(ctx context.Context, params CreateSubStrandParams) (string, error) {
		if params.StrandID != "strand_001" {
			t.Errorf("expected StrandID 'strand_001', got %q", params.StrandID)
		}
		if params.Name != "Addition" {
			t.Errorf("expected Name 'Addition', got %q", params.Name)
		}
		return "sub_001", nil
	}

	id, err := h.svc.CreateSubStrand(context.Background(), CreateSubStrandParams{StrandID: "strand_001", Name: "Addition"}, "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sub_001" {
		t.Fatalf("expected id 'sub_001', got %q", id)
	}
}

func TestCreateSubStrand_EmptyStrandID(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreateSubStrand(context.Background(), CreateSubStrandParams{StrandID: "", Name: "Addition"}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSubStrand_StrandNotInTenant(t *testing.T) {
	h := newTestHarness()
	h.repo.verifyStrandInTenantSchoolFn = func(ctx context.Context, strandID, tenantID, schoolID string) (string, error) {
		return "", ErrNotFound
	}
	_, err := h.svc.CreateSubStrand(context.Background(), CreateSubStrandParams{StrandID: "strand_999", Name: "Addition"}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for non-existent strand, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListSubStrands_HappyPath(t *testing.T) {
	h := newTestHarness()
	expected := []SubStrand{
		{ID: "sub_001", StrandID: "strand_001", Name: "Addition"},
	}
	h.repo.listSubStrandsFn = func(ctx context.Context, strandID string) ([]SubStrand, error) {
		if strandID != "strand_001" {
			t.Errorf("expected strandID 'strand_001', got %q", strandID)
		}
		return expected, nil
	}
	subs, err := h.svc.ListSubStrands(context.Background(), "strand_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub-strand, got %d", len(subs))
	}
}

func TestUpdateSubStrand_HappyPath(t *testing.T) {
	h := newTestHarness()
	newName := "Advanced Addition"
	h.repo.updateSubStrandFn = func(ctx context.Context, params UpdateSubStrandParams) error {
		if params.ID != "sub_001" {
			t.Errorf("expected ID 'sub_001', got %q", params.ID)
		}
		if params.Name == nil || *params.Name != "Advanced Addition" {
			t.Errorf("expected Name 'Advanced Addition', got %v", params.Name)
		}
		return nil
	}
	err := h.svc.UpdateSubStrand(context.Background(), UpdateSubStrandParams{ID: "sub_001", Name: ptrStr(newName)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSubStrand_HappyPath(t *testing.T) {
	h := newTestHarness()
	h.repo.deleteSubStrandFn = func(ctx context.Context, id string) error {
		if id != "sub_001" {
			t.Errorf("expected id 'sub_001', got %q", id)
		}
		return nil
	}
	err := h.svc.DeleteSubStrand(context.Background(), "sub_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests: Performance Indicators
// ============================================================================

func TestCreatePerformanceIndicator_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createPIFn = func(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error) {
		if params.SubStrandID != "sub_001" {
			t.Errorf("expected SubStrandID 'sub_001', got %q", params.SubStrandID)
		}
		if params.Description != "Solve 1+1" {
			t.Errorf("expected Description 'Solve 1+1', got %q", params.Description)
		}
		if params.SequenceOrder == nil || *params.SequenceOrder != 1 {
			t.Errorf("expected SequenceOrder 1, got %v", params.SequenceOrder)
		}
		return "pi_001", nil
	}

	id, err := h.svc.CreatePerformanceIndicator(context.Background(), CreatePerformanceIndicatorParams{
		SubStrandID:   "sub_001",
		Description:   "Solve 1+1",
		SequenceOrder: intPtr(1),
	}, "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "pi_001" {
		t.Fatalf("expected id 'pi_001', got %q", id)
	}
}

func TestCreatePerformanceIndicator_AutoSequence(t *testing.T) {
	h := newTestHarness()

	h.repo.getMaxSeqFn = func(ctx context.Context, subStrandID string) (int, error) {
		if subStrandID != "sub_001" {
			t.Errorf("expected subStrandID 'sub_001', got %q", subStrandID)
		}
		return 3, nil
	}

	h.repo.createPIFn = func(ctx context.Context, params CreatePerformanceIndicatorParams) (string, error) {
		if params.SequenceOrder == nil || *params.SequenceOrder != 4 {
			t.Errorf("expected auto-incremented SequenceOrder 4, got %v", params.SequenceOrder)
		}
		return "pi_004", nil
	}

	id, err := h.svc.CreatePerformanceIndicator(context.Background(), CreatePerformanceIndicatorParams{
		SubStrandID:   "sub_001",
		Description:   "Complex problem solving",
		SequenceOrder: nil,
	}, "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "pi_004" {
		t.Fatalf("expected id 'pi_004', got %q", id)
	}
}

func TestCreatePerformanceIndicator_EmptyDescription(t *testing.T) {
	h := newTestHarness()
	_, err := h.svc.CreatePerformanceIndicator(context.Background(), CreatePerformanceIndicatorParams{
		SubStrandID: "sub_001", Description: "", SequenceOrder: intPtr(1),
	}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty description, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreatePerformanceIndicator_SubStrandNotInTenant(t *testing.T) {
	h := newTestHarness()
	h.repo.verifySubStrandInTenantSchoolFn = func(ctx context.Context, subStrandID, tenantID, schoolID string) (string, error) {
		return "", ErrNotFound
	}
	_, err := h.svc.CreatePerformanceIndicator(context.Background(), CreatePerformanceIndicatorParams{
		SubStrandID: "sub_999", Description: "Test", SequenceOrder: intPtr(1),
	}, "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListPerformanceIndicators_OrderedBySequence(t *testing.T) {
	h := newTestHarness()

	expected := []PerformanceIndicator{
		{ID: "pi_001", SubStrandID: "sub_001", Description: "First", SequenceOrder: 1},
		{ID: "pi_002", SubStrandID: "sub_001", Description: "Second", SequenceOrder: 2},
	}

	h.repo.listPIFn = func(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error) {
		if subStrandID != "sub_001" {
			t.Errorf("expected subStrandID 'sub_001', got %q", subStrandID)
		}
		return expected, nil
	}

	indicators, err := h.svc.ListPerformanceIndicators(context.Background(), "sub_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(indicators) != 2 {
		t.Fatalf("expected 2 indicators, got %d", len(indicators))
	}
	if indicators[0].SequenceOrder != 1 {
		t.Fatalf("expected first SequenceOrder 1, got %d", indicators[0].SequenceOrder)
	}
}

func TestUpdatePerformanceIndicator_HappyPath(t *testing.T) {
	h := newTestHarness()

	newDesc := "Updated description"
	h.repo.updatePIFn = func(ctx context.Context, params UpdatePerformanceIndicatorParams) error {
		if params.ID != "pi_001" {
			t.Errorf("expected ID 'pi_001', got %q", params.ID)
		}
		if params.Description == nil || *params.Description != "Updated description" {
			t.Errorf("unexpected description")
		}
		return nil
	}

	err := h.svc.UpdatePerformanceIndicator(context.Background(), UpdatePerformanceIndicatorParams{
		ID: "pi_001", Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePerformanceIndicator_HappyPath(t *testing.T) {
	h := newTestHarness()
	h.repo.deletePIFn = func(ctx context.Context, id string) error {
		if id != "pi_001" {
			t.Errorf("expected id 'pi_001', got %q", id)
		}
		return nil
	}
	err := h.svc.DeletePerformanceIndicator(context.Background(), "pi_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests: Tree
// ============================================================================

func TestGetTree_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getTreeFn = func(ctx context.Context, learningAreaID string) (*LearningAreaTree, error) {
		return &LearningAreaTree{
			LearningArea: LearningArea{ID: "area_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"},
			Strands: []StrandTree{
				{
					Strand: Strand{ID: "strand_001", LearningAreaID: "area_001", Name: "Numbers"},
					SubStrands: []SubStrandTree{
						{
							SubStrand: SubStrand{ID: "sub_001", StrandID: "strand_001", Name: "Addition"},
							PerformanceIndicators: []PerformanceIndicator{
								{ID: "pi_001", SubStrandID: "sub_001", Description: "1+1", SequenceOrder: 1},
							},
						},
					},
				},
			},
		}, nil
	}

	tree, err := h.svc.GetTree(context.Background(), "area_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree.Strands) != 1 {
		t.Fatalf("expected 1 strand, got %d", len(tree.Strands))
	}
	if len(tree.Strands[0].SubStrands) != 1 {
		t.Fatalf("expected 1 sub-strand, got %d", len(tree.Strands[0].SubStrands))
	}
	if len(tree.Strands[0].SubStrands[0].PerformanceIndicators) != 1 {
		t.Fatalf("expected 1 indicator, got %d", len(tree.Strands[0].SubStrands[0].PerformanceIndicators))
	}
}

func TestGetTree_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.verifyLearningAreaBelongsToTenantFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	_, err := h.svc.GetTree(context.Background(), "area_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Existing Learning Area Tests (refactored to new method names)
// ============================================================================

func TestGetLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getLearningAreaByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
		if id != "area_001" {
			t.Errorf("expected id 'area_001', got %q", id)
		}
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return &LearningArea{ID: id, TenantID: tenantID, SchoolID: schoolID, Name: "Mathematics", Code: "MATH", EducationLevel: "Junior_Secondary"}, nil
	}

	area, err := h.svc.GetLearningArea(context.Background(), "area_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if area.Name != "Mathematics" {
		t.Fatalf("expected Name 'Mathematics', got %q", area.Name)
	}
}

func TestListLearningAreas_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedAreas := []LearningArea{
		{ID: "area_001", TenantID: "tenant_001", SchoolID: "school_001", Name: "English", Code: "ENG", EducationLevel: "Early_Years"},
		{ID: "area_002", TenantID: "tenant_001", SchoolID: "school_001", Name: "Mathematics", Code: "MATH", EducationLevel: "Early_Years"},
	}

	h.repo.listLearningAreasFn = func(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return expectedAreas, nil
	}

	areas, err := h.svc.ListLearningAreas(context.Background(), "tenant_001", "school_001", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(areas) != 2 {
		t.Fatalf("expected 2 areas, got %d", len(areas))
	}
}

func TestUpdateLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()

	newName := "Advanced Mathematics"
	h.repo.updateLearningAreaFn = func(ctx context.Context, params UpdateLearningAreaParams) error {
		if params.ID != "area_001" {
			t.Errorf("expected ID 'area_001', got %q", params.ID)
		}
		if params.Name == nil || *params.Name != "Advanced Mathematics" {
			t.Errorf("unexpected name")
		}
		return nil
	}

	err := h.svc.UpdateLearningArea(context.Background(), UpdateLearningAreaParams{
		ID: "area_001", TenantID: "tenant_001", SchoolID: "school_001", Name: ptrStr(newName),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteLearningArea_HappyPath(t *testing.T) {
	h := newTestHarness()
	h.repo.deleteLearningAreaFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "area_001" {
			t.Errorf("expected id 'area_001', got %q", id)
		}
		return nil
	}
	err := h.svc.DeleteLearningArea(context.Background(), "area_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

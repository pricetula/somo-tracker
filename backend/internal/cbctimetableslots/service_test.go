package cbctimetableslots

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listFn            func(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error)
	listEnrichedFn    func(ctx context.Context, filter SlotFilter) ([]SlotWithEnrichedData, error)
	getByIDFn         func(ctx context.Context, id string) (*TimetableSlot, error)
	getEnrichedByIDFn func(ctx context.Context, id string) (*SlotWithEnrichedData, error)
	createFn          func(ctx context.Context, tenantID, schoolID string, slot CreateSlotPayload) (*TimetableSlot, error)
	batchCreateFn     func(ctx context.Context, tenantID, schoolID string, slots []CreateSlotPayload) ([]TimetableSlot, error)
	updateFn          func(ctx context.Context, id string, slot UpdateSlotPayload) (*TimetableSlot, error)
	deleteFn          func(ctx context.Context, id string) error
	clearDayFn        func(ctx context.Context, structureIDs []string) error
	clearClassDayFn   func(ctx context.Context, structureID, classID string) error
}

func (m *MockRepository) List(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return []TimetableSlot{}, nil
}

func (m *MockRepository) ListEnriched(ctx context.Context, filter SlotFilter) ([]SlotWithEnrichedData, error) {
	if m.listEnrichedFn != nil {
		return m.listEnrichedFn(ctx, filter)
	}
	return []SlotWithEnrichedData{}, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*TimetableSlot, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &TimetableSlot{ID: id}, nil
}

func (m *MockRepository) GetEnrichedByID(ctx context.Context, id string) (*SlotWithEnrichedData, error) {
	if m.getEnrichedByIDFn != nil {
		return m.getEnrichedByIDFn(ctx, id)
	}
	return &SlotWithEnrichedData{TimetableSlot: TimetableSlot{ID: id}}, nil
}

func (m *MockRepository) Create(ctx context.Context, tenantID, schoolID string, slot CreateSlotPayload) (*TimetableSlot, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, schoolID, slot)
	}
	return &TimetableSlot{ID: "slot_001"}, nil
}

func (m *MockRepository) BatchCreate(ctx context.Context, tenantID, schoolID string, slots []CreateSlotPayload) ([]TimetableSlot, error) {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, tenantID, schoolID, slots)
	}
	return []TimetableSlot{}, nil
}

func (m *MockRepository) Update(ctx context.Context, id string, slot UpdateSlotPayload) (*TimetableSlot, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, slot)
	}
	return &TimetableSlot{ID: id}, nil
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) ClearDay(ctx context.Context, structureIDs []string) error {
	if m.clearDayFn != nil {
		return m.clearDayFn(ctx, structureIDs)
	}
	return nil
}

func (m *MockRepository) ClearClassDay(ctx context.Context, structureID, classID string) error {
	if m.clearClassDayFn != nil {
		return m.clearClassDayFn(ctx, structureID, classID)
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

// ============================================================================
// Tests: ListSlots
// ============================================================================

func TestListSlots_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedSlots := []TimetableSlot{
		{ID: "slot_001", ClassID: "class_001", TeacherID: "teacher_001"},
		{ID: "slot_002", ClassID: "class_001", TeacherID: "teacher_002"},
	}

	h.repo.listFn = func(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error) {
		if filter.AcademicYearID != "year_001" {
			t.Errorf("expected AcademicYearID 'year_001', got %q", filter.AcademicYearID)
		}
		if filter.ClassID != "class_001" {
			t.Errorf("expected ClassID 'class_001', got %q", filter.ClassID)
		}
		return expectedSlots, nil
	}

	result, err := h.svc.ListSlots(context.Background(), SlotFilter{
		AcademicYearID: "year_001",
		ClassID:        "class_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(result.Items))
	}
}

func TestListSlots_EmptyAcademicYearID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListSlots(context.Background(), SlotFilter{})
	if err == nil {
		t.Fatal("expected error for empty AcademicYearID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListSlots_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listFn = func(ctx context.Context, filter SlotFilter) ([]TimetableSlot, error) {
		return []TimetableSlot{}, nil
	}

	result, err := h.svc.ListSlots(context.Background(), SlotFilter{AcademicYearID: "year_001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

// ============================================================================
// Tests: ListEnrichedSlots
// ============================================================================

func TestListEnrichedSlots_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []SlotWithEnrichedData{
		{TimetableSlot: TimetableSlot{ID: "slot_001"}, ClassName: "G4 Blue", PeriodName: "Period 1"},
	}

	h.repo.listEnrichedFn = func(ctx context.Context, filter SlotFilter) ([]SlotWithEnrichedData, error) {
		if filter.AcademicYearID != "year_001" {
			t.Errorf("expected AcademicYearID 'year_001', got %q", filter.AcademicYearID)
		}
		return expected, nil
	}

	result, err := h.svc.ListEnrichedSlots(context.Background(), SlotFilter{AcademicYearID: "year_001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if result.Items[0].ClassName != "G4 Blue" {
		t.Fatalf("expected ClassName 'G4 Blue', got %q", result.Items[0].ClassName)
	}
}

func TestListEnrichedSlots_EmptyAcademicYearID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListEnrichedSlots(context.Background(), SlotFilter{})
	if err == nil {
		t.Fatal("expected error for empty AcademicYearID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: GetSlot
// ============================================================================

func TestGetSlot_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getEnrichedByIDFn = func(ctx context.Context, id string) (*SlotWithEnrichedData, error) {
		if id != "slot_001" {
			t.Errorf("expected id 'slot_001', got %q", id)
		}
		return &SlotWithEnrichedData{
			TimetableSlot: TimetableSlot{ID: id},
			ClassName:     "G4 Blue",
			PeriodName:    "Period 1",
		}, nil
	}

	slot, err := h.svc.GetSlot(context.Background(), "slot_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slot.ClassName != "G4 Blue" {
		t.Fatalf("expected ClassName 'G4 Blue', got %q", slot.ClassName)
	}
}

func TestGetSlot_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetSlot(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetSlot_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getEnrichedByIDFn = func(ctx context.Context, id string) (*SlotWithEnrichedData, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetSlot(context.Background(), "slot_999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: CreateSlot
// ============================================================================

func TestCreateSlot_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID string, slot CreateSlotPayload) (*TimetableSlot, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if slot.ClassID != "class_001" {
			t.Errorf("expected ClassID 'class_001', got %q", slot.ClassID)
		}
		if slot.TeacherID != "teacher_001" {
			t.Errorf("expected TeacherID 'teacher_001', got %q", slot.TeacherID)
		}
		return &TimetableSlot{ID: "slot_001", ClassID: "class_001", TeacherID: "teacher_001"}, nil
	}

	slot, err := h.svc.CreateSlot(context.Background(), "tenant_001", "school_001", CreateSlotPayload{
		AcademicYearID: "year_001",
		StructureID:    "struct_001",
		ClassID:        "class_001",
		LearningAreaID: "area_001",
		TeacherID:      "teacher_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slot.ID != "slot_001" {
		t.Fatalf("expected id 'slot_001', got %q", slot.ID)
	}
}

func TestCreateSlot_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSlot(context.Background(), "", "school_001", CreateSlotPayload{
		AcademicYearID: "year_001",
		StructureID:    "struct_001",
		ClassID:        "class_001",
		LearningAreaID: "area_001",
		TeacherID:      "teacher_001",
	})
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSlot_MissingAcademicYearID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSlot(context.Background(), "tenant_001", "school_001", CreateSlotPayload{
		StructureID:    "struct_001",
		ClassID:        "class_001",
		LearningAreaID: "area_001",
		TeacherID:      "teacher_001",
	})
	if err == nil {
		t.Fatal("expected error for missing academic_year_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSlot_MissingStructureID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSlot(context.Background(), "tenant_001", "school_001", CreateSlotPayload{
		AcademicYearID: "year_001",
		ClassID:        "class_001",
		LearningAreaID: "area_001",
		TeacherID:      "teacher_001",
	})
	if err == nil {
		t.Fatal("expected error for missing structure_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: BatchCreateSlots
// ============================================================================

func TestBatchCreateSlots_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedSlots := []TimetableSlot{
		{ID: "slot_001", ClassID: "class_001"},
		{ID: "slot_002", ClassID: "class_001"},
	}

	h.repo.batchCreateFn = func(ctx context.Context, tenantID, schoolID string, slots []CreateSlotPayload) ([]TimetableSlot, error) {
		if len(slots) != 2 {
			t.Errorf("expected 2 slots, got %d", len(slots))
		}
		return expectedSlots, nil
	}

	result, err := h.svc.BatchCreateSlots(context.Background(), "tenant_001", "school_001", BatchCreateSlotsPayload{
		Slots: []CreateSlotPayload{
			{AcademicYearID: "year_001", StructureID: "struct_001", ClassID: "class_001", LearningAreaID: "area_001", TeacherID: "teacher_001"},
			{AcademicYearID: "year_001", StructureID: "struct_002", ClassID: "class_001", LearningAreaID: "area_002", TeacherID: "teacher_001"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestBatchCreateSlots_EmptySlots(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchCreateSlots(context.Background(), "tenant_001", "school_001", BatchCreateSlotsPayload{
		Slots: []CreateSlotPayload{},
	})
	if err == nil {
		t.Fatal("expected error for empty slots, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchCreateSlots_ValidatesEachSlot(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchCreateSlots(context.Background(), "tenant_001", "school_001", BatchCreateSlotsPayload{
		Slots: []CreateSlotPayload{
			{AcademicYearID: "year_001", StructureID: "struct_001", ClassID: "class_001", LearningAreaID: "area_001", TeacherID: "teacher_001"},
			{AcademicYearID: "", StructureID: "struct_002", ClassID: "class_001", LearningAreaID: "area_002", TeacherID: "teacher_001"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error for second slot, got nil")
	}
}

// ============================================================================
// Tests: UpdateSlot
// ============================================================================

func TestUpdateSlot_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.updateFn = func(ctx context.Context, id string, slot UpdateSlotPayload) (*TimetableSlot, error) {
		if id != "slot_001" {
			t.Errorf("expected id 'slot_001', got %q", id)
		}
		if slot.LearningAreaID != "area_002" {
			t.Errorf("expected LearningAreaID 'area_002', got %q", slot.LearningAreaID)
		}
		if slot.TeacherID != "teacher_002" {
			t.Errorf("expected TeacherID 'teacher_002', got %q", slot.TeacherID)
		}
		return &TimetableSlot{ID: id, LearningAreaID: "area_002", TeacherID: "teacher_002"}, nil
	}

	slot, err := h.svc.UpdateSlot(context.Background(), "slot_001", UpdateSlotPayload{
		LearningAreaID: "area_002",
		TeacherID:      "teacher_002",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slot.TeacherID != "teacher_002" {
		t.Fatalf("expected TeacherID 'teacher_002', got %q", slot.TeacherID)
	}
}

func TestUpdateSlot_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateSlot(context.Background(), "", UpdateSlotPayload{
		LearningAreaID: "area_001",
		TeacherID:      "teacher_001",
	})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteSlot
// ============================================================================

func TestDeleteSlot_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, id string) error {
		if id != "slot_001" {
			t.Errorf("expected id 'slot_001', got %q", id)
		}
		return nil
	}

	err := h.svc.DeleteSlot(context.Background(), "slot_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSlot_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteSlot(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteSlot_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, id string) error {
		return ErrNotFound
	}

	err := h.svc.DeleteSlot(context.Background(), "slot_999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: ClearDay
// ============================================================================

func TestClearDay_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.clearDayFn = func(ctx context.Context, structureIDs []string) error {
		if len(structureIDs) != 2 {
			t.Errorf("expected 2 structureIDs, got %d", len(structureIDs))
		}
		if structureIDs[0] != "struct_001" {
			t.Errorf("expected structureID 'struct_001', got %q", structureIDs[0])
		}
		return nil
	}

	err := h.svc.ClearDay(context.Background(), []string{"struct_001", "struct_002"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClearDay_EmptyIDs(t *testing.T) {
	h := newTestHarness()

	err := h.svc.ClearDay(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for empty structureIDs, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ClearClassDay
// ============================================================================

func TestClearClassDay_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.clearClassDayFn = func(ctx context.Context, structureID, classID string) error {
		if structureID != "struct_001" {
			t.Errorf("expected structureID 'struct_001', got %q", structureID)
		}
		if classID != "class_001" {
			t.Errorf("expected classID 'class_001', got %q", classID)
		}
		return nil
	}

	err := h.svc.ClearClassDay(context.Background(), "struct_001", "class_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClearClassDay_EmptyStructureID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.ClearClassDay(context.Background(), "", "class_001")
	if err == nil {
		t.Fatal("expected error for empty structureID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestClearClassDay_EmptyClassID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.ClearClassDay(context.Background(), "struct_001", "")
	if err == nil {
		t.Fatal("expected error for empty classID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

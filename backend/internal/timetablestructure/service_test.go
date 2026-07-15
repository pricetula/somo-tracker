package timetablestructure

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	listByDayFn               func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) ([]TimeBlock, error)
	listAllFn                 func(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error)
	getByIDFn                 func(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error)
	createFn                  func(ctx context.Context, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error)
	batchCreateFn             func(ctx context.Context, tenantID, schoolID string, blocks []CreateTimeBlockPayload) ([]TimeBlock, error)
	replicateDayFn            func(ctx context.Context, tenantID, schoolID string, sourceDay int, targetDays []int) ([]TimeBlock, error)
	updateFn                  func(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error)
	deleteFn                  func(ctx context.Context, id, tenantID, schoolID string) error
	deleteByDayFn             func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) error
	hasLinkedLessonsFn        func(ctx context.Context, id, tenantID, schoolID string) (int, error)
	findOverlappingBlockFn    func(ctx context.Context, tenantID, schoolID string, dayOfWeek int, startTime, endTime string, excludeID string) (*TimeBlock, error)
	listByPeriodNameFn        func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error)
	listByDayAfterFn          func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int, afterTime string, excludeID string) ([]TimeBlock, error)
	batchUpdateBlocksFn       func(ctx context.Context, tenantID, schoolID string, blocks []BatchBlockUpdate) ([]TimeBlock, error)
	deleteByPeriodNameFn      func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string) (int, error)
	hasLinkedLessonsForBlocks func(ctx context.Context, ids []string) (int, error)
}

func (m *MockRepository) ListByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) ([]TimeBlock, error) {
	if m.listByDayFn != nil {
		return m.listByDayFn(ctx, tenantID, schoolID, academicYearID, dayOfWeek)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) ListAll(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, tenantID, schoolID, academicYearID)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID, schoolID)
	}
	return &TimeBlock{ID: id, PeriodName: "Period 1", DayOfWeek: 1, StartTime: "08:00", EndTime: "08:40"}, nil
}

func (m *MockRepository) Create(ctx context.Context, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, schoolID, block)
	}
	return &TimeBlock{ID: "block_001", PeriodName: block.PeriodName, DayOfWeek: block.DayOfWeek}, nil
}

func (m *MockRepository) BatchCreate(ctx context.Context, tenantID, schoolID string, blocks []CreateTimeBlockPayload) ([]TimeBlock, error) {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, tenantID, schoolID, blocks)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) ReplicateDay(ctx context.Context, tenantID, schoolID string, sourceDay int, targetDays []int) ([]TimeBlock, error) {
	if m.replicateDayFn != nil {
		return m.replicateDayFn(ctx, tenantID, schoolID, sourceDay, targetDays)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) Update(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, tenantID, schoolID, block)
	}
	return &TimeBlock{ID: id, PeriodName: block.PeriodName}, nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) DeleteByDay(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) error {
	if m.deleteByDayFn != nil {
		return m.deleteByDayFn(ctx, tenantID, schoolID, academicYearID, dayOfWeek)
	}
	return nil
}

func (m *MockRepository) HasLinkedLessons(ctx context.Context, id, tenantID, schoolID string) (int, error) {
	if m.hasLinkedLessonsFn != nil {
		return m.hasLinkedLessonsFn(ctx, id, tenantID, schoolID)
	}
	return 0, nil
}

func (m *MockRepository) FindOverlappingBlock(ctx context.Context, tenantID, schoolID string, dayOfWeek int, startTime, endTime string, excludeID string) (*TimeBlock, error) {
	if m.findOverlappingBlockFn != nil {
		return m.findOverlappingBlockFn(ctx, tenantID, schoolID, dayOfWeek, startTime, endTime, excludeID)
	}
	return nil, nil
}

func (m *MockRepository) ListByPeriodName(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error) {
	if m.listByPeriodNameFn != nil {
		return m.listByPeriodNameFn(ctx, tenantID, schoolID, academicYearID, periodName, excludeID)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) ListByDayAfter(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int, afterTime string, excludeID string) ([]TimeBlock, error) {
	if m.listByDayAfterFn != nil {
		return m.listByDayAfterFn(ctx, tenantID, schoolID, academicYearID, dayOfWeek, afterTime, excludeID)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) BatchUpdateBlocks(ctx context.Context, tenantID, schoolID string, blocks []BatchBlockUpdate) ([]TimeBlock, error) {
	if m.batchUpdateBlocksFn != nil {
		return m.batchUpdateBlocksFn(ctx, tenantID, schoolID, blocks)
	}
	return []TimeBlock{}, nil
}

func (m *MockRepository) DeleteByPeriodName(ctx context.Context, tenantID, schoolID, academicYearID, periodName string) (int, error) {
	if m.deleteByPeriodNameFn != nil {
		return m.deleteByPeriodNameFn(ctx, tenantID, schoolID, academicYearID, periodName)
	}
	return 0, nil
}

func (m *MockRepository) HasLinkedLessonsForBlocks(ctx context.Context, ids []string) (int, error) {
	if m.hasLinkedLessonsForBlocks != nil {
		return m.hasLinkedLessonsForBlocks(ctx, ids)
	}
	return 0, nil
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
// Tests: ListBlocksByDay
// ============================================================================

func TestListBlocksByDay_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []TimeBlock{
		{ID: "block_001", DayOfWeek: 1, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40"},
		{ID: "block_002", DayOfWeek: 1, PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20"},
	}

	h.repo.listByDayFn = func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) ([]TimeBlock, error) {
		if dayOfWeek != 1 {
			t.Errorf("expected dayOfWeek 1 (Monday), got %d", dayOfWeek)
		}
		return expected, nil
	}

	result, err := h.svc.ListBlocksByDay(context.Background(), "tenant_001", "school_001", "year_001", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if result.Items[0].PeriodName != "Period 1" {
		t.Fatalf("expected 'Period 1', got %q", result.Items[0].PeriodName)
	}
}

func TestListBlocksByDay_InvalidDay(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListBlocksByDay(context.Background(), "tenant_001", "school_001", "year_001", 0)
	if err == nil {
		t.Fatal("expected error for invalid day 0, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListBlocksByDay_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListBlocksByDay(context.Background(), "", "school_001", "year_001", 1)
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListBlocksByDay_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listByDayFn = func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) ([]TimeBlock, error) {
		return []TimeBlock{}, nil
	}

	result, err := h.svc.ListBlocksByDay(context.Background(), "tenant_001", "school_001", "year_001", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

// ============================================================================
// Tests: ListAllBlocks
// ============================================================================

func TestListAllBlocks_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []TimeBlock{
		{ID: "block_001", DayOfWeek: 1, PeriodName: "Period 1"},
		{ID: "block_002", DayOfWeek: 2, PeriodName: "Period 1"},
	}

	h.repo.listAllFn = func(ctx context.Context, tenantID, schoolID, academicYearID string) ([]TimeBlock, error) {
		return expected, nil
	}

	result, err := h.svc.ListAllBlocks(context.Background(), "tenant_001", "school_001", "year_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestListAllBlocks_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListAllBlocks(context.Background(), "tenant_001", "", "year_001")
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: CreateBlock
// ============================================================================

func TestCreateBlock_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
		if block.PeriodName != "Period 1" {
			t.Errorf("expected PeriodName 'Period 1', got %q", block.PeriodName)
		}
		if block.StartTime != "08:00" {
			t.Errorf("expected StartTime '08:00', got %q", block.StartTime)
		}
		if block.EndTime != "08:40" {
			t.Errorf("expected EndTime '08:40', got %q", block.EndTime)
		}
		if !block.IsBreak {
			t.Error("expected IsBreak true")
		}
		return &TimeBlock{ID: "block_001", PeriodName: "Period 1", DayOfWeek: 1}, nil
	}

	block, err := h.svc.CreateBlock(context.Background(), "tenant_001", "school_001", CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        true,
		AcademicYearID: "year_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if block.ID != "block_001" {
		t.Fatalf("expected id 'block_001', got %q", block.ID)
	}
}

func TestCreateBlock_InvalidDayOfWeek(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateBlock(context.Background(), "tenant_001", "school_001", CreateTimeBlockPayload{
		DayOfWeek:      0,
		PeriodName:     "Period 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for invalid day of week, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateBlock_EndBeforeStart(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateBlock(context.Background(), "tenant_001", "school_001", CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "09:00",
		EndTime:        "08:00",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for end before start, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateBlock_EmptyPeriodName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateBlock(context.Background(), "tenant_001", "school_001", CreateTimeBlockPayload{
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for empty period name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateBlock_EmptyTenantID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateBlock(context.Background(), "", "school_001", CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for empty tenantID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: BatchCreateBlocks
// ============================================================================

func TestBatchCreateBlocks_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedBlocks := []TimeBlock{
		{ID: "block_001", PeriodName: "Period 1", DayOfWeek: 1},
		{ID: "block_002", PeriodName: "Period 2", DayOfWeek: 1},
	}

	h.repo.batchCreateFn = func(ctx context.Context, tenantID, schoolID string, blocks []CreateTimeBlockPayload) ([]TimeBlock, error) {
		if len(blocks) != 2 {
			t.Errorf("expected 2 blocks, got %d", len(blocks))
		}
		return expectedBlocks, nil
	}

	result, err := h.svc.BatchCreateBlocks(context.Background(), "tenant_001", "school_001", BatchCreateTimeBlockPayload{
		Blocks: []CreateTimeBlockPayload{
			{DayOfWeek: 1, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40", AcademicYearID: "year_001"},
			{DayOfWeek: 1, PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20", AcademicYearID: "year_001"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestBatchCreateBlocks_EmptyBlocks(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchCreateBlocks(context.Background(), "tenant_001", "school_001", BatchCreateTimeBlockPayload{
		Blocks: []CreateTimeBlockPayload{},
	})
	if err == nil {
		t.Fatal("expected error for empty blocks, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchCreateBlocks_ValidatesEachBlock(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchCreateBlocks(context.Background(), "tenant_001", "school_001", BatchCreateTimeBlockPayload{
		Blocks: []CreateTimeBlockPayload{
			{DayOfWeek: 1, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40", AcademicYearID: "year_001"},
			{DayOfWeek: 0, PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20", AcademicYearID: "year_001"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error for second block, got nil")
	}
}

// ============================================================================
// Tests: ReplicateDayBlocks
// ============================================================================

func TestReplicateDayBlocks_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.replicateDayFn = func(ctx context.Context, tenantID, schoolID string, sourceDay int, targetDays []int) ([]TimeBlock, error) {
		if sourceDay != 1 {
			t.Errorf("expected sourceDay 1, got %d", sourceDay)
		}
		if len(targetDays) != 2 || targetDays[0] != 2 || targetDays[1] != 3 {
			t.Errorf("expected targetDays [2, 3], got %v", targetDays)
		}
		return []TimeBlock{
			{ID: "block_001", DayOfWeek: 2, PeriodName: "Period 1"},
			{ID: "block_002", DayOfWeek: 3, PeriodName: "Period 1"},
		}, nil
	}

	result, err := h.svc.ReplicateDayBlocks(context.Background(), "tenant_001", "school_001", ReplicateDayPayload{
		SourceDay:  1,
		TargetDays: []int{2, 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

func TestReplicateDayBlocks_InvalidSourceDay(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ReplicateDayBlocks(context.Background(), "tenant_001", "school_001", ReplicateDayPayload{
		SourceDay:  0,
		TargetDays: []int{2},
	})
	if err == nil {
		t.Fatal("expected error for invalid source day, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReplicateDayBlocks_NoTargetDays(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ReplicateDayBlocks(context.Background(), "tenant_001", "school_001", ReplicateDayPayload{
		SourceDay:  1,
		TargetDays: []int{},
	})
	if err == nil {
		t.Fatal("expected error for empty target days, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReplicateDayBlocks_InvalidTargetDay(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ReplicateDayBlocks(context.Background(), "tenant_001", "school_001", ReplicateDayPayload{
		SourceDay:  1,
		TargetDays: []int{8},
	})
	if err == nil {
		t.Fatal("expected error for invalid target day 8, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: UpdateBlock (basic)
// ============================================================================

func TestUpdateBlock_HappyPath_NoPropagate(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
		return &TimeBlock{
			ID: id, PeriodName: "Period 1", DayOfWeek: 1,
			StartTime: "08:00", EndTime: "08:40",
		}, nil
	}

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
		if block.StartTime != "08:30" {
			t.Errorf("expected StartTime '08:30', got %q", block.StartTime)
		}
		if block.EndTime != "09:10" {
			t.Errorf("expected EndTime '09:10', got %q", block.EndTime)
		}
		return &TimeBlock{
			ID: id, PeriodName: "Period 1", DayOfWeek: 1,
			StartTime: "08:30", EndTime: "09:10",
		}, nil
	}

	result, err := h.svc.UpdateBlock(context.Background(), "block_001", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "08:30",
		EndTime:        "09:10",
		AcademicYearID: "year_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1 (only the updated block), got %d", result.Total)
	}
}

func TestUpdateBlock_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateBlock(context.Background(), "", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateBlock_InvalidDayOfWeek(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateBlock(context.Background(), "block_001", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:  0,
		PeriodName: "Period 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
	})
	if err == nil {
		t.Fatal("expected error for invalid day of week, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateBlock_EndNotAfterStart(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateBlock(context.Background(), "block_001", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "09:00",
		EndTime:        "08:00",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for end before start, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateBlock_InvalidPropagateMode(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.UpdateBlock(context.Background(), "block_001", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: "year_001",
		Propagate:      "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid propagate mode, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: UpdateBlock with propagation
// ============================================================================

func TestUpdateBlock_PropagateAllDays(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
		return &TimeBlock{
			ID: id, PeriodName: "Period 1", DayOfWeek: 1,
			StartTime: "08:00", EndTime: "08:40",
		}, nil
	}

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
		return &TimeBlock{
			ID: id, PeriodName: "Period 1", DayOfWeek: 1,
			StartTime: "08:30", EndTime: "09:10",
		}, nil
	}

	// Same-named blocks on other days to be cascaded
	sameNameBlocks := []TimeBlock{
		{ID: "block_002", PeriodName: "Period 1", DayOfWeek: 2, StartTime: "08:00", EndTime: "08:40"},
		{ID: "block_003", PeriodName: "Period 1", DayOfWeek: 3, StartTime: "08:00", EndTime: "08:40"},
	}

	h.repo.listByPeriodNameFn = func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error) {
		if periodName != "Period 1" {
			t.Errorf("expected periodName 'Period 1', got %q", periodName)
		}
		return sameNameBlocks, nil
	}

	batchCalled := false
	h.repo.batchUpdateBlocksFn = func(ctx context.Context, tenantID, schoolID string, blocks []BatchBlockUpdate) ([]TimeBlock, error) {
		batchCalled = true
		if len(blocks) != 2 {
			t.Errorf("expected 2 blocks to cascade, got %d", len(blocks))
		}
		result := make([]TimeBlock, len(blocks))
		for i, b := range blocks {
			result[i] = TimeBlock{ID: b.ID, StartTime: sameNameBlocks[i].StartTime, EndTime: sameNameBlocks[i].EndTime}
		}
		return result, nil
	}

	result, err := h.svc.UpdateBlock(context.Background(), "block_001", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "08:30",
		EndTime:        "09:10",
		IsBreak:        false,
		AcademicYearID: "year_001",
		Propagate:      "all_days",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3 (1 updated + 2 cascaded), got %d", result.Total)
	}
	if !batchCalled {
		t.Fatal("expected batchUpdateBlocksFn to be called for propagation")
	}
}

// ============================================================================
// Tests: UpdateBlock with shift-following
// ============================================================================

func TestUpdateBlock_ShiftFollowing(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*TimeBlock, error) {
		return &TimeBlock{
			ID: id, PeriodName: "Period 1", DayOfWeek: 1,
			StartTime: "08:00", EndTime: "08:40",
		}, nil
	}

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID string, block CreateTimeBlockPayload) (*TimeBlock, error) {
		return &TimeBlock{
			ID: id, PeriodName: "Period 1", DayOfWeek: 1,
			StartTime: "09:00", EndTime: "09:40",
		}, nil
	}

	// Blocks after the shifted block's end time will be shifted by +60min
	subsequentBlocks := []TimeBlock{
		{ID: "block_004", DayOfWeek: 1, PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20"},
		{ID: "block_005", DayOfWeek: 1, PeriodName: "Period 3", StartTime: "09:20", EndTime: "10:00"},
	}

	h.repo.listByDayAfterFn = func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int, afterTime string, excludeID string) ([]TimeBlock, error) {
		return subsequentBlocks, nil
	}

	batchCalled := false
	h.repo.batchUpdateBlocksFn = func(ctx context.Context, tenantID, schoolID string, blocks []BatchBlockUpdate) ([]TimeBlock, error) {
		batchCalled = true
		// After shifting by +60min: block_004 should be 09:40-10:20, block_005 should be 10:20-11:00
		if len(blocks) != 2 {
			t.Errorf("expected 2 blocks to shift, got %d", len(blocks))
		}
		if blocks[0].StartTime != "09:40" {
			t.Errorf("expected first shifted StartTime '09:40', got %q", blocks[0].StartTime)
		}
		if blocks[1].StartTime != "10:20" {
			t.Errorf("expected second shifted StartTime '10:20', got %q", blocks[1].StartTime)
		}
		result := make([]TimeBlock, len(blocks))
		for i, b := range blocks {
			result[i] = TimeBlock{ID: b.ID, StartTime: b.StartTime, EndTime: b.EndTime}
		}
		return result, nil
	}

	result, err := h.svc.UpdateBlock(context.Background(), "block_001", "tenant_001", "school_001", UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Period 1",
		StartTime:      "09:00",
		EndTime:        "09:40",
		IsBreak:        false,
		AcademicYearID: "year_001",
		ShiftFollowing: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3 (1 updated + 2 shifted), got %d", result.Total)
	}
	if !batchCalled {
		t.Fatal("expected batchUpdateBlocksFn to be called for shift")
	}
}

// ============================================================================
// Tests: DeleteBlock
// ============================================================================

func TestDeleteBlock_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.hasLinkedLessonsFn = func(ctx context.Context, id, tenantID, schoolID string) (int, error) {
		if id != "block_001" {
			t.Errorf("expected id 'block_001', got %q", id)
		}
		return 0, nil
	}

	called := false
	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		called = true
		return nil
	}

	result, err := h.svc.DeleteBlock(context.Background(), "block_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Deleted {
		t.Fatal("expected Deleted true")
	}
	if !called {
		t.Fatal("expected deleteFn to be called")
	}
}

func TestDeleteBlock_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.DeleteBlock(context.Background(), "", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteBlock_HasLinkedLessons(t *testing.T) {
	h := newTestHarness()

	h.repo.hasLinkedLessonsFn = func(ctx context.Context, id, tenantID, schoolID string) (int, error) {
		return 5, nil
	}

	result, err := h.svc.DeleteBlock(context.Background(), "block_001", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for linked lessons, got nil")
	}
	if !errors.Is(err, ErrBlockHasLessons) {
		t.Fatalf("expected ErrBlockHasLessons, got %v", err)
	}
	if result == nil || result.Deleted {
		t.Fatal("expected Deleted false in result")
	}
	if result.LinkedLessons != 5 {
		t.Fatalf("expected LinkedLessons 5, got %d", result.LinkedLessons)
	}
}

func TestDeleteBlock_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.hasLinkedLessonsFn = func(ctx context.Context, id, tenantID, schoolID string) (int, error) {
		return 0, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	_, err := h.svc.DeleteBlock(context.Background(), "block_999", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for non-existent block, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteBlocksByName
// ============================================================================

func TestDeleteBlocksByName_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.listByPeriodNameFn = func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error) {
		return []TimeBlock{
			{ID: "block_001", PeriodName: "Break", DayOfWeek: 1},
			{ID: "block_002", PeriodName: "Break", DayOfWeek: 2},
		}, nil
	}

	h.repo.deleteByPeriodNameFn = func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string) (int, error) {
		if periodName != "Break" {
			t.Errorf("expected periodName 'Break', got %q", periodName)
		}
		return 2, nil
	}

	result, err := h.svc.DeleteBlocksByName(context.Background(), "tenant_001", "school_001", DeleteByNamePayload{
		PeriodName:     "Break",
		AcademicYearID: "year_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Deleted {
		t.Fatal("expected Deleted true")
	}
	if result.DeletedCount != 2 {
		t.Fatalf("expected DeletedCount 2, got %d", result.DeletedCount)
	}
}

func TestDeleteBlocksByName_EmptyPeriodName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.DeleteBlocksByName(context.Background(), "tenant_001", "school_001", DeleteByNamePayload{
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for empty period name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteBlocksByName_HasLinkedLessons(t *testing.T) {
	h := newTestHarness()

	h.repo.listByPeriodNameFn = func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error) {
		return []TimeBlock{
			{ID: "block_001"},
			{ID: "block_002"},
			{ID: "block_003"},
		}, nil
	}

	h.repo.hasLinkedLessonsForBlocks = func(ctx context.Context, ids []string) (int, error) {
		if len(ids) != 3 {
			t.Errorf("expected 3 ids, got %d", len(ids))
		}
		return 2, nil
	}

	_, err := h.svc.DeleteBlocksByName(context.Background(), "tenant_001", "school_001", DeleteByNamePayload{
		PeriodName:     "Break",
		AcademicYearID: "year_001",
	})
	if err == nil {
		t.Fatal("expected error for linked lessons, got nil")
	}
	if !errors.Is(err, ErrBlockHasLessons) {
		t.Fatalf("expected ErrBlockHasLessons, got %v", err)
	}
}

func TestDeleteBlocksByName_NoBlocksFound(t *testing.T) {
	h := newTestHarness()

	h.repo.listByPeriodNameFn = func(ctx context.Context, tenantID, schoolID, academicYearID, periodName string, excludeID string) ([]TimeBlock, error) {
		return []TimeBlock{}, nil
	}

	result, err := h.svc.DeleteBlocksByName(context.Background(), "tenant_001", "school_001", DeleteByNamePayload{
		PeriodName:     "NonExistent",
		AcademicYearID: "year_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted {
		t.Fatal("expected Deleted false when no blocks found")
	}
}

// ============================================================================
// Tests: DeleteDayBlocks
// ============================================================================

func TestDeleteDayBlocks_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteByDayFn = func(ctx context.Context, tenantID, schoolID, academicYearID string, dayOfWeek int) error {
		if dayOfWeek != 1 {
			t.Errorf("expected dayOfWeek 1, got %d", dayOfWeek)
		}
		return nil
	}

	err := h.svc.DeleteDayBlocks(context.Background(), "tenant_001", "school_001", "year_001", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDayBlocks_InvalidDay(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteDayBlocks(context.Background(), "tenant_001", "school_001", "year_001", 8)
	if err == nil {
		t.Fatal("expected error for invalid day, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteDayBlocks_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.DeleteDayBlocks(context.Background(), "tenant_001", "", "year_001", 1)
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: parseTimeMinutes / formatMinutes helpers
// ============================================================================

func TestParseTimeMinutes(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"00:00", 0},
		{"08:00", 480},
		{"12:00", 720},
		{"12:30", 750},
		{"23:59", 1439},
		{"24:00", 1440},
	}
	for _, tc := range tests {
		got := parseTimeMinutes(tc.input)
		if got != tc.expected {
			t.Errorf("parseTimeMinutes(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestParseTimeMinutes_ShortString(t *testing.T) {
	got := parseTimeMinutes("08")
	if got != 0 {
		t.Errorf("expected 0 for short string, got %d", got)
	}
}

func TestFormatMinutes(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "00:00"},
		{60, "01:00"},
		{480, "08:00"},
		{750, "12:30"},
		{1439, "23:59"},
		{1440, "23:59"}, // clamped
		{-1, "00:00"},   // clamped
	}
	for _, tc := range tests {
		got := formatMinutes(tc.input)
		if got != tc.expected {
			t.Errorf("formatMinutes(%d) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

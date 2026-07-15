package behavior

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
	listCategoriesFn        func(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error)
	listActiveCategoriesFn  func(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error)
	createCategoryFn        func(ctx context.Context, tenantID, schoolID, name string, defaultSeverity *string) (*BehaviorCategory, error)
	getCategoryByIDFn       func(ctx context.Context, id, tenantID string) (*BehaviorCategory, error)
	updateCategoryFn        func(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error)
	createNoteFn            func(ctx context.Context, tenantID, schoolID string, payload CreateNotePayload, authoredBy string) (*BehaviorNote, error)
	getPendingQueueFn       func(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error)
	getNoteByIDFn           func(ctx context.Context, id, tenantID string) (*BehaviorNote, error)
	reviewNoteFn            func(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error
	getNotesByStudentTermFn func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error)
}

func (m *MockRepository) ListCategories(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
	if m.listCategoriesFn != nil {
		return m.listCategoriesFn(ctx, tenantID, schoolID)
	}
	return []BehaviorCategory{}, nil
}

func (m *MockRepository) ListActiveCategories(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
	if m.listActiveCategoriesFn != nil {
		return m.listActiveCategoriesFn(ctx, tenantID, schoolID)
	}
	return []BehaviorCategory{}, nil
}

func (m *MockRepository) CreateCategory(ctx context.Context, tenantID, schoolID, name string, defaultSeverity *string) (*BehaviorCategory, error) {
	if m.createCategoryFn != nil {
		return m.createCategoryFn(ctx, tenantID, schoolID, name, defaultSeverity)
	}
	return &BehaviorCategory{ID: "cat_001", Name: name, IsActive: true}, nil
}

func (m *MockRepository) GetCategoryByID(ctx context.Context, id, tenantID string) (*BehaviorCategory, error) {
	if m.getCategoryByIDFn != nil {
		return m.getCategoryByIDFn(ctx, id, tenantID)
	}
	return &BehaviorCategory{ID: id, Name: "Test Category", IsActive: true}, nil
}

func (m *MockRepository) UpdateCategory(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error) {
	if m.updateCategoryFn != nil {
		return m.updateCategoryFn(ctx, id, tenantID, payload)
	}
	return &BehaviorCategory{ID: id, Name: "Updated", IsActive: true}, nil
}

func (m *MockRepository) CreateNote(ctx context.Context, tenantID, schoolID string, payload CreateNotePayload, authoredBy string) (*BehaviorNote, error) {
	if m.createNoteFn != nil {
		return m.createNoteFn(ctx, tenantID, schoolID, payload, authoredBy)
	}
	return &BehaviorNote{
		ID: "note_001", StudentID: payload.StudentID,
		CategoryID: payload.CategoryID, Status: StatusPendingReview,
	}, nil
}

func (m *MockRepository) GetPendingQueue(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error) {
	if m.getPendingQueueFn != nil {
		return m.getPendingQueueFn(ctx, tenantID, schoolID)
	}
	return &PendingNotesResponse{Notes: []PendingNoteItem{}}, nil
}

func (m *MockRepository) GetNoteByID(ctx context.Context, id, tenantID string) (*BehaviorNote, error) {
	if m.getNoteByIDFn != nil {
		return m.getNoteByIDFn(ctx, id, tenantID)
	}
	return &BehaviorNote{ID: id, Status: StatusPendingReview}, nil
}

func (m *MockRepository) ReviewNote(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error {
	if m.reviewNoteFn != nil {
		return m.reviewNoteFn(ctx, id, tenantID, reviewedBy, decision)
	}
	return nil
}

func (m *MockRepository) GetNotesByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error) {
	if m.getNotesByStudentTermFn != nil {
		return m.getNotesByStudentTermFn(ctx, tenantID, schoolID, studentID, termID)
	}
	return []PendingNoteItem{}, nil
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
// Tests: Categories — ListCategories
// ============================================================================

func TestListCategories_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []BehaviorCategory{
		{ID: "cat_001", Name: "Bullying", IsActive: true},
		{ID: "cat_002", Name: "Tardiness", IsActive: false},
	}

	h.repo.listCategoriesFn = func(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		return expected, nil
	}

	categories, err := h.svc.ListCategories(context.Background(), "tenant_001", "school_001", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}
	if categories[0].Name != "Bullying" {
		t.Fatalf("expected first category 'Bullying', got %q", categories[0].Name)
	}
}

func TestListCategories_ActiveOnly(t *testing.T) {
	h := newTestHarness()

	expected := []BehaviorCategory{
		{ID: "cat_001", Name: "Bullying", IsActive: true},
	}

	h.repo.listActiveCategoriesFn = func(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		return expected, nil
	}

	categories, err := h.svc.ListCategories(context.Background(), "tenant_001", "school_001", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Name != "Bullying" {
		t.Fatalf("expected category 'Bullying', got %q", categories[0].Name)
	}
}

func TestListCategories_EmptyResults(t *testing.T) {
	h := newTestHarness()

	h.repo.listCategoriesFn = func(ctx context.Context, tenantID, schoolID string) ([]BehaviorCategory, error) {
		return []BehaviorCategory{}, nil
	}

	categories, err := h.svc.ListCategories(context.Background(), "tenant_001", "school_001", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(categories))
	}
}

// ============================================================================
// Tests: Categories — CreateCategory
// ============================================================================

func TestCreateCategory_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createCategoryFn = func(ctx context.Context, tenantID, schoolID, name string, defaultSeverity *string) (*BehaviorCategory, error) {
		if name != "Bullying" {
			t.Errorf("expected name 'Bullying', got %q", name)
		}
		if defaultSeverity != nil {
			t.Errorf("expected nil defaultSeverity, got %q", *defaultSeverity)
		}
		return &BehaviorCategory{ID: "cat_001", Name: "Bullying", IsActive: true}, nil
	}

	cat, err := h.svc.CreateCategory(context.Background(), "tenant_001", "school_001", CreateCategoryPayload{Name: "Bullying"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.ID != "cat_001" {
		t.Fatalf("expected id 'cat_001', got %q", cat.ID)
	}
	if cat.Name != "Bullying" {
		t.Fatalf("expected name 'Bullying', got %q", cat.Name)
	}
}

func TestCreateCategory_WithDefaultSeverity(t *testing.T) {
	h := newTestHarness()

	severity := "MINOR"

	h.repo.createCategoryFn = func(ctx context.Context, tenantID, schoolID, name string, defaultSeverity *string) (*BehaviorCategory, error) {
		if defaultSeverity == nil || *defaultSeverity != "MINOR" {
			t.Errorf("expected defaultSeverity 'MINOR', got %v", defaultSeverity)
		}
		return &BehaviorCategory{ID: "cat_002", Name: "Late Arrival", IsActive: true, DefaultSeverity: &severity}, nil
	}

	cat, err := h.svc.CreateCategory(context.Background(), "tenant_001", "school_001", CreateCategoryPayload{
		Name:            "Late Arrival",
		DefaultSeverity: &severity,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.DefaultSeverity == nil || *cat.DefaultSeverity != "MINOR" {
		t.Fatalf("expected defaultSeverity 'MINOR', got %v", cat.DefaultSeverity)
	}
}

func TestCreateCategory_EmptyName(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateCategory(context.Background(), "tenant_001", "school_001", CreateCategoryPayload{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateCategory_InvalidSeverity(t *testing.T) {
	h := newTestHarness()

	invalid := "CRITICAL"
	_, err := h.svc.CreateCategory(context.Background(), "tenant_001", "school_001", CreateCategoryPayload{
		Name:            "Test",
		DefaultSeverity: &invalid,
	})
	if err == nil {
		t.Fatal("expected error for invalid severity, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Categories — UpdateCategory
// ============================================================================

func TestUpdateCategory_HappyPath(t *testing.T) {
	h := newTestHarness()

	newName := "Updated Name"

	h.repo.updateCategoryFn = func(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error) {
		if id != "cat_001" {
			t.Errorf("expected id 'cat_001', got %q", id)
		}
		if payload.Name == nil || *payload.Name != "Updated Name" {
			t.Errorf("expected name 'Updated Name', got %v", payload.Name)
		}
		return &BehaviorCategory{ID: id, Name: "Updated Name", IsActive: true}, nil
	}

	cat, err := h.svc.UpdateCategory(context.Background(), "cat_001", "tenant_001", UpdateCategoryPayload{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %q", cat.Name)
	}
}

func TestUpdateCategory_EmptyID(t *testing.T) {
	h := newTestHarness()

	name := "Test"
	_, err := h.svc.UpdateCategory(context.Background(), "", "tenant_001", UpdateCategoryPayload{Name: &name})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateCategory_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.updateCategoryFn = func(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error) {
		return nil, ErrNotFound
	}

	name := "Test"
	_, err := h.svc.UpdateCategory(context.Background(), "cat_999", "tenant_001", UpdateCategoryPayload{Name: &name})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: Categories — GetCategory
// ============================================================================

func TestGetCategory_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.getCategoryByIDFn = func(ctx context.Context, id, tenantID string) (*BehaviorCategory, error) {
		if id != "cat_001" {
			t.Errorf("expected id 'cat_001', got %q", id)
		}
		return &BehaviorCategory{ID: id, Name: "Bullying", IsActive: true}, nil
	}

	cat, err := h.svc.GetCategory(context.Background(), "cat_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.Name != "Bullying" {
		t.Fatalf("expected name 'Bullying', got %q", cat.Name)
	}
}

func TestGetCategory_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getCategoryByIDFn = func(ctx context.Context, id, tenantID string) (*BehaviorCategory, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetCategory(context.Background(), "cat_999", "tenant_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: Notes — CreateNote
// ============================================================================

func TestCreateNote_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createNoteFn = func(ctx context.Context, tenantID, schoolID string, payload CreateNotePayload, authoredBy string) (*BehaviorNote, error) {
		if payload.StudentID != "stu_001" {
			t.Errorf("expected studentID 'stu_001', got %q", payload.StudentID)
		}
		if payload.CategoryID != "cat_001" {
			t.Errorf("expected categoryID 'cat_001', got %q", payload.CategoryID)
		}
		if payload.Description != "Disruptive behavior" {
			t.Errorf("expected description 'Disruptive behavior', got %q", payload.Description)
		}
		if authoredBy != "teacher_001" {
			t.Errorf("expected authoredBy 'teacher_001', got %q", authoredBy)
		}
		return &BehaviorNote{
			ID: "note_001", StudentID: "stu_001",
			CategoryID: "cat_001", Description: "Disruptive behavior",
			Status: StatusPendingReview, AuthoredByID: "teacher_001",
		}, nil
	}

	note, err := h.svc.CreateNote(context.Background(), "tenant_001", "school_001", CreateNotePayload{
		StudentID:       "stu_001",
		CategoryID:      "cat_001",
		Description:     "Disruptive behavior",
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
	}, "teacher_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.ID != "note_001" {
		t.Fatalf("expected note id 'note_001', got %q", note.ID)
	}
	if note.Status != StatusPendingReview {
		t.Fatalf("expected status 'PENDING_REVIEW', got %q", note.Status)
	}
}

func TestCreateNote_EmptyStudentID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateNote(context.Background(), "tenant_001", "school_001", CreateNotePayload{
		CategoryID:      "cat_001",
		Description:     "Test",
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
	}, "teacher_001")
	if err == nil {
		t.Fatal("expected error for empty studentID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateNote_EmptyCategoryID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateNote(context.Background(), "tenant_001", "school_001", CreateNotePayload{
		StudentID:       "stu_001",
		Description:     "Test",
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
	}, "teacher_001")
	if err == nil {
		t.Fatal("expected error for empty categoryID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateNote_EmptyDescription(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateNote(context.Background(), "tenant_001", "school_001", CreateNotePayload{
		StudentID:       "stu_001",
		CategoryID:      "cat_001",
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
	}, "teacher_001")
	if err == nil {
		t.Fatal("expected error for empty description, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Notes — GetPendingQueue
// ============================================================================

func TestGetPendingQueue_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedNotes := []PendingNoteItem{
		{ID: "note_001", StudentID: "stu_001", StudentFullName: "John Doe", IsUrgent: true, Status: StatusPendingReview},
		{ID: "note_002", StudentID: "stu_002", StudentFullName: "Jane Smith", IsUrgent: false, Status: StatusPendingReview},
	}

	h.repo.getPendingQueueFn = func(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return &PendingNotesResponse{Notes: expectedNotes}, nil
	}

	result, err := h.svc.GetPendingQueue(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(result.Notes))
	}
	if !result.Notes[0].IsUrgent {
		t.Fatal("expected first note to be urgent (sorted by urgency)")
	}
}

func TestGetPendingQueue_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.getPendingQueueFn = func(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error) {
		return &PendingNotesResponse{Notes: []PendingNoteItem{}}, nil
	}

	result, err := h.svc.GetPendingQueue(context.Background(), "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Notes) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(result.Notes))
	}
}

// ============================================================================
// Tests: Notes — ReviewNote
// ============================================================================

func TestReviewNote_Approve(t *testing.T) {
	h := newTestHarness()

	h.repo.reviewNoteFn = func(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error {
		if decision.Decision != "APPROVED" {
			t.Errorf("expected decision 'APPROVED', got %q", decision.Decision)
		}
		if reviewedBy != "admin_001" {
			t.Errorf("expected reviewedBy 'admin_001', got %q", reviewedBy)
		}
		return nil
	}

	err := h.svc.ReviewNote(context.Background(), "note_001", "tenant_001", "admin_001", ReviewDecisionPayload{
		Decision: "APPROVED",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewNote_Reject(t *testing.T) {
	h := newTestHarness()

	h.repo.reviewNoteFn = func(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error {
		if decision.Decision != "REJECTED" {
			t.Errorf("expected decision 'REJECTED', got %q", decision.Decision)
		}
		return nil
	}

	err := h.svc.ReviewNote(context.Background(), "note_001", "tenant_001", "admin_001", ReviewDecisionPayload{
		Decision: "REJECTED",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewNote_InvalidDecision(t *testing.T) {
	h := newTestHarness()

	err := h.svc.ReviewNote(context.Background(), "note_001", "tenant_001", "admin_001", ReviewDecisionPayload{
		Decision: "INVALID",
	})
	if err == nil {
		t.Fatal("expected error for invalid decision, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReviewNote_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.ReviewNote(context.Background(), "", "tenant_001", "admin_001", ReviewDecisionPayload{
		Decision: "APPROVED",
	})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReviewNote_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.reviewNoteFn = func(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error {
		return ErrNotFound
	}

	err := h.svc.ReviewNote(context.Background(), "note_999", "tenant_001", "admin_001", ReviewDecisionPayload{
		Decision: "APPROVED",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: Notes — GetNotesByStudentTerm
// ============================================================================

func TestGetNotesByStudentTerm_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []PendingNoteItem{
		{ID: "note_001", StudentID: "stu_001", Description: "Late to class", Status: StatusApproved},
	}

	h.repo.getNotesByStudentTermFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error) {
		if studentID != "stu_001" {
			t.Errorf("expected studentID 'stu_001', got %q", studentID)
		}
		if termID != "term_001" {
			t.Errorf("expected termID 'term_001', got %q", termID)
		}
		return expected, nil
	}

	notes, err := h.svc.GetNotesByStudentTerm(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Description != "Late to class" {
		t.Fatalf("expected description 'Late to class', got %q", notes[0].Description)
	}
}

func TestGetNotesByStudentTerm_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.getNotesByStudentTermFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error) {
		return []PendingNoteItem{}, nil
	}

	notes, err := h.svc.GetNotesByStudentTerm(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(notes))
	}
}

// ============================================================================
// Tests: Notes — GetNote
// ============================================================================

func TestGetNote_HappyPath(t *testing.T) {
	h := newTestHarness()

	now := time.Now()
	h.repo.getNoteByIDFn = func(ctx context.Context, id, tenantID string) (*BehaviorNote, error) {
		if id != "note_001" {
			t.Errorf("expected id 'note_001', got %q", id)
		}
		return &BehaviorNote{
			ID: id, Description: "Test note",
			Status: StatusPendingReview, CreatedAt: now,
		}, nil
	}

	note, err := h.svc.GetNote(context.Background(), "note_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.Description != "Test note" {
		t.Fatalf("expected description 'Test note', got %q", note.Description)
	}
}

func TestGetNote_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getNoteByIDFn = func(ctx context.Context, id, tenantID string) (*BehaviorNote, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetNote(context.Background(), "note_999", "tenant_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

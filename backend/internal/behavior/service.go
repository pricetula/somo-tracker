package behavior

import (
	"context"
	"fmt"
)

// Service handles business logic for behavior operations.
type Service struct {
	repo Repository
}

// NewService creates a new behavior Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ── Categories ────────────────────────────────────────────────────────────

// ListCategories returns all categories for a school.
func (s *Service) ListCategories(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]BehaviorCategory, error) {
	if activeOnly {
		return s.repo.ListActiveCategories(ctx, tenantID, schoolID)
	}
	return s.repo.ListCategories(ctx, tenantID, schoolID)
}

// CreateCategory creates a new behavior category.
func (s *Service) CreateCategory(ctx context.Context, tenantID, schoolID string, payload CreateCategoryPayload) (*BehaviorCategory, error) {
	if payload.Name == "" {
		return nil, fmt.Errorf("behavior.Service.CreateCategory: name is required: %w", ErrInvalidInput)
	}
	if payload.DefaultSeverity != nil {
		valid := *payload.DefaultSeverity == "MINOR" || *payload.DefaultSeverity == "NEEDS_FOLLOW_UP"
		if !valid {
			return nil, fmt.Errorf("behavior.Service.CreateCategory: invalid severity %q: %w", *payload.DefaultSeverity, ErrInvalidInput)
		}
	}
	return s.repo.CreateCategory(ctx, tenantID, schoolID, payload.Name, payload.DefaultSeverity)
}

// UpdateCategory updates a behavior category.
func (s *Service) UpdateCategory(ctx context.Context, id, tenantID string, payload UpdateCategoryPayload) (*BehaviorCategory, error) {
	if id == "" {
		return nil, fmt.Errorf("behavior.Service.UpdateCategory: id is required: %w", ErrInvalidInput)
	}
	return s.repo.UpdateCategory(ctx, id, tenantID, payload)
}

// GetCategory returns a single category by ID.
func (s *Service) GetCategory(ctx context.Context, id, tenantID string) (*BehaviorCategory, error) {
	return s.repo.GetCategoryByID(ctx, id, tenantID)
}

// ── Notes ─────────────────────────────────────────────────────────────────

// CreateNote creates a new behavior note.
func (s *Service) CreateNote(ctx context.Context, tenantID, schoolID string, payload CreateNotePayload, authoredBy string) (*BehaviorNote, error) {
	if payload.StudentID == "" {
		return nil, fmt.Errorf("behavior.Service.CreateNote: student_id is required: %w", ErrInvalidInput)
	}
	if payload.CategoryID == "" {
		return nil, fmt.Errorf("behavior.Service.CreateNote: category_id is required: %w", ErrInvalidInput)
	}
	if payload.Description == "" {
		return nil, fmt.Errorf("behavior.Service.CreateNote: description is required: %w", ErrInvalidInput)
	}
	if payload.TimetableSlotID == "" {
		return nil, fmt.Errorf("behavior.Service.CreateNote: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if payload.Date == "" {
		return nil, fmt.Errorf("behavior.Service.CreateNote: date is required: %w", ErrInvalidInput)
	}
	return s.repo.CreateNote(ctx, tenantID, schoolID, payload, authoredBy)
}

// GetPendingQueue returns all notes pending admin review.
func (s *Service) GetPendingQueue(ctx context.Context, tenantID, schoolID string) (*PendingNotesResponse, error) {
	return s.repo.GetPendingQueue(ctx, tenantID, schoolID)
}

// GetNote returns a single note by ID.
func (s *Service) GetNote(ctx context.Context, id, tenantID string) (*BehaviorNote, error) {
	return s.repo.GetNoteByID(ctx, id, tenantID)
}

// ReviewNote approves or rejects a behavior note.
func (s *Service) ReviewNote(ctx context.Context, id, tenantID, reviewedBy string, decision ReviewDecisionPayload) error {
	if id == "" {
		return fmt.Errorf("behavior.Service.ReviewNote: note id is required: %w", ErrInvalidInput)
	}
	if decision.Decision != "APPROVED" && decision.Decision != "REJECTED" {
		return fmt.Errorf("behavior.Service.ReviewNote: decision must be APPROVED or REJECTED: %w", ErrInvalidInput)
	}
	return s.repo.ReviewNote(ctx, id, tenantID, reviewedBy, decision)
}

// GetNotesByStudentTerm returns approved behavior notes for a student in a term.
func (s *Service) GetNotesByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]PendingNoteItem, error) {
	return s.repo.GetNotesByStudentTerm(ctx, tenantID, schoolID, studentID, termID)
}

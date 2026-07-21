package teacherperformance

import (
	"context"
	"fmt"
)

// Service handles business logic for teacher performance operations.
type Service struct {
	repo Repository
}

// NewService creates a new teacher performance Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RefreshComputation triggers the batch computation of teacher performance
// summaries for all SUBJECT_TEACHER assignments in the given term.
func (s *Service) RefreshComputation(ctx context.Context, termID string) error {
	if termID == "" {
		return fmt.Errorf("teacherperformance.Service.RefreshComputation: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.RefreshComputation(ctx, termID)
}

// ListByTeacher returns performance summaries for a specific teacher in a
// given term, optionally filtered by learning area.
func (s *Service) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string, learningAreaID *string) (*TeacherPerformanceListResponse, error) {
	if tenantID == "" || schoolID == "" || userID == "" || termID == "" {
		return nil, fmt.Errorf("teacherperformance.Service.ListByTeacher: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListByTeacher(ctx, tenantID, schoolID, userID, termID, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("teacherperformance.Service.ListByTeacher: %w", err)
	}
	return &TeacherPerformanceListResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// ListByTerm returns all teacher performance summaries for a given term,
// optionally filtered by class or learning area.
func (s *Service) ListByTerm(ctx context.Context, tenantID, schoolID, termID string, classID, learningAreaID *string) (*TeacherPerformanceListResponse, error) {
	if tenantID == "" || schoolID == "" || termID == "" {
		return nil, fmt.Errorf("teacherperformance.Service.ListByTerm: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListByTerm(ctx, tenantID, schoolID, termID, classID, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("teacherperformance.Service.ListByTerm: %w", err)
	}
	return &TeacherPerformanceListResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// GetByTeacherClassSubject returns a single summary row by its grain.
func (s *Service) GetByTeacherClassSubject(ctx context.Context, userID, learningAreaID, classID, termID string) (*TeacherSubjectPerformanceSummary, error) {
	if userID == "" || learningAreaID == "" || classID == "" || termID == "" {
		return nil, fmt.Errorf("teacherperformance.Service.GetByTeacherClassSubject: %w", ErrInvalidInput)
	}
	return s.repo.GetByTeacherClassSubject(ctx, userID, learningAreaID, classID, termID)
}

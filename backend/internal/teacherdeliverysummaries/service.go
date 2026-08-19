package teacherdeliverysummaries

import (
	"context"
	"fmt"
)

// Service handles business logic for teacher delivery summary operations.
type Service struct {
	repo Repository
}

// NewService creates a new teacher delivery summaries Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RefreshComputation triggers the batch computation of delivery summaries
// for all teachers in the given term.
func (s *Service) RefreshComputation(ctx context.Context, termID string) error {
	if termID == "" {
		return fmt.Errorf("teacherdeliverysummaries.Service.RefreshComputation: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.RefreshComputation(ctx, termID)
}

// ListByTeacher returns delivery summaries for a specific teacher in a given term.
func (s *Service) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string) (*DeliverySummaryListResponse, error) {
	if tenantID == "" || schoolID == "" || userID == "" || termID == "" {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.ListByTeacher: %w", ErrInvalidInput)
	}
	result, err := s.repo.ListByTeacher(ctx, tenantID, schoolID, userID, termID)
	if err != nil {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.ListByTeacher: %w", err)
	}
	return result, nil
}

// ListByTerm returns all delivery summaries for a given term.
func (s *Service) ListByTerm(ctx context.Context, tenantID, schoolID, termID string) (*DeliverySummaryListResponse, error) {
	if tenantID == "" || schoolID == "" || termID == "" {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.ListByTerm: %w", ErrInvalidInput)
	}
	result, err := s.repo.ListByTerm(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.ListByTerm: %w", err)
	}
	return result, nil
}

// GetByTeacherTerm returns a single delivery summary for a teacher in a term.
func (s *Service) GetByTeacherTerm(ctx context.Context, userID, termID string) (*TeacherDeliverySummary, error) {
	if userID == "" || termID == "" {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.GetByTeacherTerm: %w", ErrInvalidInput)
	}
	return s.repo.GetByTeacherTerm(ctx, userID, termID)
}

// ListDeliveryBreakdown returns per-teacher Marked vs. Missed slot counts for
// the School Administrator dashboard grouped bar chart, sorted by missed
// slots descending so chronic non-compliant teachers surface first
// (compliance risk watch).
func (s *Service) ListDeliveryBreakdown(ctx context.Context, tenantID, schoolID, termID string) (*TeacherDeliveryBreakdownListResponse, error) {
	if tenantID == "" || schoolID == "" || termID == "" {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.ListDeliveryBreakdown: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListDeliveryBreakdown(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("teacherdeliverysummaries.Service.ListDeliveryBreakdown: %w", err)
	}
	if items == nil {
		items = []TeacherDeliveryBreakdownItem{}
	}
	return &TeacherDeliveryBreakdownListResponse{Items: items, Total: len(items)}, nil
}

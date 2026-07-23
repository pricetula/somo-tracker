package teacherworkloadsummaries

import (
	"context"
	"fmt"
)

// Service handles business logic for teacher workload summary operations.
type Service struct {
	repo Repository
}

// NewService creates a new teacher workload summaries Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RefreshComputation triggers the batch computation of workload summaries
// for all teachers in the given academic year.
func (s *Service) RefreshComputation(ctx context.Context, academicYearID string) error {
	if academicYearID == "" {
		return fmt.Errorf("teacherworkloadsummaries.Service.RefreshComputation: academic_year_id is required: %w", ErrInvalidInput)
	}
	return s.repo.RefreshComputation(ctx, academicYearID)
}

// ListByTeacher returns workload summaries for a specific teacher in a given year.
func (s *Service) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, yearID string) (*WorkloadSummaryListResponse, error) {
	if tenantID == "" || schoolID == "" || userID == "" || yearID == "" {
		return nil, fmt.Errorf("teacherworkloadsummaries.Service.ListByTeacher: %w", ErrInvalidInput)
	}
	result, err := s.repo.ListByTeacher(ctx, tenantID, schoolID, userID, yearID)
	if err != nil {
		return nil, fmt.Errorf("teacherworkloadsummaries.Service.ListByTeacher: %w", err)
	}
	return result, nil
}

// ListByYear returns all workload summaries for a given academic year.
func (s *Service) ListByYear(ctx context.Context, tenantID, schoolID, yearID string) (*WorkloadSummaryListResponse, error) {
	if tenantID == "" || schoolID == "" || yearID == "" {
		return nil, fmt.Errorf("teacherworkloadsummaries.Service.ListByYear: %w", ErrInvalidInput)
	}
	result, err := s.repo.ListByYear(ctx, tenantID, schoolID, yearID)
	if err != nil {
		return nil, fmt.Errorf("teacherworkloadsummaries.Service.ListByYear: %w", err)
	}
	return result, nil
}

// GetByTeacherYear returns a single workload summary for a teacher in a year.
func (s *Service) GetByTeacherYear(ctx context.Context, userID, yearID string) (*TeacherWorkloadSummary, error) {
	if userID == "" || yearID == "" {
		return nil, fmt.Errorf("teacherworkloadsummaries.Service.GetByTeacherYear: %w", ErrInvalidInput)
	}
	return s.repo.GetByTeacherYear(ctx, userID, yearID)
}

package cbcclasses

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// Service contains business logic for the cbcclasses domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// ListClasses returns a paginated list of classes.
func (s *Service) ListClasses(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
	if filter.TenantID == "" || filter.SchoolID == "" || filter.AcademicYearID == "" || filter.AcademicTermID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.ListClasses: %w", ErrInvalidInput)
	}
	return s.Repo.List(ctx, filter)
}

// CreateClass creates a new class with atomic student enrollment.
func (s *Service) CreateClass(ctx context.Context, params CreateClassParams) (*Class, error) {
	if params.TenantID == "" || params.SchoolID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.CreateClass: %w", ErrInvalidInput)
	}
	if params.GradeLevel == "" || params.AcademicYearID == "" || params.AcademicTermID == "" || params.StreamID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.CreateClass: grade_level, academic_year_id, academic_term_id, and stream_id are required: %w", ErrInvalidInput)
	}

	// Context validation: verify all FK targets exist and belong to this tenant/school
	valid, err := s.Repo.ValidateAcademicYear(ctx, params.AcademicYearID, params.TenantID, params.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.CreateClass: %w", err)
	}
	if !valid {
		return nil, &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"academic_year_id": {"Academic year not found or does not belong to this school"}},
		}
	}

	valid, err = s.Repo.ValidateAcademicTerm(ctx, params.AcademicTermID, params.AcademicYearID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.CreateClass: %w", err)
	}
	if !valid {
		return nil, &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"academic_term_id": {"Academic term not found or does not belong to this academic year"}},
		}
	}

	valid, err = s.Repo.ValidateStream(ctx, params.StreamID, params.TenantID, params.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.CreateClass: %w", err)
	}
	if !valid {
		return nil, &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"stream_id": {"Stream not found or does not belong to this school"}},
		}
	}

	class, err := s.Repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.CreateClass: %w", err)
	}
	return class, nil
}

// UpdateClass updates a class with differential enrollment sync.
func (s *Service) UpdateClass(ctx context.Context, params UpdateClassParams) (*Class, error) {
	if params.ClassID == "" || params.TenantID == "" || params.SchoolID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: %w", ErrInvalidInput)
	}
	if params.GradeLevel == "" || params.StreamID == "" || params.AcademicTermID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: grade_level, stream_id, and academic_term_id are required: %w", ErrInvalidInput)
	}

	// Security: fetch class to verify it belongs to this tenant + school
	existing, err := s.Repo.GetByID(ctx, params.ClassID, params.TenantID, params.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: %w", ErrNotFound)
	}

	// Immutability guard: check for assessment records
	hasAssessments, err := s.Repo.HasAssessmentSessions(ctx, params.ClassID, params.TenantID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: %w", err)
	}
	if hasAssessments {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: class has assessment records: %w", ErrClassLocked)
	}

	class, err := s.Repo.Update(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: %w", err)
	}
	return class, nil
}

// BulkDeleteClasses deletes multiple classes after pre-flight checks.
func (s *Service) BulkDeleteClasses(ctx context.Context, classIDs []string, tenantID, schoolID string) error {
	if len(classIDs) == 0 || tenantID == "" || schoolID == "" {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: %w", ErrInvalidInput)
	}
	if len(classIDs) > 100 {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: max 100 class IDs per request: %w", ErrInvalidInput)
	}

	// Pre-flight: check for assessment sessions
	hasAssessments, err := s.Repo.HasAnyAssessmentSessions(ctx, classIDs, tenantID)
	if err != nil {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: %w", err)
	}
	if hasAssessments {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: one or more classes have assessment records: %w", ErrClassHasAssessments)
	}

	if err := s.Repo.BulkDelete(ctx, classIDs, tenantID, schoolID); err != nil {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: %w", err)
	}
	return nil
}

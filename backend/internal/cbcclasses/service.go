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

	class, err := s.Repo.Update(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.UpdateClass: %w", err)
	}
	return class, nil
}

// GetClass returns a single class by ID, scoped to tenant + school.
func (s *Service) GetClass(ctx context.Context, classID, tenantID, schoolID string) (*Class, error) {
	if classID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.GetClass: %w", ErrInvalidInput)
	}

	class, err := s.Repo.GetByID(ctx, classID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.GetClass: %w", err)
	}
	return class, nil
}

// ─── GetRoster ───────────────────────────────────────────────────────────

// GetRoster returns a paginated roster of students enrolled in a class, with optional search.
func (s *Service) GetRoster(ctx context.Context, classID, tenantID, schoolID, academicTermID string, page, limit int, search string) (*RosterListResult, error) {
	if classID == "" || tenantID == "" || schoolID == "" || academicTermID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.GetRoster: %w", ErrInvalidInput)
	}

	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Verify class exists and belongs to this tenant + school
	_, err := s.Repo.GetByID(ctx, classID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.GetRoster: %w", err)
	}

	result, err := s.Repo.GetRoster(ctx, classID, tenantID, schoolID, academicTermID, limit, offset, search)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.GetRoster: %w", err)
	}
	return result, nil
}

// ─── BatchEnroll ──────────────────────────────────────────────────────────

// BatchEnroll atomically enrolls multiple students into a class for the active term.
// If any student has a concurrent enrollment elsewhere, the entire batch rolls back.
func (s *Service) BatchEnroll(ctx context.Context, classID, tenantID, schoolID, academicTermID string, studentIDs []string) (*BatchEnrollResponse, error) {
	if classID == "" || tenantID == "" || schoolID == "" || academicTermID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.BatchEnroll: %w", ErrInvalidInput)
	}
	if len(studentIDs) == 0 {
		return nil, fmt.Errorf("cbcclasses.Service.BatchEnroll: student_ids is required: %w", ErrInvalidInput)
	}

	// Verify class exists
	_, err := s.Repo.GetByID(ctx, classID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.BatchEnroll: %w", err)
	}

	enrolledCount, err := s.Repo.BatchEnrollStudents(ctx, classID, tenantID, schoolID, academicTermID, studentIDs)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Service.BatchEnroll: %w", err)
	}

	return &BatchEnrollResponse{
		Code:          "enrollment_successful",
		Message:       fmt.Sprintf("%d students successfully enrolled.", enrolledCount),
		EnrolledCount: enrolledCount,
	}, nil
}

// ─── UnenrollStudent ──────────────────────────────────────────────────────

// UnenrollStudent removes a single student from this class.
func (s *Service) UnenrollStudent(ctx context.Context, classID, studentID, tenantID, schoolID string) error {
	if classID == "" || studentID == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("cbcclasses.Service.UnenrollStudent: %w", ErrInvalidInput)
	}

	if err := s.Repo.UnenrollStudent(ctx, classID, studentID, tenantID, schoolID); err != nil {
		return fmt.Errorf("cbcclasses.Service.UnenrollStudent: %w", err)
	}
	return nil
}

// ─── GetAvailableStudents ─────────────────────────────────────────────────

// GetAvailableStudents returns a paginated list of students not enrolled in this class
// for the active term, with optional search.
func (s *Service) GetAvailableStudents(ctx context.Context, filter AvailableStudentsFilter) (*AvailableStudentsResponse, error) {
	if filter.TenantID == "" || filter.SchoolID == "" || filter.ClassID == "" || filter.AcademicTermID == "" {
		return nil, fmt.Errorf("cbcclasses.Service.GetAvailableStudents: %w", ErrInvalidInput)
	}

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	return s.Repo.GetAvailableStudents(ctx, filter)
}

// BulkDeleteClasses deletes multiple classes after pre-flight checks.
func (s *Service) BulkDeleteClasses(ctx context.Context, classIDs []string, tenantID, schoolID string) error {
	if len(classIDs) == 0 || tenantID == "" || schoolID == "" {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: %w", ErrInvalidInput)
	}
	if len(classIDs) > 100 {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: max 100 class IDs per request: %w", ErrInvalidInput)
	}

	if err := s.Repo.BulkDelete(ctx, classIDs, tenantID, schoolID); err != nil {
		return fmt.Errorf("cbcclasses.Service.BulkDeleteClasses: %w", err)
	}
	return nil
}

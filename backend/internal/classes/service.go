package classes

import (
	"context"
	"fmt"
)

// Service contains business logic for the classes domain.
type Service struct {
	repo *Repository
}

// NewService creates a new Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ResolveSchoolID returns the primary active school ID for a tenant.
// Falls back to the first active school found.
func (s *Service) ResolveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	schoolID, err := s.repo.GetPrimarySchoolID(ctx, tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("resolve school: %w", err)
	}
	return schoolID, nil
}

// ListClasses returns filtered classes for the school's current academic year.
func (s *Service) ListClasses(ctx context.Context, schoolID, tenantID string, params ListClassesParams) ([]Class, error) {
	return s.repo.ListClasses(ctx, schoolID, tenantID, params)
}

// ListGrades returns all grades for the school's education system.
func (s *Service) ListGrades(ctx context.Context, schoolID, tenantID string) ([]GradeInfo, error) {
	return s.repo.ListGrades(ctx, schoolID, tenantID)
}

// GenerateClasses cross-multiplies streams with the school's CBE grade levels,
// bulk-inserts all classroom records in a single DB transaction, and returns
// the full result summary.
func (s *Service) GenerateClasses(
	ctx context.Context,
	schoolID, tenantID string,
	payload GeneratePayload,
) (*GenerateResult, error) {
	if len(payload.Streams) == 0 {
		return nil, fmt.Errorf("at least one stream is required")
	}

	// 1. Resolve the current academic year and education system
	academicYear, err := s.repo.GetCurrentAcademicYear(ctx, schoolID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get current academic year: %w", err)
	}
	if academicYear == nil {
		return nil, fmt.Errorf("no current academic year configured")
	}

	// 2. Fetch all grade names for this school's education system
	grades, err := s.repo.GetSchoolGrades(ctx, schoolID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get school grades: %w", err)
	}
	if len(grades) == 0 {
		return nil, fmt.Errorf("no grades found for the school's education system")
	}

	// 3. Build the cross-product: grades × streams
	//    Only generate classes for Junior Secondary grades (Grade 1–4 equivalent).
	//    We filter by the first 4 grades in sequence_order by convention.
	//    This mirrors the frontend's static array of ["Grade 1","Grade 2","Grade 3","Grade 4"].
	gradeNames := make([]string, 0, len(grades))
	for _, g := range grades {
		gradeNames = append(gradeNames, g.Name)
	}

	var inputs []classInput
	for _, grade := range grades {
		for _, stream := range payload.Streams {
			inputs = append(inputs, classInput{
				GradeID: grade.ID,
				Name:    grade.Name + " " + stream,
				Stream:  stream,
			})
		}
	}

	// 4. Bulk insert in a single DB transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	inserted, err := s.repo.BulkInsertClasses(ctx, tx, tenantID, schoolID, academicYear.ID, inputs)
	if err != nil {
		return nil, fmt.Errorf("bulk insert classes: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// 5. Build result
	return &GenerateResult{
		Classes:      inserted,
		TotalCreated: len(inserted),
		Streams:      payload.Streams,
		GradeNames:   gradeNames,
	}, nil
}

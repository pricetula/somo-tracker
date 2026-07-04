package cbcschools

import (
	"context"
	"fmt"
	"log/slog"
)

// Service contains business logic for the cbcschools domain.
type Service struct {
	Repo   Repository
	seeder CurriculumSeeder
}

// NewService creates a new Service. If a CurriculumSeeder is provided (via the
// optional second argument), the service will seed the CBC curriculum
// immediately after a school is created.
func NewService(repo Repository, seeders ...CurriculumSeeder) *Service {
	svc := &Service{Repo: repo}
	if len(seeders) > 0 {
		svc.seeder = seeders[0]
	}
	return svc
}

// CreateSchool creates a new school and returns its ID. If a CurriculumSeeder
// is configured, the CBC curriculum is seeded for the new school automatically.
func (s *Service) CreateSchool(ctx context.Context, tenantID string, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", ErrInvalidInput)
	}

	schoolID, err := s.Repo.Create(ctx, tenantID, name)
	if err != nil {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", err)
	}

	// Seed curriculum automatically for the new school
	if s.seeder != nil {
		if seedErr := s.seeder.SeedForSchool(ctx, tenantID, schoolID); seedErr != nil {
			// Log the seeding failure but do NOT fail school creation — the school
			// was already persisted successfully. The admin can retry seeding later.
			slog.WarnContext(ctx, "cbcschools: curriculum seeding failed for new school",
				slog.String("tenant_id", tenantID),
				slog.String("school_id", schoolID),
				slog.String("error", seedErr.Error()),
			)
		}
	}

	return schoolID, nil
}

// ListSchoolsByTenantID returns all schools for a tenant with member counts
// and whether each school is the user's currently active school.
func (s *Service) ListSchoolsByTenantID(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("cbcschools.Service.ListSchoolsByTenantID: %w", ErrInvalidInput)
	}
	return s.Repo.ListByTenantID(ctx, tenantID, userID)
}

// UpdateSchool applies partial updates to a school.
func (s *Service) UpdateSchool(ctx context.Context, school SchoolUpdateFields) error {
	if school.ID == "" {
		return fmt.Errorf("cbcschools.Service.UpdateSchool: %w", ErrInvalidInput)
	}
	// Ensure at least one field is being updated
	if school.Name == nil && school.County == nil && school.SubCounty == nil &&
		school.Ward == nil && school.KnecSchoolCode == nil && school.NemisCode == nil &&
		school.SchoolType == nil && school.IsActive == nil {
		return fmt.Errorf("cbcschools.Service.UpdateSchool: %w", ErrInvalidInput)
	}
	return s.Repo.Update(ctx, school)
}

// DeleteSchool removes a school by ID.
func (s *Service) DeleteSchool(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("cbcschools.Service.DeleteSchool: %w", ErrInvalidInput)
	}
	return s.Repo.Delete(ctx, id)
}

package cbcschools

import (
	"context"
	"fmt"
	"log/slog"
)

// Service contains business logic for the cbcschools domain.
type Service struct {
	Repo     Repository
	seeder   CurriculumSeeder
	enroller UserSchoolEnroller
}

// ServiceOption is a functional option for configuring the Service.
type ServiceOption func(*Service)

// WithCurriculumSeeder sets the curriculum seeder for automatic seeding.
func WithCurriculumSeeder(seeder CurriculumSeeder) ServiceOption {
	return func(s *Service) {
		s.seeder = seeder
	}
}

// WithUserSchoolEnroller sets the enroller for auto-enrolling the school creator.
func WithUserSchoolEnroller(enroller UserSchoolEnroller) ServiceOption {
	return func(s *Service) {
		s.enroller = enroller
	}
}

// NewService creates a new Service with optional configuration.
func NewService(repo Repository, opts ...ServiceOption) *Service {
	svc := &Service{Repo: repo}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// CreateSchool creates a new school and returns its ID.
// If a creatorUserID is provided, the creator is enrolled as SCHOOL_ADMIN
// and the school is set as their active school. If a CurriculumSeeder is
// configured, the CBC curriculum is seeded automatically.
func (s *Service) CreateSchool(ctx context.Context, tenantID string, name string, creatorUserID ...string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", ErrInvalidInput)
	}

	schoolID, err := s.Repo.Create(ctx, tenantID, name)
	if err != nil {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", err)
	}

	// Enroll the creator as SCHOOL_ADMIN and set as active school
	if len(creatorUserID) > 0 && creatorUserID[0] != "" && s.enroller != nil {
		if enrollErr := s.enroller.CreateMembership(ctx, creatorUserID[0], schoolID, tenantID, "SCHOOL_ADMIN"); enrollErr != nil {
			slog.WarnContext(ctx, "cbcschools: failed to create membership for school creator",
				slog.String("user_id", creatorUserID[0]),
				slog.String("school_id", schoolID),
				slog.String("error", enrollErr.Error()),
			)
		}
		if activeErr := s.enroller.SetActiveSchool(ctx, creatorUserID[0], tenantID, schoolID); activeErr != nil {
			slog.WarnContext(ctx, "cbcschools: failed to set active school for creator",
				slog.String("user_id", creatorUserID[0]),
				slog.String("school_id", schoolID),
				slog.String("error", activeErr.Error()),
			)
		}
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

// SetActiveSchool switches the user's active school to the given school.
// If the user does not yet have a membership in the target school, a
// membership is auto-created using the user's highest role in the tenant.
func (s *Service) SetActiveSchool(ctx context.Context, userID, tenantID, schoolID string) error {
	if schoolID == "" {
		return fmt.Errorf("cbcschools.Service.SetActiveSchool: %w", ErrInvalidInput)
	}

	// Verify the school exists and belongs to the tenant
	school, err := s.Repo.GetByID(ctx, schoolID)
	if err != nil {
		return fmt.Errorf("cbcschools.Service.SetActiveSchool: %w", err)
	}
	if school.TenantID != tenantID {
		return fmt.Errorf("cbcschools.Service.SetActiveSchool: %w", ErrForbidden)
	}

	if s.enroller != nil {
		// Look up the user's role in this tenant so we can create a membership
		// if one doesn't exist yet. This handles the SchoolSwitcher case where
		// the user clicks a school they can see but don't belong to.
		role, roleErr := s.enroller.GetUserRoleInTenant(ctx, userID, tenantID)
		if roleErr != nil {
			return fmt.Errorf("cbcschools.Service.SetActiveSchool: lookup role: %w", roleErr)
		}

		// Upsert membership — safe to call even if one already exists
		// (ON CONFLICT DO UPDATE).
		if enrollErr := s.enroller.CreateMembership(ctx, userID, schoolID, tenantID, role); enrollErr != nil {
			return fmt.Errorf("cbcschools.Service.SetActiveSchool: create membership: %w", enrollErr)
		}

		if activeErr := s.enroller.SetActiveSchool(ctx, userID, tenantID, schoolID); activeErr != nil {
			return fmt.Errorf("cbcschools.Service.SetActiveSchool: %w", activeErr)
		}
	}

	return nil
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

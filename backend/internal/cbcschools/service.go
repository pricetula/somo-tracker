package cbcschools

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Service contains business logic for the cbcschools domain.
type Service struct {
	repo       Repository
	seeder     CurriculumSeeder
	yearSeeder AcademicYearSeeder
	enroller   UserSchoolEnroller
	logger     *zap.SugaredLogger
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

// WithAcademicYearSeeder sets the academic year seeder for automatic
// initial year and term setup when a new school is created.
func WithAcademicYearSeeder(seeder AcademicYearSeeder) ServiceOption {
	return func(s *Service) {
		s.yearSeeder = seeder
	}
}

// NewService creates a new Service with optional configuration.
func NewService(repo Repository, logger *zap.SugaredLogger, opts ...ServiceOption) *Service {
	svc := &Service{repo: repo, logger: logger}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// CreateSchool creates a new school and returns its ID.
// If a creatorUserID is provided, the creator is enrolled with the given role
// and the school is set as their active school. If a CurriculumSeeder is
// configured, the CBC curriculum is seeded automatically.
func (s *Service) CreateSchool(ctx context.Context, tenantID string, name string, role string, creatorUserID ...string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", ErrInvalidInput)
	}
	if role == "" {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: role is required: %w", ErrInvalidInput)
	}

	schoolID, err := s.repo.Create(ctx, tenantID, name)
	if err != nil {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", err)
	}

	// Enroll the creator with the given role and set as active school
	if len(creatorUserID) > 0 && creatorUserID[0] != "" && s.enroller != nil {
		if enrollErr := s.enroller.CreateMembership(ctx, creatorUserID[0], schoolID, tenantID, role); enrollErr != nil {
			s.logger.Warnw("cbcschools: failed to create membership for school creator",
				"user_id", creatorUserID[0],
				"school_id", schoolID,
				"role", role,
				"error", enrollErr.Error(),
			)
		}
		if activeErr := s.enroller.SetActiveSchool(ctx, creatorUserID[0], tenantID, schoolID); activeErr != nil {
			s.logger.Warnw("cbcschools: failed to set active school for creator",
				"user_id", creatorUserID[0],
				"school_id", schoolID,
				"error", activeErr.Error(),
			)
		}
	}

	// Seed curriculum automatically for the new school
	if s.seeder != nil {
		if seedErr := s.seeder.SeedForSchool(ctx, tenantID, schoolID); seedErr != nil {
			// Log the seeding failure but do NOT fail school creation — the school
			// was already persisted successfully. The admin can retry seeding later.
			s.logger.Warnw("cbcschools: curriculum seeding failed for new school",
				"tenant_id", tenantID,
				"school_id", schoolID,
				"error", seedErr.Error(),
			)
		}
	}

	// Set up the initial academic year and CBC terms for the new school.
	// Only runs when a creatorUserID is provided (we need an actor for the
	// audit trail). Seeding failure is logged but does NOT fail school creation
	// — the admin can set up academic years later via the UI.
	if len(creatorUserID) > 0 && creatorUserID[0] != "" && s.yearSeeder != nil {
		if yearErr := s.yearSeeder.SetupInitialYear(ctx, tenantID, schoolID, creatorUserID[0], nil); yearErr != nil {
			s.logger.Warnw("cbcschools: academic year seeding failed for new school",
				"tenant_id", tenantID,
				"school_id", schoolID,
				"error", yearErr.Error(),
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
	return s.repo.ListByTenantID(ctx, tenantID, userID)
}

// SetActiveSchool switches the user's active school to the given school.
// If the user does not yet have a membership in the target school, a
// membership is auto-created using the user's highest role in the tenant.
func (s *Service) SetActiveSchool(ctx context.Context, userID, tenantID, schoolID string) error {
	if schoolID == "" {
		return fmt.Errorf("cbcschools.Service.SetActiveSchool: %w", ErrInvalidInput)
	}

	// Verify the school exists and belongs to the tenant
	school, err := s.repo.GetByID(ctx, schoolID)
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
	return s.repo.Update(ctx, school)
}

// DeleteSchool removes a school by ID.
func (s *Service) DeleteSchool(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("cbcschools.Service.DeleteSchool: %w", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id)
}

// GetSchool retrieves a school by ID and verifies it belongs to the tenant.
// Returns ErrNotFound if the school doesn't exist, ErrForbidden if it belongs
// to a different tenant.
func (s *Service) GetSchool(ctx context.Context, id, tenantID string) (*School, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("cbcschools.Service.GetSchool: %w", ErrInvalidInput)
	}
	school, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("cbcschools.Service.GetSchool: %w", err)
	}
	if school.TenantID != tenantID {
		return nil, fmt.Errorf("cbcschools.Service.GetSchool: %w", ErrForbidden)
	}
	return school, nil
}

// GetSchoolByName retrieves a school within a tenant by its name.
// Returns ErrNotFound when the tenant has no school with that name.
func (s *Service) GetSchoolByName(ctx context.Context, tenantID, name string) (*School, error) {
	if tenantID == "" || name == "" {
		return nil, fmt.Errorf("cbcschools.Service.GetSchoolByName: %w", ErrInvalidInput)
	}
	school, err := s.repo.GetByTenantAndName(ctx, tenantID, name)
	if err != nil {
		return nil, fmt.Errorf("cbcschools.Service.GetSchoolByName: %w", err)
	}
	return school, nil
}

// SeedCurriculum seeds school with default CBE/ CBC learning areas
func (s *Service) SeedCurriculum(ctx context.Context, tenantID string, schoolID string) error {
	// Seed curriculum automatically for the new school
	if s.seeder != nil {
		if seedErr := s.seeder.SeedForSchool(ctx, tenantID, schoolID); seedErr != nil {
			// Log the seeding failure but do NOT fail school creation — the school
			// was already persisted successfully. The admin can retry seeding later.
			s.logger.Warnw("cbcschools: curriculum seeding failed for new school",
				"tenant_id", tenantID,
				"school_id", schoolID,
				"error", seedErr.Error(),
			)
		}
	}

	return nil
}

// OnboardingStatus returns the onboarding status for the given tenant.
func (s *Service) OnboardingStatus(ctx context.Context, tenantID string) (*OnboardingStatus, error) {
	return s.repo.OnboardingStatus(ctx, tenantID)
}

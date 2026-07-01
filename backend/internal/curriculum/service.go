package curriculum

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// validEducationLevels contains the allowed cbc_education_level enum values.
var validEducationLevels = map[string]bool{
	"Early_Years":      true,
	"Upper_Primary":    true,
	"Junior_Secondary": true,
	"Senior_School":    true,
}

// codePattern validates that code is uppercase alphanumeric + underscore.
var codePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Service contains business logic for the curriculum domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// CreateLearningArea creates a new learning area and returns its ID.
func (s *Service) CreateLearningArea(ctx context.Context, params CreateLearningAreaParams) (string, error) {
	if err := validateCreateLearningArea(params); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateLearningArea: %w", err)
	}
	return s.Repo.Create(ctx, params)
}

// GetLearningArea retrieves a single learning area by ID.
func (s *Service) GetLearningArea(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("curriculum.Service.GetLearningArea: %w", ErrInvalidInput)
	}
	return s.Repo.GetByID(ctx, id, tenantID, schoolID)
}

// ListLearningAreas returns all learning areas for the given tenant and school,
// optionally filtered by education_level.
func (s *Service) ListLearningAreas(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("curriculum.Service.ListLearningAreas: %w", ErrInvalidInput)
	}
	return s.Repo.List(ctx, tenantID, schoolID, educationLevel)
}

// UpdateLearningArea applies partial updates to a learning area.
func (s *Service) UpdateLearningArea(ctx context.Context, params UpdateLearningAreaParams) error {
	if params.ID == "" || params.TenantID == "" || params.SchoolID == "" {
		return fmt.Errorf("curriculum.Service.UpdateLearningArea: %w", ErrInvalidInput)
	}
	if params.Name == nil && params.Code == nil && params.EducationLevel == nil {
		return fmt.Errorf("curriculum.Service.UpdateLearningArea: %w", ErrInvalidInput)
	}
	if params.Name != nil {
		if err := validateName(*params.Name); err != nil {
			return fmt.Errorf("curriculum.Service.UpdateLearningArea: %w", err)
		}
	}
	if params.Code != nil {
		if err := validateCode(*params.Code); err != nil {
			return fmt.Errorf("curriculum.Service.UpdateLearningArea: %w", err)
		}
	}
	if params.EducationLevel != nil {
		if err := validateEducationLevel(*params.EducationLevel); err != nil {
			return fmt.Errorf("curriculum.Service.UpdateLearningArea: %w", err)
		}
	}
	return s.Repo.Update(ctx, params)
}

// DeleteLearningArea removes a learning area by ID.
func (s *Service) DeleteLearningArea(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("curriculum.Service.DeleteLearningArea: %w", ErrInvalidInput)
	}
	return s.Repo.Delete(ctx, id, tenantID, schoolID)
}

// ── Validation helpers ────────────────────────────────────────────────────

func validateCreateLearningArea(params CreateLearningAreaParams) error {
	if params.TenantID == "" || params.SchoolID == "" {
		return fmt.Errorf("tenant_id and school_id are required: %w", ErrInvalidInput)
	}
	if err := validateName(params.Name); err != nil {
		return err
	}
	if err := validateCode(params.Code); err != nil {
		return err
	}
	if err := validateEducationLevel(params.EducationLevel); err != nil {
		return err
	}
	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required: %w", ErrInvalidInput)
	}
	if len(name) > 150 {
		return fmt.Errorf("name must not exceed 150 characters: %w", ErrInvalidInput)
	}
	return nil
}

func validateCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("code is required: %w", ErrInvalidInput)
	}
	if len(code) > 50 {
		return fmt.Errorf("code must not exceed 50 characters: %w", ErrInvalidInput)
	}
	if !codePattern.MatchString(code) {
		return fmt.Errorf("code must be uppercase alphanumeric with underscores (e.g. MATH, INT_SCI): %w", ErrInvalidInput)
	}
	return nil
}

func validateEducationLevel(level string) error {
	if level == "" {
		return fmt.Errorf("education_level is required: %w", ErrInvalidInput)
	}
	if !validEducationLevels[level] {
		valid := make([]string, 0, len(validEducationLevels))
		for k := range validEducationLevels {
			valid = append(valid, k)
		}
		return fmt.Errorf(
			"invalid education_level %q; must be one of: %s: %w",
			level, strings.Join(valid, ", "), ErrInvalidInput,
		)
	}
	return nil
}

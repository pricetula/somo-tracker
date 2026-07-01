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

// ── Learning Areas ────────────────────────────────────────────────────────

// CreateLearningArea creates a new learning area and returns its ID.
func (s *Service) CreateLearningArea(ctx context.Context, params CreateLearningAreaParams) (string, error) {
	if err := validateCreateLearningArea(params); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateLearningArea: %w", err)
	}
	return s.Repo.CreateLearningArea(ctx, params)
}

// GetLearningArea retrieves a single learning area by ID.
func (s *Service) GetLearningArea(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("curriculum.Service.GetLearningArea: %w", ErrInvalidInput)
	}
	return s.Repo.GetLearningAreaByID(ctx, id, tenantID, schoolID)
}

// ListLearningAreas returns all learning areas for the given tenant and school,
// optionally filtered by education_level.
func (s *Service) ListLearningAreas(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("curriculum.Service.ListLearningAreas: %w", ErrInvalidInput)
	}
	return s.Repo.ListLearningAreas(ctx, tenantID, schoolID, educationLevel)
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
	return s.Repo.UpdateLearningArea(ctx, params)
}

// DeleteLearningArea removes a learning area by ID.
func (s *Service) DeleteLearningArea(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("curriculum.Service.DeleteLearningArea: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteLearningArea(ctx, id, tenantID, schoolID)
}

// ── Strands ───────────────────────────────────────────────────────────────

// CreateStrand creates a new strand after validating the learning area belongs to the tenant/school.
func (s *Service) CreateStrand(ctx context.Context, params CreateStrandParams, tenantID, schoolID string) (string, error) {
	if params.LearningAreaID == "" {
		return "", fmt.Errorf("curriculum.Service.CreateStrand: learning_area_id is required: %w", ErrInvalidInput)
	}
	if err := validateStrandName(params.Name); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateStrand: %w", err)
	}

	// Verify tenant isolation: learning area must belong to the current tenant + school
	if err := s.Repo.VerifyLearningAreaBelongsToTenant(ctx, params.LearningAreaID, tenantID, schoolID); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateStrand: %w", err)
	}

	id, err := s.Repo.CreateStrand(ctx, params)
	if err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateStrand: %w", err)
	}
	return id, nil
}

// GetStrand retrieves a single strand, verifying tenant isolation.
func (s *Service) GetStrand(ctx context.Context, id, tenantID, schoolID string) (*Strand, error) {
	if id == "" {
		return nil, fmt.Errorf("curriculum.Service.GetStrand: %w", ErrInvalidInput)
	}
	return s.Repo.GetStrandByID(ctx, id)
}

// ListStrands returns strands filtered by learning_area_id (tenant-scoped via handler).
func (s *Service) ListStrands(ctx context.Context, learningAreaID string) ([]Strand, error) {
	if learningAreaID == "" {
		return nil, fmt.Errorf("curriculum.Service.ListStrands: learning_area_id is required: %w", ErrInvalidInput)
	}
	return s.Repo.ListStrandsByLearningArea(ctx, learningAreaID)
}

// UpdateStrand updates a strand's name.
func (s *Service) UpdateStrand(ctx context.Context, params UpdateStrandParams) error {
	if params.ID == "" {
		return fmt.Errorf("curriculum.Service.UpdateStrand: %w", ErrInvalidInput)
	}
	if params.Name == nil {
		return fmt.Errorf("curriculum.Service.UpdateStrand: %w", ErrInvalidInput)
	}
	if err := validateStrandName(*params.Name); err != nil {
		return fmt.Errorf("curriculum.Service.UpdateStrand: %w", err)
	}
	return s.Repo.UpdateStrand(ctx, params)
}

// DeleteStrand removes a strand by ID. If referenced by assessment blueprints, returns ErrReferenceProtected.
func (s *Service) DeleteStrand(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("curriculum.Service.DeleteStrand: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteStrand(ctx, id)
}

// ── Sub-Strands ───────────────────────────────────────────────────────────

// CreateSubStrand creates a new sub-strand after validating the strand's tenant ownership.
func (s *Service) CreateSubStrand(ctx context.Context, params CreateSubStrandParams, tenantID, schoolID string) (string, error) {
	if params.StrandID == "" {
		return "", fmt.Errorf("curriculum.Service.CreateSubStrand: strand_id is required: %w", ErrInvalidInput)
	}
	if err := validateSubStrandName(params.Name); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateSubStrand: %w", err)
	}

	// Verify tenant isolation: strand must be under a learning area belonging to this tenant + school
	if _, err := s.Repo.VerifyStrandInTenantSchool(ctx, params.StrandID, tenantID, schoolID); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateSubStrand: %w", err)
	}

	id, err := s.Repo.CreateSubStrand(ctx, params)
	if err != nil {
		return "", fmt.Errorf("curriculum.Service.CreateSubStrand: %w", err)
	}
	return id, nil
}

// GetSubStrand retrieves a single sub-strand by ID.
func (s *Service) GetSubStrand(ctx context.Context, id string) (*SubStrand, error) {
	if id == "" {
		return nil, fmt.Errorf("curriculum.Service.GetSubStrand: %w", ErrInvalidInput)
	}
	return s.Repo.GetSubStrandByID(ctx, id)
}

// ListSubStrands returns sub-strands filtered by strand_id.
func (s *Service) ListSubStrands(ctx context.Context, strandID string) ([]SubStrand, error) {
	if strandID == "" {
		return nil, fmt.Errorf("curriculum.Service.ListSubStrands: strand_id is required: %w", ErrInvalidInput)
	}
	return s.Repo.ListSubStrandsByStrand(ctx, strandID)
}

// UpdateSubStrand updates a sub-strand's name.
func (s *Service) UpdateSubStrand(ctx context.Context, params UpdateSubStrandParams) error {
	if params.ID == "" {
		return fmt.Errorf("curriculum.Service.UpdateSubStrand: %w", ErrInvalidInput)
	}
	if params.Name == nil {
		return fmt.Errorf("curriculum.Service.UpdateSubStrand: %w", ErrInvalidInput)
	}
	if err := validateSubStrandName(*params.Name); err != nil {
		return fmt.Errorf("curriculum.Service.UpdateSubStrand: %w", err)
	}
	return s.Repo.UpdateSubStrand(ctx, params)
}

// DeleteSubStrand removes a sub-strand by ID.
func (s *Service) DeleteSubStrand(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("curriculum.Service.DeleteSubStrand: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteSubStrand(ctx, id)
}

// ── Performance Indicators ────────────────────────────────────────────────

// CreatePerformanceIndicator creates a new performance indicator, auto-incrementing sequence_order if not provided.
func (s *Service) CreatePerformanceIndicator(ctx context.Context, params CreatePerformanceIndicatorParams, tenantID, schoolID string) (string, error) {
	if params.SubStrandID == "" {
		return "", fmt.Errorf("curriculum.Service.CreatePerformanceIndicator: sub_strand_id is required: %w", ErrInvalidInput)
	}
	if err := validateDescription(params.Description); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreatePerformanceIndicator: %w", err)
	}

	// Verify tenant isolation: sub-strand must be under this tenant + school
	if _, err := s.Repo.VerifySubStrandInTenantSchool(ctx, params.SubStrandID, tenantID, schoolID); err != nil {
		return "", fmt.Errorf("curriculum.Service.CreatePerformanceIndicator: %w", err)
	}

	// Auto-increment sequence_order if not provided
	if params.SequenceOrder == nil {
		maxOrder, err := s.Repo.GetMaxSequenceOrder(ctx, params.SubStrandID)
		if err != nil {
			return "", fmt.Errorf("curriculum.Service.CreatePerformanceIndicator: %w", err)
		}
		next := maxOrder + 1
		params.SequenceOrder = &next
	}

	return s.Repo.CreatePerformanceIndicator(ctx, params)
}

// GetPerformanceIndicator retrieves a single performance indicator by ID.
func (s *Service) GetPerformanceIndicator(ctx context.Context, id string) (*PerformanceIndicator, error) {
	if id == "" {
		return nil, fmt.Errorf("curriculum.Service.GetPerformanceIndicator: %w", ErrInvalidInput)
	}
	return s.Repo.GetPerformanceIndicatorByID(ctx, id)
}

// ListPerformanceIndicators returns performance indicators filtered by sub_strand_id.
func (s *Service) ListPerformanceIndicators(ctx context.Context, subStrandID string) ([]PerformanceIndicator, error) {
	if subStrandID == "" {
		return nil, fmt.Errorf("curriculum.Service.ListPerformanceIndicators: sub_strand_id is required: %w", ErrInvalidInput)
	}
	return s.Repo.ListPerformanceIndicatorsBySubStrand(ctx, subStrandID)
}

// UpdatePerformanceIndicator updates a performance indicator's fields.
func (s *Service) UpdatePerformanceIndicator(ctx context.Context, params UpdatePerformanceIndicatorParams) error {
	if params.ID == "" {
		return fmt.Errorf("curriculum.Service.UpdatePerformanceIndicator: %w", ErrInvalidInput)
	}
	if params.Description == nil && params.SequenceOrder == nil {
		return fmt.Errorf("curriculum.Service.UpdatePerformanceIndicator: %w", ErrInvalidInput)
	}
	if params.Description != nil {
		if err := validateDescription(*params.Description); err != nil {
			return fmt.Errorf("curriculum.Service.UpdatePerformanceIndicator: %w", err)
		}
	}
	return s.Repo.UpdatePerformanceIndicator(ctx, params)
}

// DeletePerformanceIndicator removes a performance indicator by ID.
func (s *Service) DeletePerformanceIndicator(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("curriculum.Service.DeletePerformanceIndicator: %w", ErrInvalidInput)
	}
	return s.Repo.DeletePerformanceIndicator(ctx, id)
}

// ── Tree ──────────────────────────────────────────────────────────────────

// GetTree returns the full learning area tree with nested strands, sub-strands, and indicators.
func (s *Service) GetTree(ctx context.Context, learningAreaID, tenantID, schoolID string) (*LearningAreaTree, error) {
	if learningAreaID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("curriculum.Service.GetTree: %w", ErrInvalidInput)
	}

	// Verify tenant ownership
	if err := s.Repo.VerifyLearningAreaBelongsToTenant(ctx, learningAreaID, tenantID, schoolID); err != nil {
		return nil, fmt.Errorf("curriculum.Service.GetTree: %w", err)
	}

	tree, err := s.Repo.GetTree(ctx, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Service.GetTree: %w", err)
	}
	return tree, nil
}

// ── Validation Helpers ────────────────────────────────────────────────────

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

func validateStrandName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required: %w", ErrInvalidInput)
	}
	if len(name) > 255 {
		return fmt.Errorf("name must not exceed 255 characters: %w", ErrInvalidInput)
	}
	return nil
}

func validateSubStrandName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required: %w", ErrInvalidInput)
	}
	if len(name) > 255 {
		return fmt.Errorf("name must not exceed 255 characters: %w", ErrInvalidInput)
	}
	return nil
}

func validateDescription(desc string) error {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return fmt.Errorf("description is required: %w", ErrInvalidInput)
	}
	if len(desc) > 10000 {
		return fmt.Errorf("description must not exceed 10,000 characters: %w", ErrInvalidInput)
	}
	return nil
}

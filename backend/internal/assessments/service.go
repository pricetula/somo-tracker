package assessments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"somotracker/backend/internal/academicyears"
)

// Service handles business logic for assessment sessions.
type Service struct {
	repo   Repository
	svcAY  *academicyears.Service
	logger *zap.SugaredLogger
}

// NewService creates a new Service.
func NewService(repo Repository, svcAY *academicyears.Service, logger *zap.SugaredLogger) *Service {
	return &Service{repo: repo, svcAY: svcAY, logger: logger}
}

// List returns paginated sessions for a school.
func (s *Service) List(ctx context.Context, filter SessionListFilter) (*SessionListResult, error) {
	return s.repo.List(ctx, filter)
}

// GetByID returns a single session scoped to tenant + school.
func (s *Service) GetByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error) {
	return s.repo.GetByID(ctx, id, tenantID, schoolID)
}

// Create validates and creates a new session in DRAFT status.
func (s *Service) Create(ctx context.Context, payload CreateSessionPayload, tenantID, schoolID, userID string) (*AssessmentSession, error) {
	if err := validateCreate(payload); err != nil {
		return nil, err
	}

	current, err := s.svcAY.GetCurrent(ctx, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.Create: get current academic year/term: %w", err)
	}

	var scheduledDate *time.Time
	if payload.ScheduledDate != nil && *payload.ScheduledDate != "" {
		t, err := time.Parse("2006-01-02", *payload.ScheduledDate)
		if err != nil {
			return nil, fmt.Errorf("%w: scheduled_date must be YYYY-MM-DD", ErrInvalidInput)
		}
		scheduledDate = &t
	}

	params := CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		AcademicYearID:        current.AcademicYearID,
		AcademicTermID:        current.AcademicTermID,
		ClassID:               payload.ClassID,
		LearningAreaID:        payload.LearningAreaID,
		Name:                  strings.TrimSpace(payload.Name),
		EvaluationMethod:      payload.EvaluationMethod,
		MaxPoints:             payload.MaxPoints,
		GradingScaleProfileID: payload.GradingScaleProfileID,
		ScheduledDate:         scheduledDate,
		CreatedBy:             userID,
	}
	return s.repo.Create(ctx, params)
}

// Update modifies a session. Only DRAFT sessions are editable.
func (s *Service) Update(ctx context.Context, payload UpdateSessionPayload, tenantID, schoolID, id string) (*AssessmentSession, error) {
	existing, err := s.repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, err
	}
	if existing.Status != "DRAFT" {
		return nil, fmt.Errorf("assessments.Service.Update: %w: only DRAFT sessions can be edited", ErrInvalidStatus)
	}
	if err := validateUpdate(existing, payload); err != nil {
		return nil, err
	}

	var scheduledDate *time.Time
	if payload.ScheduledDate != nil && *payload.ScheduledDate != "" {
		t, err := time.Parse("2006-01-02", *payload.ScheduledDate)
		if err != nil {
			return nil, fmt.Errorf("%w: scheduled_date must be YYYY-MM-DD", ErrInvalidInput)
		}
		scheduledDate = &t
	}

	params := UpdateSessionParams{
		ID:                    id,
		TenantID:              tenantID,
		SchoolID:              schoolID,
		Name:                  strings.TrimSpace(payload.Name),
		EvaluationMethod:      payload.EvaluationMethod,
		MaxPoints:             payload.MaxPoints,
		GradingScaleProfileID: payload.GradingScaleProfileID,
		ScheduledDate:         scheduledDate,
	}
	return s.repo.Update(ctx, params)
}

// Delete removes a session. Only allowed when DRAFT and no scores exist.
func (s *Service) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	existing, err := s.repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return err
	}
	if existing.Status != "DRAFT" {
		return fmt.Errorf("assessments.Service.Delete: %w: only DRAFT sessions can be deleted", ErrInvalidStatus)
	}
	return s.repo.Delete(ctx, id, tenantID, schoolID)
}

// Submit moves a session into the approval queue.
func (s *Service) Submit(ctx context.Context, id, tenantID, schoolID, userID string) error {
	existing, err := s.repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return err
	}
	if existing.Status != "DRAFT" {
		return fmt.Errorf("assessments.Service.Submit: %w: only DRAFT sessions can be submitted", ErrInvalidStatus)
	}
	return s.repo.Submit(ctx, id, tenantID, schoolID, userID)
}

// Approve publishes a session. DB trigger refreshes rollups.
func (s *Service) Approve(ctx context.Context, id, tenantID, schoolID, userID string) error {
	existing, err := s.repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return err
	}
	if existing.Status != "PENDING_APPROVAL" {
		return fmt.Errorf("assessments.Service.Approve: %w: only PENDING_APPROVAL sessions can be approved", ErrInvalidStatus)
	}
	return s.repo.Approve(ctx, id, tenantID, schoolID, userID)
}

// Reject returns a session to DRAFT with an admin comment.
func (s *Service) Reject(ctx context.Context, id, tenantID, schoolID, userID, comment string) error {
	existing, err := s.repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return err
	}
	if existing.Status != "PENDING_APPROVAL" {
		return fmt.Errorf("assessments.Service.Reject: %w: only PENDING_APPROVAL sessions can be rejected", ErrInvalidStatus)
	}
	if strings.TrimSpace(comment) == "" {
		return fmt.Errorf("%w: comment is required when rejecting", ErrInvalidInput)
	}
	return s.repo.Reject(ctx, id, tenantID, schoolID, userID, strings.TrimSpace(comment))
}

// ─── Validation ──────────────────────────────────────────────────────────

func validateCreate(p CreateSessionPayload) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if p.ClassID == "" {
		return fmt.Errorf("%w: class_id is required", ErrInvalidInput)
	}
	if p.LearningAreaID == "" {
		return fmt.Errorf("%w: learning_area_id is required", ErrInvalidInput)
	}
	switch p.EvaluationMethod {
	case "QUANTITATIVE":
		if p.MaxPoints == nil || *p.MaxPoints <= 0 {
			return fmt.Errorf("%w: max_points is required and must be > 0 for QUANTITATIVE", ErrInvalidInput)
		}
		if p.GradingScaleProfileID == nil || *p.GradingScaleProfileID == "" {
			return fmt.Errorf("%w: grading_scale_profile_id is required for QUANTITATIVE", ErrInvalidInput)
		}
	case "RUBRIC":
		if p.MaxPoints != nil {
			return fmt.Errorf("%w: max_points must be null for RUBRIC", ErrInvalidInput)
		}
		if p.GradingScaleProfileID != nil && *p.GradingScaleProfileID != "" {
			return fmt.Errorf("%w: grading_scale_profile_id must be null for RUBRIC", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: evaluation_method must be QUANTITATIVE or RUBRIC", ErrInvalidInput)
	}
	return nil
}

func validateUpdate(existing *AssessmentSession, p UpdateSessionPayload) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	switch p.EvaluationMethod {
	case "QUANTITATIVE":
		if p.MaxPoints == nil || *p.MaxPoints <= 0 {
			return fmt.Errorf("%w: max_points is required and must be > 0 for QUANTITATIVE", ErrInvalidInput)
		}
		if p.GradingScaleProfileID == nil || *p.GradingScaleProfileID == "" {
			return fmt.Errorf("%w: grading_scale_profile_id is required for QUANTITATIVE", ErrInvalidInput)
		}
	case "RUBRIC":
		if p.MaxPoints != nil {
			return fmt.Errorf("%w: max_points must be null for RUBRIC", ErrInvalidInput)
		}
		if p.GradingScaleProfileID != nil && *p.GradingScaleProfileID != "" {
			return fmt.Errorf("%w: grading_scale_profile_id must be null for RUBRIC", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: evaluation_method must be QUANTITATIVE or RUBRIC", ErrInvalidInput)
	}
	return nil
}

// ─── Score Methods ───────────────────────────────────────────────────────

func (s *Service) UpsertScores(ctx context.Context, sessionID, tenantID string, scores []ScoreEntryPayload) (int, error) {
	session, err := s.repo.GetByID(ctx, sessionID, tenantID, "")
	if err != nil {
		return 0, err
	}
	if session.Status == "PUBLISHED" {
		return 0, fmt.Errorf("%w: scores are read-only after session is PUBLISHED", ErrInvalidStatus)
	}
	return s.repo.UpsertScores(ctx, sessionID, tenantID, scores)
}

func (s *Service) ListScores(ctx context.Context, sessionID, tenantID string, page, limit int) (*ScoreListResult, error) {
	return s.repo.ListScores(ctx, sessionID, tenantID, page, limit)
}
func (s *Service) ListGradingScaleProfiles(ctx context.Context, tenantID, schoolID string) ([]map[string]interface{}, error) {
	return s.repo.ListGradingScaleProfiles(ctx, tenantID, schoolID)
}

func (s *Service) UpsertRubricOutcomes(ctx context.Context, sessionID, tenantID string, entries []RubricEntryPayload) (int, error) {
	session, err := s.repo.GetByID(ctx, sessionID, tenantID, "")
	if err != nil {
		return 0, err
	}
	if session.EvaluationMethod != "RUBRIC" {
		return 0, fmt.Errorf("%w: session is not RUBRIC", ErrInvalidInput)
	}
	if session.Status == "PUBLISHED" {
		return 0, fmt.Errorf("%w: rubric grades are read-only after session is PUBLISHED", ErrInvalidStatus)
	}
	for _, e := range entries {
		switch e.AwardedLevel {
		case "EE", "ME", "AE", "BE":
		default:
			return 0, fmt.Errorf("%w: awarded_level must be EE, ME, AE, or BE", ErrInvalidInput)
		}
	}
	return s.repo.UpsertRubricOutcomes(ctx, sessionID, tenantID, entries)
}

func (s *Service) ListRubricOutcomes(ctx context.Context, sessionID, tenantID string) ([]RubricOutcome, error) {
	return s.repo.ListRubricOutcomes(ctx, sessionID, tenantID)
}

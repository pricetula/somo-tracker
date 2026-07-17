package assessments

import (
	"context"
	"fmt"
	"strings"
)

// allowedTransitions maps current status → allowed next statuses.
var allowedTransitions = map[string]map[string]bool{
	"DRAFT": {
		"PENDING_APPROVAL": true,
	},
	"PENDING_APPROVAL": {
		"DRAFT":     true, // rejection
		"PUBLISHED": true, // approval
	},
	"PUBLISHED": {}, // terminal state
}

// ── Service ──────────────────────────────────────────────────────────────

// Service contains business logic for the assessments domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// ============================================================================
// GRADING SCALE PROFILES
// ============================================================================

// CreateScaleProfile creates a new grading scale profile.
func (s *Service) CreateScaleProfile(ctx context.Context, params CreateScaleProfileParams) (string, error) {
	params.Name = strings.TrimSpace(params.Name)
	if params.TenantID == "" || params.SchoolID == "" {
		return "", fmt.Errorf("assessments.Service.CreateScaleProfile: %w", ErrInvalidInput)
	}
	if params.Name == "" {
		return "", fmt.Errorf("assessments.Service.CreateScaleProfile: name is required: %w", ErrInvalidInput)
	}
	if len(params.Name) > 255 {
		return "", fmt.Errorf("assessments.Service.CreateScaleProfile: name must not exceed 255 characters: %w", ErrInvalidInput)
	}
	return s.Repo.CreateScaleProfile(ctx, params)
}

// GetScaleProfile retrieves a single scale profile by ID.
func (s *Service) GetScaleProfile(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfile, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetScaleProfile: %w", ErrInvalidInput)
	}
	return s.Repo.GetScaleProfileByID(ctx, id, tenantID, schoolID)
}

// ListScaleProfiles returns all scale profiles for a tenant/school.
func (s *Service) ListScaleProfiles(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]ScaleProfile, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.ListScaleProfiles: %w", ErrInvalidInput)
	}
	return s.Repo.ListScaleProfiles(ctx, tenantID, schoolID, activeOnly)
}

// ToggleScaleProfileActive toggles the is_active flag on a profile.
func (s *Service) ToggleScaleProfileActive(ctx context.Context, id, tenantID, schoolID string, isActive bool) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessments.Service.ToggleScaleProfileActive: %w", ErrInvalidInput)
	}
	return s.Repo.ToggleScaleProfileActive(ctx, id, tenantID, schoolID, isActive)
}

// DeleteScaleProfile removes a scale profile.
func (s *Service) DeleteScaleProfile(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessments.Service.DeleteScaleProfile: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteScaleProfile(ctx, id, tenantID, schoolID)
}

// GetScaleProfileWithRanges returns a profile with its ranges.
func (s *Service) GetScaleProfileWithRanges(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfileWithRanges, error) {
	profile, err := s.Repo.GetScaleProfileByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetScaleProfileWithRanges: %w", err)
	}
	ranges, err := s.Repo.GetScaleRangesByProfile(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetScaleProfileWithRanges: get ranges: %w", err)
	}
	result := &ScaleProfileWithRanges{
		ScaleProfile: *profile,
		Ranges:       ranges,
	}
	if result.Ranges == nil {
		result.Ranges = []ScaleRange{}
	}
	return result, nil
}

// ============================================================================
// GRADING SCALE RANGES
// ============================================================================

// BulkSetScaleRanges replaces all ranges for a profile, with validation.
func (s *Service) BulkSetScaleRanges(ctx context.Context, profileID, tenantID, schoolID string, ranges []CreateScaleRangeParams) ([]string, error) {
	// Verify profile exists and belongs to tenant
	if _, err := s.Repo.GetScaleProfileByID(ctx, profileID, tenantID, schoolID); err != nil {
		return nil, fmt.Errorf("assessments.Service.BulkSetScaleRanges: %w", err)
	}

	// Validate ranges
	if len(ranges) == 0 {
		return nil, fmt.Errorf("assessments.Service.BulkSetScaleRanges: at least one range is required: %w", ErrInvalidInput)
	}

	// Validate all four levels are covered
	levelsPresent := make(map[string]bool)
	for _, r := range ranges {
		if r.MinPercentage < 0 || r.MinPercentage > 100 || r.MaxPercentage < 0 || r.MaxPercentage > 100 {
			return nil, fmt.Errorf("assessments.Service.BulkSetScaleRanges: percentages must be between 0 and 100: %w", ErrInvalidInput)
		}
		if r.MaxPercentage <= r.MinPercentage {
			return nil, fmt.Errorf("assessments.Service.BulkSetScaleRanges: max_percentage must be greater than min_percentage: %w", ErrInvalidInput)
		}
		if !IsValidPerformanceLevel(r.PerformanceLevel) {
			return nil, fmt.Errorf("assessments.Service.BulkSetScaleRanges: invalid performance level %q: %w", r.PerformanceLevel, ErrInvalidInput)
		}
		levelsPresent[r.PerformanceLevel] = true
	}

	// Warn but don't block if not all levels are covered — the exclusion
	// constraint handles gaps. But validate at least EE, ME, AE should be present.
	required := []string{"EE", "ME", "AE"}
	for _, l := range required {
		if !levelsPresent[l] {
			return nil, fmt.Errorf("assessments.Service.BulkSetScaleRanges: missing required level %s: %w", l, ErrInvalidInput)
		}
	}

	return s.Repo.BulkSetScaleRanges(ctx, profileID, ranges)
}

// GetScaleRanges returns all ranges for a profile.
func (s *Service) GetScaleRanges(ctx context.Context, profileID, tenantID, schoolID string) ([]ScaleRange, error) {
	if profileID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetScaleRanges: %w", ErrInvalidInput)
	}
	return s.Repo.GetScaleRangesByProfile(ctx, profileID, tenantID, schoolID)
}

// DeleteScaleRange removes a single range from a profile.
func (s *Service) DeleteScaleRange(ctx context.Context, rangeID, profileID, tenantID, schoolID string) error {
	if rangeID == "" || profileID == "" {
		return fmt.Errorf("assessments.Service.DeleteScaleRange: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteScaleRange(ctx, rangeID, profileID, tenantID, schoolID)
}

// ============================================================================
// ASSESSMENT SESSIONS
// ============================================================================

// checkTermNotFinalised returns ErrTermFinalised if the given term is finalised.
func (s *Service) checkTermNotFinalised(ctx context.Context, termID string) error {
	finalised, err := s.Repo.IsTermFinalised(ctx, termID)
	if err != nil {
		return err
	}
	if finalised {
		return fmt.Errorf("assessments.Service.checkTermNotFinalised: %w", ErrTermFinalised)
	}
	return nil
}

// CreateSession creates a new assessment session in DRAFT status.
func (s *Service) CreateSession(ctx context.Context, params CreateSessionParams) (string, error) {
	params.Name = strings.TrimSpace(params.Name)

	if params.TenantID == "" || params.SchoolID == "" {
		return "", fmt.Errorf("assessments.Service.CreateSession: %w", ErrInvalidInput)
	}
	if params.ClassID == "" || params.LearningAreaID == "" || params.AcademicTermID == "" || params.AcademicYearID == "" {
		return "", fmt.Errorf("assessments.Service.CreateSession: class_id, learning_area_id, academic_term_id, and academic_year_id are required: %w", ErrInvalidInput)
	}
	if params.Name == "" {
		return "", fmt.Errorf("assessments.Service.CreateSession: name is required: %w", ErrInvalidInput)
	}
	if params.EvaluationMethod != "QUANTITATIVE" && params.EvaluationMethod != "RUBRIC" {
		return "", fmt.Errorf("assessments.Service.CreateSession: evaluation_method must be QUANTITATIVE or RUBRIC: %w", ErrInvalidInput)
	}
	if params.CreatedBy == "" {
		return "", fmt.Errorf("assessments.Service.CreateSession: created_by is required: %w", ErrInvalidInput)
	}

	// For QUANTITATIVE, require max_points and grading_scale_profile_id
	if params.EvaluationMethod == "QUANTITATIVE" {
		if params.MaxPoints == nil || *params.MaxPoints <= 0 {
			return "", fmt.Errorf("assessments.Service.CreateSession: max_points is required for QUANTITATIVE sessions: %w", ErrInvalidInput)
		}
		if params.GradingScaleProfileID == nil || *params.GradingScaleProfileID == "" {
			return "", fmt.Errorf("assessments.Service.CreateSession: grading_scale_profile_id is required for QUANTITATIVE sessions: %w", ErrInvalidInput)
		}
	}

	// Archival Integrity: cannot create sessions for finalised terms
	if err := s.checkTermNotFinalised(ctx, params.AcademicTermID); err != nil {
		return "", fmt.Errorf("assessments.Service.CreateSession: %w", err)
	}

	return s.Repo.CreateSession(ctx, params)
}

// GetSession retrieves a single assessment session.
func (s *Service) GetSession(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetSession: %w", ErrInvalidInput)
	}
	return s.Repo.GetSessionByID(ctx, id, tenantID, schoolID)
}

// ListSessions returns paginated assessment sessions.
func (s *Service) ListSessions(ctx context.Context, tenantID, schoolID string, filters SessionFilters) ([]AssessmentSession, int, error) {
	if tenantID == "" || schoolID == "" {
		return nil, 0, fmt.Errorf("assessments.Service.ListSessions: %w", ErrInvalidInput)
	}
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 50
	}
	return s.Repo.ListSessions(ctx, tenantID, schoolID, filters)
}

// SubmitSession transitions a session from DRAFT to PENDING_APPROVAL.
func (s *Service) SubmitSession(ctx context.Context, id, tenantID, schoolID, userID string) error {
	if id == "" || tenantID == "" || schoolID == "" || userID == "" {
		return fmt.Errorf("assessments.Service.SubmitSession: %w", ErrInvalidInput)
	}

	session, err := s.Repo.GetSessionByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Service.SubmitSession: %w", err)
	}

	// Archival Integrity: cannot modify sessions for finalised terms
	if err := s.checkTermNotFinalised(ctx, session.AcademicTermID); err != nil {
		return fmt.Errorf("assessments.Service.SubmitSession: %w", err)
	}

	// State transition check
	if !allowedTransitions[session.Status]["PENDING_APPROVAL"] {
		return fmt.Errorf("assessments.Service.SubmitSession: cannot submit session in status %q: %w", session.Status, ErrInvalidStateTransition)
	}

	return s.Repo.UpdateSessionStatus(ctx, id, tenantID, schoolID, "PENDING_APPROVAL", nil, &userID)
}

// ApproveSession transitions a session from PENDING_APPROVAL to PUBLISHED,
// snapshotting performance levels for QUANTITATIVE sessions.
func (s *Service) ApproveSession(ctx context.Context, id, tenantID, schoolID, userID string) error {
	if id == "" || tenantID == "" || schoolID == "" || userID == "" {
		return fmt.Errorf("assessments.Service.ApproveSession: %w", ErrInvalidInput)
	}

	session, err := s.Repo.GetSessionByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Service.ApproveSession: %w", err)
	}

	// Archival Integrity: cannot modify sessions for finalised terms
	if err := s.checkTermNotFinalised(ctx, session.AcademicTermID); err != nil {
		return fmt.Errorf("assessments.Service.ApproveSession: %w", err)
	}

	// State transition check
	if !allowedTransitions[session.Status]["PUBLISHED"] {
		return fmt.Errorf("assessments.Service.ApproveSession: cannot approve session in status %q: %w", session.Status, ErrInvalidStateTransition)
	}

	// For QUANTITATIVE sessions, snapshot performance levels
	if session.EvaluationMethod == "QUANTITATIVE" && session.GradingScaleProfileID != nil {
		profile, err := s.Repo.GetScaleProfileByID(ctx, *session.GradingScaleProfileID, tenantID, schoolID)
		if err != nil {
			return fmt.Errorf("assessments.Service.ApproveSession: get scale profile: %w", err)
		}
		if err := s.Repo.SnapshotPerformanceLevels(ctx, id, profile); err != nil {
			return fmt.Errorf("assessments.Service.ApproveSession: snapshot: %w", err)
		}
	}

	return s.Repo.UpdateSessionStatus(ctx, id, tenantID, schoolID, "PUBLISHED", nil, &userID)
}

// RejectSession transitions a session from PENDING_APPROVAL back to DRAFT.
func (s *Service) RejectSession(ctx context.Context, id, tenantID, schoolID, rejectionComment string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessments.Service.RejectSession: %w", ErrInvalidInput)
	}
	rejectionComment = strings.TrimSpace(rejectionComment)
	if rejectionComment == "" {
		return fmt.Errorf("assessments.Service.RejectSession: rejection_comment is required: %w", ErrInvalidInput)
	}

	session, err := s.Repo.GetSessionByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Service.RejectSession: %w", err)
	}

	// Archival Integrity: cannot modify sessions for finalised terms
	if err := s.checkTermNotFinalised(ctx, session.AcademicTermID); err != nil {
		return fmt.Errorf("assessments.Service.RejectSession: %w", err)
	}

	// State transition check
	if !allowedTransitions[session.Status]["DRAFT"] {
		return fmt.Errorf("assessments.Service.RejectSession: cannot reject session in status %q: %w", session.Status, ErrInvalidStateTransition)
	}

	return s.Repo.UpdateSessionStatus(ctx, id, tenantID, schoolID, "DRAFT", &rejectionComment, nil)
}

// ============================================================================
// STUDENT SCORES (Quantitative)
// ============================================================================

// BulkUpsertScores bulk-upserts quantitative scores for a session.
func (s *Service) BulkUpsertScores(ctx context.Context, params []UpsertScoreParams) error {
	if len(params) == 0 {
		return fmt.Errorf("assessments.Service.BulkUpsertScores: at least one score is required: %w", ErrInvalidInput)
	}

	// Verify session exists and is in DRAFT status
	sessionID := params[0].SessionID
	status, termID, err := s.Repo.GetSessionStatusAndTerm(ctx, sessionID, params[0].TenantID)
	if err != nil {
		return fmt.Errorf("assessments.Service.BulkUpsertScores: %w", err)
	}
	if status != "DRAFT" {
		return fmt.Errorf("assessments.Service.BulkUpsertScores: scores can only be modified in DRAFT status: %w", ErrInvalidStateTransition)
	}

	// Archival Integrity: cannot modify scores for finalised terms
	if err := s.checkTermNotFinalised(ctx, termID); err != nil {
		return fmt.Errorf("assessments.Service.BulkUpsertScores: %w", err)
	}

	// Validate each score
	for _, p := range params {
		if p.StudentID == "" {
			return fmt.Errorf("assessments.Service.BulkUpsertScores: student_id is required: %w", ErrInvalidInput)
		}
		if p.RawScore != nil && *p.RawScore < 0 {
			return fmt.Errorf("assessments.Service.BulkUpsertScores: raw_score cannot be negative: %w", ErrInvalidInput)
		}
		if p.EnrollmentStatus == "ABSENT" || p.EnrollmentStatus == "EXEMPT" {
			return fmt.Errorf("assessments.Service.BulkUpsertScores: student %s is %s, cannot grade: %w", p.StudentID, p.EnrollmentStatus, ErrStudentNotGradable)
		}
	}

	return s.Repo.BulkUpsertStudentScores(ctx, params)
}

// GetStudentScores returns all scores for a session.
func (s *Service) GetStudentScores(ctx context.Context, sessionID, tenantID, schoolID string) ([]StudentScore, error) {
	if sessionID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetStudentScores: %w", ErrInvalidInput)
	}
	return s.Repo.GetStudentScoresBySession(ctx, sessionID, tenantID, schoolID)
}

// ============================================================================
// STUDENT OUTCOME GRADES (Rubric)
// ============================================================================

// BulkUpsertOutcomeGrades bulk-upserts rubric outcome grades for a session.
func (s *Service) BulkUpsertOutcomeGrades(ctx context.Context, params []UpsertOutcomeGradeParams) error {
	if len(params) == 0 {
		return fmt.Errorf("assessments.Service.BulkUpsertOutcomeGrades: at least one grade is required: %w", ErrInvalidInput)
	}

	// Verify session exists and is in DRAFT status
	sessionID := params[0].SessionID
	status, termID, err := s.Repo.GetSessionStatusAndTerm(ctx, sessionID, params[0].TenantID)
	if err != nil {
		return fmt.Errorf("assessments.Service.BulkUpsertOutcomeGrades: %w", err)
	}
	if status != "DRAFT" {
		return fmt.Errorf("assessments.Service.BulkUpsertOutcomeGrades: grades can only be modified in DRAFT status: %w", ErrInvalidStateTransition)
	}

	// Archival Integrity: cannot modify grades for finalised terms
	if err := s.checkTermNotFinalised(ctx, termID); err != nil {
		return fmt.Errorf("assessments.Service.BulkUpsertOutcomeGrades: %w", err)
	}

	// Validate
	for _, p := range params {
		if p.StudentID == "" || p.PerformanceIndicatorID == "" {
			return fmt.Errorf("assessments.Service.BulkUpsertOutcomeGrades: student_id and performance_indicator_id are required: %w", ErrInvalidInput)
		}
		if !IsValidPerformanceLevel(p.AwardedLevel) {
			return fmt.Errorf("assessments.Service.BulkUpsertOutcomeGrades: invalid performance level %q: %w", p.AwardedLevel, ErrInvalidInput)
		}
	}

	return s.Repo.BulkUpsertOutcomeGrades(ctx, params)
}

// GetOutcomeGrades returns all outcome grades for a session.
func (s *Service) GetOutcomeGrades(ctx context.Context, sessionID, tenantID, schoolID string) ([]OutcomeGrade, error) {
	if sessionID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetOutcomeGrades: %w", ErrInvalidInput)
	}
	return s.Repo.GetOutcomeGradesBySession(ctx, sessionID, tenantID, schoolID)
}

// ============================================================================
// PARENT VIEW & REPORT CARDS
// ============================================================================

// GetParentAssessments returns all published assessments for a student in a term.
func (s *Service) GetParentAssessments(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]ParentAssessmentView, error) {
	if tenantID == "" || schoolID == "" || studentID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetParentAssessments: %w", ErrInvalidInput)
	}

	views, err := s.Repo.GetPublishedSessionsForParent(ctx, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetParentAssessments: %w", err)
	}

	// Enrich rubric sessions with outcome grades
	for i, v := range views {
		if v.EvaluationMethod == "RUBRIC" {
			grades, err := s.Repo.GetOutcomeGradesByStudent(ctx, v.SessionID, studentID)
			if err != nil {
				return nil, fmt.Errorf("assessments.Service.GetParentAssessments: get outcome grades: %w", err)
			}
			views[i].OutcomeGrades = grades
		}
	}

	if views == nil {
		views = []ParentAssessmentView{}
	}
	return views, nil
}

// GetStudentTermGrades compiles the final term grades using the "Last One" chronological mode.
func (s *Service) GetStudentTermGrades(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
	if tenantID == "" || schoolID == "" || studentID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetStudentTermGrades: %w", ErrInvalidInput)
	}

	grades, err := s.Repo.GetStudentTermGrades(ctx, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetStudentTermGrades: %w", err)
	}

	if grades == nil {
		grades = []StudentTermGrade{}
	}
	return grades, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ASSESSMENT WEIGHT CONFIGS
// ═══════════════════════════════════════════════════════════════════════════

// CreateWeightConfig creates a new weight config (system-level).
func (s *Service) CreateWeightConfig(ctx context.Context, params CreateWeightConfigParams) (string, error) {
	params.AssessmentTypeCode = strings.TrimSpace(params.AssessmentTypeCode)
	params.TargetExam = strings.TrimSpace(params.TargetExam)

	if params.GradeLevel == "" {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: grade_level is required: %w", ErrInvalidInput)
	}
	if params.AssessmentTypeCode == "" {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: assessment_type_code is required: %w", ErrInvalidInput)
	}
	if params.TargetExam == "" {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: target_exam is required: %w", ErrInvalidInput)
	}
	if params.WeightPercent <= 0 || params.WeightPercent > 100 {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: weight_percent must be between 0 and 100: %w", ErrInvalidInput)
	}
	if params.EffectiveFrom < 2024 || params.EffectiveFrom > 2100 {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: effective_from must be a valid academic year: %w", ErrInvalidInput)
	}
	if len(params.AssessmentTypeCode) > 50 {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: assessment_type_code must not exceed 50 characters: %w", ErrInvalidInput)
	}
	if len(params.TargetExam) > 20 {
		return "", fmt.Errorf("assessments.Service.CreateWeightConfig: target_exam must not exceed 20 characters: %w", ErrInvalidInput)
	}

	return s.Repo.CreateWeightConfig(ctx, params)
}

// ListWeightConfigs returns assessment weight configs.
func (s *Service) ListWeightConfigs(ctx context.Context, filter AssessmentWeightConfigFilter) (*ListWeightConfigsResponse, error) {
	items, err := s.Repo.ListWeightConfigs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.ListWeightConfigs: %w", err)
	}
	if items == nil {
		items = []AssessmentWeightConfig{}
	}
	return &ListWeightConfigsResponse{
		Items: items,
		Total: len(items),
		Page:  1,
		Limit: len(items),
	}, nil
}

// GetWeightConfigByID returns a single weight config.
func (s *Service) GetWeightConfigByID(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
	if id == "" {
		return nil, fmt.Errorf("assessments.Service.GetWeightConfigByID: id is required: %w", ErrInvalidInput)
	}
	return s.Repo.GetWeightConfigByID(ctx, id)
}

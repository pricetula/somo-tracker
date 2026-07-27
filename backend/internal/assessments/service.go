package assessments

import (
	"context"
	"fmt"
	"net/url"
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
	Repo           Repository
	RosterProvider RosterProvider
	Enqueuer       *Enqueuer
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// SetRosterProvider sets the roster provider for cross-domain roster lookups.
func (s *Service) SetRosterProvider(rp RosterProvider) {
	s.RosterProvider = rp
}

// SetEnqueuer sets the background task enqueuer for cascading summary refreshes.
func (s *Service) SetEnqueuer(e *Enqueuer) {
	s.Enqueuer = e
}

// ============================================================================
// GRADING SCALE PROFILES
// ============================================================================

// CreateScaleProfile creates a new grading scale profile with its ranges.
// The profile and all its percentage-to-level ranges are created in a single
// atomic transaction. At minimum EE, ME, and AE ranges must be provided.
func (s *Service) CreateScaleProfile(ctx context.Context, params CreateScaleProfileParams) (string, []string, error) {
	params.Name = strings.TrimSpace(params.Name)
	if params.TenantID == "" || params.SchoolID == "" {
		return "", nil, fmt.Errorf("assessments.Service.CreateScaleProfile: %w", ErrInvalidInput)
	}
	if params.Name == "" {
		return "", nil, fmt.Errorf("assessments.Service.CreateScaleProfile: name is required: %w", ErrInvalidInput)
	}
	if len(params.Name) > 255 {
		return "", nil, fmt.Errorf("assessments.Service.CreateScaleProfile: name must not exceed 255 characters: %w", ErrInvalidInput)
	}
	if len(params.Ranges) == 0 {
		return "", nil, fmt.Errorf("assessments.Service.CreateScaleProfile: grading scale profiles must include at least one range: %w", ErrInvalidInput)
	}

	if err := s.validateScaleRanges(params.Ranges); err != nil {
		return "", nil, fmt.Errorf("assessments.Service.CreateScaleProfile: %w", err)
	}

	return s.Repo.CreateScaleProfileWithRanges(ctx, params)
}

// validateScaleRanges checks that ranges are valid: coverage of required levels,
// valid percentages, and non-overlapping bounds.
func (s *Service) validateScaleRanges(ranges []CreateScaleRangeParams) error {
	if len(ranges) == 0 {
		return fmt.Errorf("at least one range is required: %w", ErrInvalidInput)
	}

	levelsPresent := make(map[string]bool)
	for _, r := range ranges {
		if r.MinPercentage < 0 || r.MinPercentage > 100 || r.MaxPercentage < 0 || r.MaxPercentage > 100 {
			return fmt.Errorf("percentages must be between 0 and 100: %w", ErrInvalidInput)
		}
		if r.MaxPercentage <= r.MinPercentage {
			return fmt.Errorf("max_percentage must be greater than min_percentage: %w", ErrInvalidInput)
		}
		if !IsValidPerformanceLevel(r.PerformanceLevel) {
			return fmt.Errorf("invalid performance level %q: %w", r.PerformanceLevel, ErrInvalidInput)
		}
		levelsPresent[r.PerformanceLevel] = true
	}

	required := []string{"EE", "ME", "AE"}
	for _, l := range required {
		if !levelsPresent[l] {
			return fmt.Errorf("missing required level %s: %w", l, ErrInvalidInput)
		}
	}

	return nil
}

// GetScaleProfile retrieves a single scale profile by ID, including its ranges.
func (s *Service) GetScaleProfile(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfileWithRanges, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetScaleProfile: %w", ErrInvalidInput)
	}
	profile, err := s.Repo.GetScaleProfileByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, err
	}
	if profile.Ranges == nil {
		profile.Ranges = []ScaleRange{}
	}
	return profile, nil
}

// ListScaleProfiles returns all scale profiles for a tenant/school, each with its ranges.
func (s *Service) ListScaleProfiles(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]ScaleProfileWithRanges, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.ListScaleProfiles: %w", ErrInvalidInput)
	}
	profiles, err := s.Repo.ListScaleProfiles(ctx, tenantID, schoolID, activeOnly)
	if err != nil {
		return nil, err
	}
	if profiles == nil {
		profiles = []ScaleProfileWithRanges{}
	}
	return profiles, nil
}

// GetScaleRanges returns all ranges for a profile.
func (s *Service) GetScaleRanges(ctx context.Context, profileID string) ([]ScaleRange, error) {
	if profileID == "" {
		return nil, fmt.Errorf("assessments.Service.GetScaleRanges: %w", ErrInvalidInput)
	}
	ranges, err := s.Repo.GetScaleRanges(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if ranges == nil {
		ranges = []ScaleRange{}
	}
	return ranges, nil
}

// ReplaceScaleRanges replaces all ranges for a profile (delete + insert).
func (s *Service) ReplaceScaleRanges(ctx context.Context, profileID string, ranges []CreateScaleRangeParams) ([]string, error) {
	if profileID == "" {
		return nil, fmt.Errorf("assessments.Service.ReplaceScaleRanges: %w", ErrInvalidInput)
	}
	if len(ranges) == 0 {
		return nil, fmt.Errorf("assessments.Service.ReplaceScaleRanges: at least one range is required: %w", ErrInvalidInput)
	}
	if err := s.validateScaleRanges(ranges); err != nil {
		return nil, fmt.Errorf("assessments.Service.ReplaceScaleRanges: %w", err)
	}
	return s.Repo.ReplaceScaleRanges(ctx, profileID, ranges)
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

// GetSession retrieves a single assessment session, including an
// enroll_students_url derived from the session's class_id and academic_term_id.
func (s *Service) GetSession(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetSession: %w", ErrInvalidInput)
	}
	session, err := s.Repo.GetSessionByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, err
	}

	session.EnrollStudentsURL = buildEnrollStudentsURL(session.ClassID, session.AcademicYearID, session.AcademicTermID)

	return session, nil
}

// buildEnrollStudentsURL constructs the frontend route for enrolling students
// into a class for a specific academic term.
// Route: /classes/{class_id}/enroll?academictermid={academic_term_id}
func buildEnrollStudentsURL(classID, academicYearID, academicTermID string) *string {
	if classID == "" || academicTermID == "" {
		return nil
	}
	u := url.Values{}
	u.Set("academictermid", academicTermID)
	path := "/classes/" + classID + "/enroll?" + u.Encode()
	return &path
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
		if err := s.Repo.SnapshotPerformanceLevels(ctx, id, &profile.ScaleProfile); err != nil {
			return fmt.Errorf("assessments.Service.ApproveSession: snapshot: %w", err)
		}
	}

	// Update session status to PUBLISHED
	if err := s.Repo.UpdateSessionStatus(ctx, id, tenantID, schoolID, "PUBLISHED", nil, &userID); err != nil {
		return fmt.Errorf("assessments.Service.ApproveSession: %w", err)
	}

	// Refresh student_term_subject_summaries for all students in this session.
	// The DB trigger (trg_assessment_sessions_refresh_summary) also runs, but
	// calling it explicitly ensures the Go layer can depend on it without
	// relying solely on trigger semantics.
	if err := s.Repo.RefreshSessionSummary(ctx, id); err != nil {
		return fmt.Errorf("assessments.Service.ApproveSession: refresh summary: %w", err)
	}

	// ── Background cascading refreshes (best-effort) ────────────────────
	// These run asynchronously via Asynq workers so the HTTP response is
	// not blocked by potentially heavy batch computations.
	if s.Enqueuer != nil {
		termID := session.AcademicTermID
		s.Enqueuer.EnqueueOverallSummaryRefresh(ctx, termID)
		s.Enqueuer.EnqueueProjectionsRefresh(ctx, termID)
		s.Enqueuer.EnqueueTeacherPerformanceRefresh(ctx, termID)
		s.Enqueuer.EnqueueCohortPositionsRefresh(ctx, termID)
	}

	return nil
}

// DeleteSession hard-deletes a DRAFT assessment session and its scores/grades.
func (s *Service) DeleteSession(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessments.Service.DeleteSession: %w", ErrInvalidInput)
	}

	// Only allow deleting DRAFT sessions
	session, err := s.Repo.GetSessionByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessments.Service.DeleteSession: %w", err)
	}
	if session.Status != "DRAFT" {
		return fmt.Errorf("assessments.Service.DeleteSession: only DRAFT sessions can be deleted, session is %q: %w", session.Status, ErrInvalidInput)
	}

	return s.Repo.DeleteSession(ctx, id, tenantID, schoolID)
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
// STUDENT TERM SUBJECT SUMMARIES
// ═══════════════════════════════════════════════════════════════════════════

// RefreshSessionSummary triggers a recomputation of student_term_subject_summaries
// for all students in the given session. This is typically called automatically
// when a session transitions to PUBLISHED, but can also be invoked manually.
func (s *Service) RefreshSessionSummary(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("assessments.Service.RefreshSessionSummary: %w", ErrInvalidInput)
	}
	return s.Repo.RefreshSessionSummary(ctx, sessionID)
}

// GetStudentTermSubjectSummaries returns the blended summaries for a student
// across all learning areas in a given term.
func (s *Service) GetStudentTermSubjectSummaries(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermSubjectSummary, error) {
	if tenantID == "" || schoolID == "" || studentID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetStudentTermSubjectSummaries: %w", ErrInvalidInput)
	}

	summaries, err := s.Repo.GetStudentTermSubjectSummaries(ctx, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetStudentTermSubjectSummaries: %w", err)
	}
	return summaries, nil
}

// GetLearningAreaSummaries returns summaries for all students in a learning area
// for a given term (e.g. teacher dashboard showing all students in Mathematics).
func (s *Service) GetLearningAreaSummaries(ctx context.Context, tenantID, schoolID, termID, learningAreaID string) ([]StudentTermSubjectSummary, error) {
	if tenantID == "" || schoolID == "" || termID == "" || learningAreaID == "" {
		return nil, fmt.Errorf("assessments.Service.GetLearningAreaSummaries: %w", ErrInvalidInput)
	}

	summaries, err := s.Repo.GetLearningAreaSummaries(ctx, tenantID, schoolID, termID, learningAreaID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetLearningAreaSummaries: %w", err)
	}
	return summaries, nil
}

// SetTeacherRemark updates the teacher_remark on a summary row.
func (s *Service) SetTeacherRemark(ctx context.Context, summaryID, tenantID, schoolID string, remark *string) error {
	if summaryID == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessments.Service.SetTeacherRemark: %w", ErrInvalidInput)
	}
	return s.Repo.SetTeacherRemark(ctx, summaryID, tenantID, schoolID, remark)
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

// DeleteWeightConfig hard-deletes a weight config. SYSTEM_ADMIN only.
func (s *Service) DeleteWeightConfig(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("assessments.Service.DeleteWeightConfig: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteWeightConfig(ctx, id)
}

// GetWeightConfigByID returns a single weight config.
func (s *Service) GetWeightConfigByID(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
	if id == "" {
		return nil, fmt.Errorf("assessments.Service.GetWeightConfigByID: id is required: %w", ErrInvalidInput)
	}
	return s.Repo.GetWeightConfigByID(ctx, id)
}

// ═══════════════════════════════════════════════════════════════════════════
// STUDENT TERM OVERALL SUMMARIES
// ═══════════════════════════════════════════════════════════════════════════

// RefreshTermOverallSummaries triggers computation of overall summaries
// for ALL students in the given term.
func (s *Service) RefreshTermOverallSummaries(ctx context.Context, termID string) error {
	if termID == "" {
		return fmt.Errorf("assessments.Service.RefreshTermOverallSummaries: %w", ErrInvalidInput)
	}
	return s.Repo.RefreshTermOverallSummaries(ctx, termID)
}

// RefreshSingleStudentOverallSummary triggers computation for one student+term.
func (s *Service) RefreshSingleStudentOverallSummary(ctx context.Context, studentID, termID string) error {
	if studentID == "" || termID == "" {
		return fmt.Errorf("assessments.Service.RefreshSingleStudentOverallSummary: %w", ErrInvalidInput)
	}
	return s.Repo.RefreshSingleStudentOverallSummary(ctx, studentID, termID)
}

// GetStudentTermOverallSummary returns the overall summary for a single
// student+term. Returns a pointer — nil when not found.
func (s *Service) GetStudentTermOverallSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*StudentTermOverallSummary, error) {
	if tenantID == "" || schoolID == "" || studentID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetStudentTermOverallSummary: %w", ErrInvalidInput)
	}
	return s.Repo.GetStudentTermOverallSummary(ctx, tenantID, schoolID, studentID, termID)
}

// ListStudentTermOverallSummaries returns overall summaries for all students
// in the given term.
func (s *Service) ListStudentTermOverallSummaries(ctx context.Context, tenantID, schoolID, termID string) ([]StudentTermOverallSummary, error) {
	if tenantID == "" || schoolID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.ListStudentTermOverallSummaries: %w", ErrInvalidInput)
	}
	items, err := s.Repo.ListStudentTermOverallSummaries(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.ListStudentTermOverallSummaries: %w", err)
	}
	return items, nil
}

// SetHeadteacherRemark updates the headteacher_remark on an overall summary.
func (s *Service) SetHeadteacherRemark(ctx context.Context, summaryID, tenantID, schoolID string, remark *string) error {
	if summaryID == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessments.Service.SetHeadteacherRemark: %w", ErrInvalidInput)
	}
	return s.Repo.SetHeadteacherRemark(ctx, summaryID, tenantID, schoolID, remark)
}

// ============================================================================
// GRADING DATA (merged roster + scores/grades)
// ============================================================================

// ═══════════════════════════════════════════════════════════════════════════
// STUDENT SUBJECT STRAND SUMMARIES
// ═══════════════════════════════════════════════════════════════════════════

// RefreshSubjectStrandSummaries triggers a refresh of sub-strand summaries
// for all students in the given session. Only affects RUBRIC sessions.
func (s *Service) RefreshSubjectStrandSummaries(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("assessments.Service.RefreshSubjectStrandSummaries: %w", ErrInvalidInput)
	}
	return s.Repo.RefreshSubjectStrandSummaries(ctx, sessionID)
}

// GetStudentSubjectStrandSummaries returns all sub-strand summaries for a
// specific student in a specific term.
func (s *Service) GetStudentSubjectStrandSummaries(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentSubjectStrandSummary, error) {
	if tenantID == "" || schoolID == "" || studentID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetStudentSubjectStrandSummaries: %w", ErrInvalidInput)
	}
	items, err := s.Repo.GetStudentSubjectStrandSummaries(ctx, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetStudentSubjectStrandSummaries: %w", err)
	}
	return items, nil
}

// GetSubjectStrandSummariesByTerm returns all sub-strand summaries for all
// students in a given term (used for term-end batch reports).
func (s *Service) GetSubjectStrandSummariesByTerm(ctx context.Context, tenantID, schoolID, termID string) ([]StudentSubjectStrandSummary, error) {
	if tenantID == "" || schoolID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetSubjectStrandSummariesByTerm: %w", ErrInvalidInput)
	}
	items, err := s.Repo.GetSubjectStrandSummariesByTerm(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetSubjectStrandSummariesByTerm: %w", err)
	}
	return items, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// STUDENT PERFORMANCE PROJECTIONS
// ═══════════════════════════════════════════════════════════════════════════

// RefreshProjections triggers a batch computation of performance projections
// for all students in the given academic term.
func (s *Service) RefreshProjections(ctx context.Context, termID string) error {
	if termID == "" {
		return fmt.Errorf("assessments.Service.RefreshProjections: %w", ErrInvalidInput)
	}
	return s.Repo.RefreshProjections(ctx, termID)
}

// GetStudentProjection returns the performance projection for a specific
// student+term, optionally scoped to a learning area.
func (s *Service) GetStudentProjection(ctx context.Context, tenantID, schoolID, studentID, termID string, learningAreaID *string) (*StudentPerformanceProjection, error) {
	if tenantID == "" || schoolID == "" || studentID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.GetStudentProjection: %w", ErrInvalidInput)
	}
	return s.Repo.GetStudentProjection(ctx, tenantID, schoolID, studentID, termID, learningAreaID)
}

// ListStudentProjections returns all performance projections for a given term.
func (s *Service) ListStudentProjections(ctx context.Context, tenantID, schoolID, termID string) ([]StudentPerformanceProjection, error) {
	if tenantID == "" || schoolID == "" || termID == "" {
		return nil, fmt.Errorf("assessments.Service.ListStudentProjections: %w", ErrInvalidInput)
	}
	items, err := s.Repo.ListStudentProjections(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.ListStudentProjections: %w", err)
	}
	return items, nil
}

// ============================================================================
// GRADING DATA (merged roster + scores/grades)
// ============================================================================

// GetGradingData returns the session, roster, and merged scores/grades in a
// single response. The backend resolves the roster from the session's class_id
// and academic_term_id, so the frontend doesn't need to pass them separately.
func (s *Service) GetGradingData(ctx context.Context, sessionID, tenantID, schoolID string) (*GradingDataResponse, error) {
	if sessionID == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessments.Service.GetGradingData: %w", ErrInvalidInput)
	}

	// 1. Load the session to get class_id and academic_term_id
	session, err := s.Repo.GetSessionByID(ctx, sessionID, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetGradingData: %w", err)
	}

	// 2. Load the roster from the class domain
	roster, err := s.RosterProvider.GetRosterByClassAndTerm(ctx, session.ClassID, tenantID, schoolID, session.AcademicTermID)
	if err != nil {
		return nil, fmt.Errorf("assessments.Service.GetGradingData: %w", err)
	}

	// 3. Build merged student list
	students := make([]GradingDataStudent, 0, len(roster))

	if session.EvaluationMethod == "QUANTITATIVE" {
		// Load scores and merge by student_id
		scores, err := s.Repo.GetStudentScoresBySession(ctx, sessionID, tenantID, schoolID)
		if err != nil {
			return nil, fmt.Errorf("assessments.Service.GetGradingData: %w", err)
		}

		scoreByStudent := make(map[string]*StudentScore, len(scores))
		for i := range scores {
			scoreByStudent[scores[i].StudentID] = &scores[i]
		}

		for _, r := range roster {
			s := GradingDataStudent{
				StudentID:        r.StudentID,
				StudentName:      r.StudentName,
				AdmissionNumber:  r.AdmissionNumber,
				Gender:           r.Gender,
				EnrollmentStatus: "ACTIVE",
			}
			if sc, ok := scoreByStudent[r.StudentID]; ok {
				s.Score = sc
				if sc.EnrollmentStatus != "" {
					s.EnrollmentStatus = sc.EnrollmentStatus
				}
			}
			students = append(students, s)
		}
	} else {
		// RUBRIC: load outcome grades
		grades, err := s.Repo.GetOutcomeGradesBySession(ctx, sessionID, tenantID, schoolID)
		if err != nil {
			return nil, fmt.Errorf("assessments.Service.GetGradingData: %w", err)
		}

		gradesByStudent := make(map[string][]OutcomeGrade, len(grades))
		for _, g := range grades {
			gradesByStudent[g.StudentID] = append(gradesByStudent[g.StudentID], g)
		}

		for _, r := range roster {
			s := GradingDataStudent{
				StudentID:        r.StudentID,
				StudentName:      r.StudentName,
				AdmissionNumber:  r.AdmissionNumber,
				Gender:           r.Gender,
				EnrollmentStatus: "ACTIVE",
			}
			if gs, ok := gradesByStudent[r.StudentID]; ok {
				s.Grades = gs
			}
			students = append(students, s)
		}
	}

	if students == nil {
		students = []GradingDataStudent{}
	}

	return &GradingDataResponse{
		Session:  session,
		Students: students,
	}, nil
}

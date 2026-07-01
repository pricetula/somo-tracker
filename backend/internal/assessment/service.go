package assessment

import (
	"context"
	"fmt"
	"log/slog"
)

// gradeLevelToEducationLevel maps cbc_grade_level values to their
// corresponding cbc_education_level. This is used to validate that
// performance indicators linked to a blueprint belong to a learning area
// whose education level matches the blueprint's grade level.
var gradeLevelToEducationLevel = map[string]string{
	"PP1": "Early_Years", "PP2": "Early_Years",
	"G1": "Early_Years", "G2": "Early_Years", "G3": "Early_Years",
	"G4": "Upper_Primary", "G5": "Upper_Primary", "G6": "Upper_Primary",
	"G7": "Junior_Secondary", "G8": "Junior_Secondary", "G9": "Junior_Secondary",
	"G10": "Senior_School", "G11": "Senior_School", "G12": "Senior_School",
}

// validAssessmentTypes is the set of valid cbc_assessment_type values.
var validAssessmentTypes = map[string]bool{
	"Formative_Classroom":     true,
	"KNEC_Written_Assessment": true,
	"KNEC_SBA_Project":        true,
	"National_KPSEA":          true,
	"National_KJSEA":          true,
	"National_KSSEA":          true,
}

// ============================================================================
// Service
// ============================================================================

// validRubricLevels is the set of valid KNEC 4-level rubric outcomes.
var validRubricLevels = map[string]bool{
	"EE": true,
	"ME": true,
	"AE": true,
	"BE": true,
}

// validScoreTypes is the set of valid lrr_score_type values.
var validScoreTypes = map[string]bool{
	"Numeric_Raw":   true,
	"Rubric_Direct": true,
}

// Service contains business logic for assessment blueprints and weight configs.
type Service struct {
	Repo          Repository
	LearningAreas LearningAreaResolver
	ClassStudents ClassStudentResolver
}

// NewService creates a new Service.
func NewService(repo Repository, laResolver LearningAreaResolver, csResolver ClassStudentResolver) *Service {
	return &Service{
		Repo:          repo,
		LearningAreas: laResolver,
		ClassStudents: csResolver,
	}
}

// ============================================================================
// BLUEPRINTS
// ============================================================================

// CreateBlueprint validates and creates a new assessment blueprint.
func (s *Service) CreateBlueprint(ctx context.Context, tenantID, schoolID string, payload CreateBlueprintPayload) (*AssessmentBlueprint, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrInvalidInput)
	}

	// Validate title
	if payload.Title == "" {
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrInvalidInput)
	}

	// Validate term (1-3)
	if payload.Term < 1 || payload.Term > 3 {
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrInvalidInput)
	}

	// Validate academic_year (>= 2017)
	if payload.AcademicYear < 2017 {
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrInvalidInput)
	}

	// Validate type
	if !validAssessmentTypes[payload.Type] {
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrInvalidInput)
	}

	// Validate grade_level
	if _, ok := gradeLevelToEducationLevel[payload.GradeLevel]; !ok {
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrInvalidInput)
	}

	bp := &AssessmentBlueprint{
		TenantID:     tenantID,
		SchoolID:     schoolID,
		Title:        payload.Title,
		Type:         payload.Type,
		GradeLevel:   payload.GradeLevel,
		AcademicYear: payload.AcademicYear,
		Term:         payload.Term,
	}

	id, err := s.Repo.CreateBlueprint(ctx, bp)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", ErrAlreadyExists)
		}
		return nil, fmt.Errorf("assessment.Service.CreateBlueprint: %w", err)
	}
	bp.ID = id

	slog.Info("assessment_blueprint.created",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"title", payload.Title,
		"type", payload.Type,
		"grade_level", payload.GradeLevel,
		"academic_year", payload.AcademicYear,
		"term", payload.Term,
	)

	return bp, nil
}

// ListBlueprints returns blueprints matching the given filters.
func (s *Service) ListBlueprints(ctx context.Context, tenantID string, query ListBlueprintsQuery) ([]AssessmentBlueprint, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("assessment.Service.ListBlueprints: %w", ErrInvalidInput)
	}
	return s.Repo.ListBlueprints(ctx, tenantID, query)
}

// GetBlueprintDetail returns a blueprint with its linked performance indicators.
func (s *Service) GetBlueprintDetail(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("assessment.Service.GetBlueprintDetail: %w", ErrInvalidInput)
	}
	return s.Repo.GetBlueprintDetail(ctx, id, tenantID, schoolID)
}

// UpdateBlueprint applies partial updates to a blueprint.
func (s *Service) UpdateBlueprint(ctx context.Context, id, tenantID, schoolID string, payload UpdateBlueprintPayload) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessment.Service.UpdateBlueprint: %w", ErrInvalidInput)
	}

	bp, err := s.Repo.GetBlueprintByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessment.Service.UpdateBlueprint: %w", err)
	}

	if payload.Title != nil {
		if *payload.Title == "" {
			return fmt.Errorf("assessment.Service.UpdateBlueprint: %w", ErrInvalidInput)
		}
		bp.Title = *payload.Title
	}
	if payload.Type != nil {
		if !validAssessmentTypes[*payload.Type] {
			return fmt.Errorf("assessment.Service.UpdateBlueprint: %w", ErrInvalidInput)
		}
		bp.Type = *payload.Type
	}

	if err := s.Repo.UpdateBlueprint(ctx, bp); err != nil {
		return fmt.Errorf("assessment.Service.UpdateBlueprint: %w", err)
	}

	slog.Info("assessment_blueprint.updated",
		"tenant_id", tenantID,
		"school_id", schoolID,
		"resource_id", id,
		"title", payload.Title,
		"type", payload.Type,
	)

	return nil
}

// DeleteBlueprint removes a blueprint. Returns ErrConflict if referenced by
// assessment sessions.
func (s *Service) DeleteBlueprint(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessment.Service.DeleteBlueprint: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteBlueprint(ctx, id, tenantID, schoolID)
}

// ============================================================================
// BLUEPRINT ↔ INDICATOR LINKING
// ============================================================================

// LinkIndicators links performance indicators to a blueprint after validating
// that each indicator belongs to a learning area with a compatible grade level.
func (s *Service) LinkIndicators(ctx context.Context, blueprintID, tenantID, schoolID string, payload LinkIndicatorPayload) error {
	if blueprintID == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessment.Service.LinkIndicators: %w", ErrInvalidInput)
	}
	if len(payload.IndicatorIDs) == 0 {
		return fmt.Errorf("assessment.Service.LinkIndicators: %w", ErrInvalidInput)
	}

	// Fetch the blueprint to get its grade_level
	bp, err := s.Repo.GetBlueprintByID(ctx, blueprintID, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("assessment.Service.LinkIndicators: %w", err)
	}

	expectedEducationLevel, ok := gradeLevelToEducationLevel[bp.GradeLevel]
	if !ok {
		return fmt.Errorf("assessment.Service.LinkIndicators: %w", ErrInvalidInput)
	}

	// Validate each indicator
	for _, indicatorID := range payload.IndicatorIDs {
		// Check if already linked
		linked, err := s.Repo.IsIndicatorLinked(ctx, blueprintID, indicatorID)
		if err != nil {
			return fmt.Errorf("assessment.Service.LinkIndicators: %w", err)
		}
		if linked {
			return fmt.Errorf("assessment.Service.LinkIndicators: indicator %s: %w", indicatorID, ErrIndicatorLinked)
		}

		// Validate grade level match via learning area resolver
		indicatorEducationLevel, err := s.LearningAreas.GetPerformanceIndicatorEducationLevel(ctx, indicatorID)
		if err != nil {
			return fmt.Errorf("assessment.Service.LinkIndicators: %w", err)
		}
		if indicatorEducationLevel != expectedEducationLevel {
			return fmt.Errorf("assessment.Service.LinkIndicators: indicator %s: %w", indicatorID, ErrGradeLevelMismatch)
		}
	}

	// Bulk link
	if err := s.Repo.LinkIndicators(ctx, blueprintID, payload.IndicatorIDs); err != nil {
		return fmt.Errorf("assessment.Service.LinkIndicators: %w", err)
	}

	slog.Info("assessment_blueprint_indicators.linked",
		"blueprint_id", blueprintID,
		"tenant_id", tenantID,
		"school_id", schoolID,
		"indicator_count", len(payload.IndicatorIDs),
	)

	return nil
}

// UnlinkIndicator removes a performance indicator from a blueprint.
func (s *Service) UnlinkIndicator(ctx context.Context, blueprintID, indicatorID, tenantID, schoolID string) error {
	if blueprintID == "" || indicatorID == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("assessment.Service.UnlinkIndicator: %w", ErrInvalidInput)
	}

	// Verify blueprint exists
	if _, err := s.Repo.GetBlueprintByID(ctx, blueprintID, tenantID, schoolID); err != nil {
		return fmt.Errorf("assessment.Service.UnlinkIndicator: %w", err)
	}

	if err := s.Repo.UnlinkIndicator(ctx, blueprintID, indicatorID); err != nil {
		return fmt.Errorf("assessment.Service.UnlinkIndicator: %w", err)
	}

	slog.Info("assessment_blueprint_indicator.unlinked",
		"blueprint_id", blueprintID,
		"indicator_id", indicatorID,
		"tenant_id", tenantID,
		"school_id", schoolID,
	)

	return nil
}

// ============================================================================
// ASSESSMENT SESSIONS
// ============================================================================

// CreateSession validates and creates a new assessment session.
func (s *Service) CreateSession(ctx context.Context, tenantID, userID string, payload CreateSessionPayload) (*AssessmentSession, error) {
	if tenantID == "" || userID == "" {
		return nil, fmt.Errorf("assessment.Service.CreateSession: %w", ErrInvalidInput)
	}

	// Validate blueprint_id
	if payload.BlueprintID == "" {
		return nil, fmt.Errorf("assessment.Service.CreateSession: %w", ErrInvalidInput)
	}

	// Validate class_id
	if payload.ClassID == "" {
		return nil, fmt.Errorf("assessment.Service.CreateSession: %w", ErrInvalidInput)
	}

	// Validate date_administered (YYYY-MM-DD format, not blank)
	if payload.DateAdministered == "" {
		return nil, fmt.Errorf("assessment.Service.CreateSession: %w", ErrInvalidInput)
	}
	if !isValidDate(payload.DateAdministered) {
		return nil, fmt.Errorf("assessment.Service.CreateSession: %w", ErrInvalidInput)
	}

	session := &AssessmentSession{
		TenantID:         tenantID,
		BlueprintID:      payload.BlueprintID,
		ClassID:          payload.ClassID,
		AssessedByUserID: userID,
		DateAdministered: payload.DateAdministered,
	}

	id, err := s.Repo.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("assessment.Service.CreateSession: %w", err)
	}
	session.ID = id

	slog.Info("assessment_session.created",
		"tenant_id", tenantID,
		"user_id", userID,
		"resource_id", id,
		"blueprint_id", payload.BlueprintID,
		"class_id", payload.ClassID,
		"date_administered", payload.DateAdministered,
	)

	return session, nil
}

// ListSessions returns sessions matching the given filters.
func (s *Service) ListSessions(ctx context.Context, tenantID string, query ListSessionsQuery) ([]AssessmentSession, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("assessment.Service.ListSessions: %w", ErrInvalidInput)
	}
	return s.Repo.ListSessions(ctx, tenantID, query)
}

// GetSessionDetail returns a session with its nested rubric results.
func (s *Service) GetSessionDetail(ctx context.Context, id, tenantID string) (*SessionDetail, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("assessment.Service.GetSessionDetail: %w", ErrInvalidInput)
	}
	return s.Repo.GetSessionDetail(ctx, id, tenantID)
}

// UpdateSession applies partial updates to a session.
func (s *Service) UpdateSession(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("assessment.Service.UpdateSession: %w", ErrInvalidInput)
	}

	session, err := s.Repo.GetSessionByID(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("assessment.Service.UpdateSession: %w", err)
	}

	if payload.DateAdministered != nil {
		if *payload.DateAdministered == "" {
			return fmt.Errorf("assessment.Service.UpdateSession: %w", ErrInvalidInput)
		}
		if !isValidDate(*payload.DateAdministered) {
			return fmt.Errorf("assessment.Service.UpdateSession: %w", ErrInvalidInput)
		}
		session.DateAdministered = *payload.DateAdministered
	}

	// If knec_upload_reference is explicitly set, update it (including setting to NULL)
	if payload.KNECUploadReference != nil {
		session.KNECUploadReference = payload.KNECUploadReference
	}

	if err := s.Repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("assessment.Service.UpdateSession: %w", err)
	}

	slog.Info("assessment_session.updated",
		"tenant_id", tenantID,
		"resource_id", id,
		"date_administered", payload.DateAdministered,
	)

	return nil
}

// DeleteSession removes a session and cascades its results.
func (s *Service) DeleteSession(ctx context.Context, id, tenantID string) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("assessment.Service.DeleteSession: %w", ErrInvalidInput)
	}
	return s.Repo.DeleteSession(ctx, id, tenantID)
}

// ============================================================================
// LEARNER RUBRIC RESULTS
// ============================================================================

// BatchUpsertResults validates and upserts a batch of rubric results.
func (s *Service) BatchUpsertResults(ctx context.Context, sessionID, tenantID string, payload BatchUpsertResultsPayload) (int, error) {
	if sessionID == "" || tenantID == "" {
		return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: %w", ErrInvalidInput)
	}
	if len(payload.Results) == 0 {
		return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: %w", ErrInvalidInput)
	}

	// Verify session exists
	session, err := s.Repo.GetSessionByID(ctx, sessionID, tenantID)
	if err != nil {
		return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: %w", err)
	}

	// Get the blueprint's linked indicators for validation
	// We use a simpler approach — validate each result's indicator exists.
	// Note: we can't call GetBlueprintDetail from the service here because
	// blueprints are in the same package. But we need to validate indicators
	// belong to the blueprint.
	// Fetch the blueprint's linked indicators for validation
	indicators, err := s.Repo.ListBlueprintIndicators(ctx, session.BlueprintID)
	if err != nil {
		return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: %w", err)
	}

	// Build set of valid indicator IDs for this blueprint
	validIndicators := make(map[string]bool, len(indicators))
	for _, ind := range indicators {
		validIndicators[ind.ID] = true
	}

	var results []LearnerRubricResult
	for i, input := range payload.Results {
		// Validate rubric level
		if !validRubricLevels[input.RubricLevel] {
			return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: invalid rubric_level %q: %w",
				i, input.RubricLevel, ErrInvalidRubricLevel)
		}

		// Validate score type
		if !validScoreTypes[input.ScoreType] {
			return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: invalid score_type %q: %w",
				i, input.ScoreType, ErrInvalidInput)
		}

		// Validate mutual exclusivity of score_type
		if input.ScoreType == "Rubric_Direct" && input.RawScore != nil {
			return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: %w",
				i, ErrScoreTypeMismatch)
		}
		if input.ScoreType == "Numeric_Raw" && input.RawScore == nil {
			return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: raw_score required for Numeric_Raw: %w",
				i, ErrInvalidInput)
		}

		// Validate indicator belongs to blueprint
		if !validIndicators[input.IndicatorID] {
			return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: indicator %s: %w",
				i, input.IndicatorID, ErrIndicatorNotInBP)
		}

		// Validate student belongs to class
		if s.ClassStudents != nil {
			inClass, err := s.ClassStudents.IsStudentInClass(ctx, input.StudentID, session.ClassID)
			if err != nil {
				return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: %w", i, err)
			}
			if !inClass {
				return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: result[%d]: student %s: %w",
					i, input.StudentID, ErrStudentNotInClass)
			}
		}

		results = append(results, LearnerRubricResult{
			SessionID:               sessionID,
			StudentID:               input.StudentID,
			IndicatorID:             input.IndicatorID,
			ScoreType:               input.ScoreType,
			RawScore:                input.RawScore,
			RubricLevel:             input.RubricLevel,
			TeacherObservationNotes: input.TeacherObservationNotes,
		})
	}

	count, err := s.Repo.BatchUpsertResults(ctx, sessionID, tenantID, results)
	if err != nil {
		return 0, fmt.Errorf("assessment.Service.BatchUpsertResults: %w", err)
	}

	slog.Info("learner_rubric_results.upserted",
		"tenant_id", tenantID,
		"session_id", sessionID,
		"result_count", len(results),
		"rows_affected", count,
	)

	return count, nil
}

// ListResults returns all rubric results for a given session.
func (s *Service) ListResults(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
	if sessionID == "" || tenantID == "" {
		return nil, fmt.Errorf("assessment.Service.ListResults: %w", ErrInvalidInput)
	}
	return s.Repo.ListResults(ctx, sessionID, tenantID)
}

// ============================================================================
// Helpers
// ============================================================================

// isValidDate checks if the given string is a valid YYYY-MM-DD date.
func isValidDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	// Basic range checks
	year := s[0:4]
	month := s[5:7]
	day := s[8:10]
	for _, c := range year {
		if c < '0' || c > '9' {
			return false
		}
	}
	for _, c := range month {
		if c < '0' || c > '9' {
			return false
		}
	}
	for _, c := range day {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ============================================================================
// WEIGHT CONFIGS
// ============================================================================

// ListWeightConfigs returns KNEC weight configs filtered by the given criteria.
func (s *Service) ListWeightConfigs(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error) {
	return s.Repo.ListWeightConfigs(ctx, query)
}

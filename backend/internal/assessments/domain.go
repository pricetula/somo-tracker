// Package assessments implements the CBC assessment & grading engine.
//
// Covers grading scale profiles & ranges, assessment session lifecycle
// (DRAFT → PENDING_APPROVAL → PUBLISHED), quantitative score capture with
// performance-level snapshotting, and rubric (indicator-level) grading.
package assessments

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// ── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound               = fmt.Errorf("assessments not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists          = fmt.Errorf("assessments already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput           = fmt.Errorf("invalid assessments input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized           = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden              = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict               = fmt.Errorf("assessments conflict: %w", middleware.ErrConflict)
	ErrInvalidStateTransition = fmt.Errorf("invalid state transition: %w", middleware.ErrConflict)
	ErrScoresExist            = fmt.Errorf("scores already exist for this session: %w", middleware.ErrConflict)
	ErrTermFinalised          = fmt.Errorf("term is finalised: %w", middleware.ErrForbidden)
	ErrScaleReferenced        = fmt.Errorf("scale profile is referenced by existing sessions: %w", middleware.ErrConflict)
	ErrStudentNotGradable     = fmt.Errorf("student is not in a gradable state: %w", middleware.ErrInvalidInput)
)

// ── Performance Level helpers ────────────────────────────────────────────

// ValidPerformanceLevels lists all allowed CBC rubric levels.
var ValidPerformanceLevels = []string{"EE", "ME", "AE", "BE"}

// IsValidPerformanceLevel returns true if the given string is a valid CBC level.
func IsValidPerformanceLevel(level string) bool {
	for _, l := range ValidPerformanceLevels {
		if l == level {
			return true
		}
	}
	return false
}

// ── Repository ───────────────────────────────────────────────────────────

// Repository defines the contract for assessment persistence.
type Repository interface {
	// Term finalisation check
	IsTermFinalised(ctx context.Context, termID string) (bool, error)
	// Grading Scale Profiles
	CreateScaleProfile(ctx context.Context, params CreateScaleProfileParams) (string, error)
	GetScaleProfileByID(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfile, error)
	ListScaleProfiles(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]ScaleProfile, error)
	ToggleScaleProfileActive(ctx context.Context, id, tenantID, schoolID string, isActive bool) error
	DeleteScaleProfile(ctx context.Context, id, tenantID, schoolID string) error

	// Grading Scale Ranges
	CreateScaleRange(ctx context.Context, params CreateScaleRangeParams) (string, error)
	GetScaleRangesByProfile(ctx context.Context, profileID, tenantID, schoolID string) ([]ScaleRange, error)
	DeleteScaleRange(ctx context.Context, rangeID, profileID, tenantID, schoolID string) error
	BulkSetScaleRanges(ctx context.Context, profileID string, ranges []CreateScaleRangeParams) ([]string, error)

	// Assessment Sessions
	CreateSession(ctx context.Context, params CreateSessionParams) (string, error)
	GetSessionByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error)
	GetSessionStatusAndTerm(ctx context.Context, id, tenantID string) (status, termID string, err error)
	ListSessions(ctx context.Context, tenantID, schoolID string, filters SessionFilters) ([]AssessmentSession, int, error)
	UpdateSessionStatus(ctx context.Context, id, tenantID, schoolID string, status string, rejectionComment *string, approvedBy *string) error
	HasScoresForSession(ctx context.Context, sessionID string) (bool, error)
	CountSessionsReferencingScale(ctx context.Context, profileID string) (int, error)

	// Student Scores (Quantitative)
	UpsertStudentScore(ctx context.Context, params UpsertScoreParams) error
	BulkUpsertStudentScores(ctx context.Context, params []UpsertScoreParams) error
	GetStudentScoresBySession(ctx context.Context, sessionID, tenantID, schoolID string) ([]StudentScore, error)
	SnapshotPerformanceLevels(ctx context.Context, sessionID string, profile *ScaleProfile) error

	// Student Outcome Grades (Rubric)
	UpsertOutcomeGrade(ctx context.Context, params UpsertOutcomeGradeParams) error
	BulkUpsertOutcomeGrades(ctx context.Context, params []UpsertOutcomeGradeParams) error
	GetOutcomeGradesBySession(ctx context.Context, sessionID, tenantID, schoolID string) ([]OutcomeGrade, error)
	GetOutcomeGradesByStudent(ctx context.Context, sessionID, studentID string) ([]OutcomeGrade, error)

	// Report Card Aggregation
	GetStudentTermGrades(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error)
	GetPublishedSessionsForParent(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]ParentAssessmentView, error)

	// Assessment Weight Configs
	CreateWeightConfig(ctx context.Context, params CreateWeightConfigParams) (string, error)
	ListWeightConfigs(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error)
	GetWeightConfigByID(ctx context.Context, id string) (*AssessmentWeightConfig, error)
}

// ── Domain Models ────────────────────────────────────────────────────────

// ScaleProfile represents a grading scale profile (the directory).
type ScaleProfile struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"-"`
	SchoolID  string    `json:"-"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScaleRange represents a single percentage-to-level mapping within a profile.
type ScaleRange struct {
	ID                       string   `json:"id"`
	ProfileID                string   `json:"profile_id"`
	PerformanceLevel         string   `json:"performance_level"`
	MinPercentage            float64  `json:"min_percentage"`
	MaxPercentage            float64  `json:"max_percentage"`
	DefaultPercentageMapping *float64 `json:"default_percentage_mapping,omitempty"`
}

// AssessmentSession represents an assessment session through its lifecycle.
type AssessmentSession struct {
	ID                    string    `json:"id"`
	TenantID              string    `json:"-"`
	SchoolID              string    `json:"-"`
	ClassID               string    `json:"class_id"`
	LearningAreaID        string    `json:"learning_area_id"`
	AcademicTermID        string    `json:"academic_term_id"`
	AcademicYearID        string    `json:"academic_year_id"`
	Name                  string    `json:"name"`
	EvaluationMethod      string    `json:"evaluation_method"`
	MaxPoints             *float64  `json:"max_points,omitempty"`
	GradingScaleProfileID *string   `json:"grading_scale_profile_id,omitempty"`
	Status                string    `json:"status"`
	RejectionComment      *string   `json:"rejection_comment,omitempty"`
	SubmittedBy           *string   `json:"submitted_by,omitempty"`
	ApprovedBy            *string   `json:"approved_by,omitempty"`
	ScheduledDate         *string   `json:"scheduled_date,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	CreatedBy             string    `json:"-"`
}

// StudentScore holds a student's quantitative assessment result.
type StudentScore struct {
	ID                    string   `json:"id"`
	SessionID             string   `json:"session_id"`
	StudentID             string   `json:"student_id"`
	RawScore              *float64 `json:"raw_score,omitempty"`
	CalculatedPercentage  *float64 `json:"calculated_percentage,omitempty"`
	FinalPerformanceLevel *string  `json:"final_performance_level,omitempty"`
	EnrollmentStatus      string   `json:"enrollment_status"`
}

// OutcomeGrade holds a student's rubric-level grade for a performance indicator.
type OutcomeGrade struct {
	ID                     string `json:"id"`
	SessionID              string `json:"session_id"`
	StudentID              string `json:"student_id"`
	PerformanceIndicatorID string `json:"performance_indicator_id"`
	AwardedLevel           string `json:"awarded_level"`
}

// StudentTermGrade represents the compiled term grade for a learning area.
type StudentTermGrade struct {
	LearningAreaID   string `json:"learning_area_id"`
	LearningAreaName string `json:"learning_area_name"`
	LearningAreaCode string `json:"learning_area_code"`
	FinalLevel       string `json:"final_level"`
	AssessmentCount  int    `json:"assessment_count"`
}

// ParentAssessmentView is a published assessment shown in the parent portal.
type ParentAssessmentView struct {
	SessionID        string         `json:"session_id"`
	SessionName      string         `json:"session_name"`
	EvaluationMethod string         `json:"evaluation_method"`
	ScheduledDate    *string        `json:"scheduled_date,omitempty"`
	RawScore         *float64       `json:"raw_score,omitempty"`
	MaxPoints        *float64       `json:"max_points,omitempty"`
	PerformanceLevel *string        `json:"performance_level,omitempty"`
	OutcomeGrades    []OutcomeGrade `json:"outcome_grades,omitempty"`
}

// ── Assessment Weight Configs ────────────────────────────────────────────

// CreateWeightConfigPayload is the JSON body for creating a weight config.
type CreateWeightConfigPayload struct {
	GradeLevel         string  `json:"grade_level"`
	AssessmentTypeCode string  `json:"assessment_type_code"`
	TargetExam         string  `json:"target_exam"`
	WeightPercent      float64 `json:"weight_percent"`
	EffectiveFrom      int     `json:"effective_from"`
	Notes              *string `json:"notes,omitempty"`
}

// AssessmentWeightConfig represents a KNEC weighting formula entry.
type AssessmentWeightConfig struct {
	ID                 string    `json:"id"`
	GradeLevel         string    `json:"grade_level"`
	AssessmentTypeCode string    `json:"assessment_type_code"`
	TargetExam         string    `json:"target_exam"`
	WeightPercent      float64   `json:"weight_percent"`
	EffectiveFrom      int       `json:"effective_from"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// AssessmentWeightConfigFilter holds filters for listing weight configs.
type AssessmentWeightConfigFilter struct {
	GradeLevel    *string
	TargetExam    *string
	EffectiveFrom *int
}

// CreateWeightConfigParams holds fields needed to create a weight config.
type CreateWeightConfigParams struct {
	GradeLevel         string
	AssessmentTypeCode string
	TargetExam         string
	WeightPercent      float64
	EffectiveFrom      int
	Notes              *string
}

// ListWeightConfigsResponse wraps a list of weight configs.
type ListWeightConfigsResponse struct {
	Items []AssessmentWeightConfig `json:"items"`
}

// ── Params (internal) ────────────────────────────────────────────────────

// CreateScaleProfileParams holds fields needed to create a scale profile.
type CreateScaleProfileParams struct {
	TenantID string
	SchoolID string
	Name     string
}

// CreateScaleRangeParams holds fields needed to create a scale range.
type CreateScaleRangeParams struct {
	ProfileID                string
	PerformanceLevel         string
	MinPercentage            float64
	MaxPercentage            float64
	DefaultPercentageMapping *float64
}

// CreateSessionParams holds fields needed to create an assessment session.
type CreateSessionParams struct {
	TenantID              string
	SchoolID              string
	ClassID               string
	LearningAreaID        string
	AcademicTermID        string
	AcademicYearID        string
	Name                  string
	EvaluationMethod      string
	MaxPoints             *float64
	GradingScaleProfileID *string
	CreatedBy             string
	ScheduledDate         *string
}

// UpsertScoreParams holds fields for upserting a student's quantitative score.
type UpsertScoreParams struct {
	TenantID         string
	SessionID        string
	StudentID        string
	RawScore         *float64
	EnrollmentStatus string
}

// UpsertOutcomeGradeParams holds fields for upserting a rubric outcome grade.
type UpsertOutcomeGradeParams struct {
	TenantID               string
	SessionID              string
	StudentID              string
	PerformanceIndicatorID string
	AwardedLevel           string
}

// SessionFilters for listing assessment sessions.
type SessionFilters struct {
	ClassID          *string
	LearningAreaID   *string
	AcademicTermID   *string
	Status           *string
	EvaluationMethod *string
	Search           string
	Page             int
	Limit            int
}

// ── HTTP Request / Response Payloads ─────────────────────────────────────

// CreateScaleProfilePayload is the JSON body for creating a scale profile.
type CreateScaleProfilePayload struct {
	Name string `json:"name"`
}

// BulkSetRangesPayload is the JSON body for setting all ranges on a profile.
type BulkSetRangesPayload struct {
	Ranges []ScaleRangePayload `json:"ranges"`
}

// ScaleRangePayload is a single range definition in JSON.
type ScaleRangePayload struct {
	PerformanceLevel         string   `json:"performance_level"`
	MinPercentage            float64  `json:"min_percentage"`
	MaxPercentage            float64  `json:"max_percentage"`
	DefaultPercentageMapping *float64 `json:"default_percentage_mapping,omitempty"`
}

// ListScaleProfilesResponse is the response for listing scale profiles.
type ListScaleProfilesResponse struct {
	Items []ScaleProfile `json:"items"`
}

// ScaleProfileWithRanges is a profile with its ranges included.
type ScaleProfileWithRanges struct {
	ScaleProfile
	Ranges []ScaleRange `json:"ranges"`
}

// CreateSessionPayload is the JSON body for creating an assessment session.
type CreateSessionPayload struct {
	ClassID               string   `json:"class_id"`
	LearningAreaID        string   `json:"learning_area_id"`
	AcademicTermID        string   `json:"academic_term_id"`
	AcademicYearID        string   `json:"academic_year_id"`
	Name                  string   `json:"name"`
	EvaluationMethod      string   `json:"evaluation_method"`
	MaxPoints             *float64 `json:"max_points,omitempty"`
	GradingScaleProfileID *string  `json:"grading_scale_profile_id,omitempty"`
	ScheduledDate         *string  `json:"scheduled_date,omitempty"`
}

// ListSessionsResponse is the paginated response for listing sessions.
type ListSessionsResponse struct {
	Items []AssessmentSession `json:"items"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// BulkUpsertScoresPayload is the JSON body for bulk upserting scores.
type BulkUpsertScoresPayload struct {
	Scores []StudentScorePayload `json:"scores"`
}

// StudentScorePayload is a single score entry in a bulk upsert.
type StudentScorePayload struct {
	StudentID string   `json:"student_id"`
	RawScore  *float64 `json:"raw_score,omitempty"`
}

// BulkUpsertOutcomeGradesPayload is the JSON body for bulk upserting rubric grades.
type BulkUpsertOutcomeGradesPayload struct {
	Grades []OutcomeGradePayload `json:"grades"`
}

// OutcomeGradePayload is a single rubric grade entry.
type OutcomeGradePayload struct {
	StudentID              string `json:"student_id"`
	PerformanceIndicatorID string `json:"performance_indicator_id"`
	AwardedLevel           string `json:"awarded_level"`
}

// RejectSessionPayload is the JSON body for rejecting a session.
type RejectSessionPayload struct {
	RejectionComment string `json:"rejection_comment"`
}

// StudentScoresResponse is the response for getting student scores for a session.
type StudentScoresResponse struct {
	Items []StudentScore `json:"items"`
}

// OutcomeGradesResponse is the response for getting outcome grades for a session.
type OutcomeGradesResponse struct {
	Items []OutcomeGrade `json:"items"`
}

// ParentTermAssessmentsResponse is the response shown in the parent portal.
type ParentTermAssessmentsResponse struct {
	Items []ParentAssessmentView `json:"items"`
}

// StudentTermGradesResponse is the compiled term grades response.
type StudentTermGradesResponse struct {
	Items []StudentTermGrade `json:"items"`
}

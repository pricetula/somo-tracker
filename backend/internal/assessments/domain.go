// Package assessments implements the CBC assessment & grading engine.
//
// Covers grading scale profiles & ranges, assessment session lifecycle
// (DRAFT → PENDING_APPROVAL → PUBLISHED), quantitative score capture with
// performance-level snapshotting, and rubric (indicator-level) grading.
package assessments

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// ── Sentinel domain errors ───────────────────────────────────────────────

var (
	ErrNotFound               = xerrors.NotFound("assessment not found")
	ErrAlreadyExists          = xerrors.AlreadyExists("assessment already exists")
	ErrInvalidInput           = xerrors.InvalidInput("invalid assessment input")
	ErrUnauthorized           = xerrors.Unauthorized("unauthorized")
	ErrForbidden              = xerrors.Forbidden("forbidden")
	ErrConflict               = xerrors.Conflict("assessment conflict")
	ErrInvalidStateTransition = xerrors.Conflict("invalid state transition")
	ErrScoresExist            = xerrors.Conflict("scores already exist for this session")
	ErrTermFinalised          = xerrors.Forbidden("term is finalised")
	ErrScaleReferenced        = xerrors.Conflict("scale profile is referenced by existing sessions")
	ErrStudentNotGradable     = xerrors.InvalidInput("student is not in a gradable state")
	// ErrInternal is a recognized *xerrors.DomainError so that the HTTP
	// error handler maps it to a 500 with a generic message.
	ErrInternal = &xerrors.DomainError{Code: "internal_error", Status: 500, Message: "an unexpected error occurred"}
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
	// Grading Scale Profiles (profiles are always created with their ranges)
	CreateScaleProfileWithRanges(ctx context.Context, params CreateScaleProfileParams) (profileID string, rangeIDs []string, err error)
	GetScaleProfileByID(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfileWithRanges, error)
	ListScaleProfiles(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]ScaleProfileWithRanges, error)
	ToggleScaleProfileActive(ctx context.Context, id, tenantID, schoolID string, isActive bool) error
	DeleteScaleProfile(ctx context.Context, id, tenantID, schoolID string) error

	// Grading Scale Ranges (standalone range endpoints)
	GetScaleRanges(ctx context.Context, profileID string) ([]ScaleRange, error)
	ReplaceScaleRanges(ctx context.Context, profileID string, ranges []CreateScaleRangeParams) ([]string, error)

	// Assessment Sessions
	CreateSession(ctx context.Context, params CreateSessionParams) (string, error)
	GetSessionByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error)
	GetSessionStatusAndTerm(ctx context.Context, id, tenantID string) (status, termID string, err error)
	ListSessions(ctx context.Context, tenantID, schoolID string, filters SessionFilters) ([]AssessmentSession, int, error)
	UpdateSessionStatus(ctx context.Context, id, tenantID, schoolID string, status string, rejectionComment *string, approvedBy *string) error
	DeleteSession(ctx context.Context, id, tenantID, schoolID string) error
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

	// Student Term Subject Summaries
	RefreshSessionSummary(ctx context.Context, sessionID string) error
	GetStudentTermSubjectSummaries(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermSubjectSummary, error)
	GetLearningAreaSummaries(ctx context.Context, tenantID, schoolID, termID, learningAreaID string) ([]StudentTermSubjectSummary, error)
	SetTeacherRemark(ctx context.Context, summaryID, tenantID, schoolID string, remark *string) error

	// Assessment Weight Configs
	CreateWeightConfig(ctx context.Context, params CreateWeightConfigParams) (string, error)
	ListWeightConfigs(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error)
	GetWeightConfigByID(ctx context.Context, id string) (*AssessmentWeightConfig, error)
	DeleteWeightConfig(ctx context.Context, id string) error

	// Student Term Overall Summaries (term-level rollup across subjects)
	RefreshTermOverallSummaries(ctx context.Context, termID string) error
	RefreshSingleStudentOverallSummary(ctx context.Context, studentID, termID string) error
	GetStudentTermOverallSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*StudentTermOverallSummary, error)
	ListStudentTermOverallSummaries(ctx context.Context, tenantID, schoolID, termID string) ([]StudentTermOverallSummary, error)
	SetHeadteacherRemark(ctx context.Context, summaryID, tenantID, schoolID string, remark *string) error

	// Student Subject Strand Summaries (rubric-only sub-strand level)
	RefreshSubjectStrandSummaries(ctx context.Context, sessionID string) error
	GetStudentSubjectStrandSummaries(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentSubjectStrandSummary, error)
	GetSubjectStrandSummariesByTerm(ctx context.Context, tenantID, schoolID, termID string) ([]StudentSubjectStrandSummary, error)

	// Student Performance Projections (periodic batch)
	RefreshProjections(ctx context.Context, termID string) error
	GetStudentProjection(ctx context.Context, tenantID, schoolID, studentID, termID string, learningAreaID *string) (*StudentPerformanceProjection, error)
	ListStudentProjections(ctx context.Context, tenantID, schoolID, termID string) ([]StudentPerformanceProjection, error)
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
	TenantID                 string   `json:"-"`
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
	MaxPoints             *float64  `json:"max_points"`
	GradingScaleProfileID *string   `json:"grading_scale_profile_id"`
	Status                string    `json:"status"`
	RejectionComment      *string   `json:"rejection_comment"`
	SubmittedBy           *string   `json:"submitted_by"`
	ApprovedBy            *string   `json:"approved_by"`
	ScheduledDate         *string   `json:"scheduled_date"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	CreatedBy             string    `json:"-"`
	ClassName             string    `json:"class_name"`
	GradeLevel            string    `json:"grade_level"`
	EnrollStudentsURL     *string   `json:"enroll_students_url,omitempty"`
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

// StudentTermSubjectSummary represents the blended summary of assessment
// results for a single student, term, and learning area.
//
// Quantitative scores contribute their calculated_percentage directly.
// Rubric outcome grades are converted via default_percentage_mapping.
// Both sources are blended into average_percentage, with source-type flags
// so reports can avoid implying false precision for rubric-only subjects.
type StudentTermSubjectSummary struct {
	ID                          string   `json:"id"`
	TenantID                    string   `json:"-"`
	SchoolID                    string   `json:"-"`
	StudentID                   string   `json:"student_id"`
	AcademicTermID              string   `json:"academic_term_id"`
	LearningAreaID              string   `json:"learning_area_id"`
	AveragePercentage           *float64 `json:"average_percentage,omitempty"`
	MappedPerformanceLevel      *string  `json:"mapped_performance_level,omitempty"`
	QuantitativeAssessmentCount int      `json:"quantitative_assessment_count"`
	RubricAssessmentCount       int      `json:"rubric_assessment_count"`
	IndicatorsAssessedCount     int      `json:"indicators_assessed_count"`
	HasQuantitativeData         bool     `json:"has_quantitative_data"`
	HasRubricData               bool     `json:"has_rubric_data"`
	TeacherRemark               *string  `json:"teacher_remark,omitempty"`
	LastRefreshedAt             string   `json:"last_refreshed_at"`
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
	Total int                      `json:"total"`
	Page  int                      `json:"page"`
	Limit int                      `json:"limit"`
}

// ── Grading Data (merged roster + scores) ───────────────────────────────

// RosterStudent represents a student in the class roster for grading.
type RosterStudent struct {
	StudentID       string
	StudentName     string
	AdmissionNumber string
	Gender          string
}

// RosterProvider provides class roster data for the assessments domain.
type RosterProvider interface {
	GetRosterByClassAndTerm(ctx context.Context, classID, tenantID, schoolID, academicTermID string) ([]RosterStudent, error)
}

// GradingDataStudent represents a student with their score or grades attached.
type GradingDataStudent struct {
	StudentID        string         `json:"student_id"`
	StudentName      string         `json:"student_name"`
	AdmissionNumber  string         `json:"admission_number"`
	Gender           string         `json:"gender"`
	EnrollmentStatus string         `json:"enrollment_status"`
	Score            *StudentScore  `json:"score,omitempty"`
	Grades           []OutcomeGrade `json:"grades,omitempty"`
}

// GradingDataResponse is the response for GET /assessments/sessions/:id/grading-data.
type GradingDataResponse struct {
	Session  *AssessmentSession   `json:"session"`
	Students []GradingDataStudent `json:"students"`
}

// ── Params (internal) ────────────────────────────────────────────────────

// CreateScaleProfileParams holds fields needed to create a scale profile.
type CreateScaleProfileParams struct {
	TenantID string
	SchoolID string
	Name     string
	Ranges   []CreateScaleRangeParams
}

// CreateScaleRangeParams holds fields needed to create a scale range.
type CreateScaleRangeParams struct {
	TenantID                 string
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
// When ranges are included, the profile and its ranges are created atomically.
type CreateScaleProfilePayload struct {
	Name   string              `json:"name"`
	Ranges []ScaleRangePayload `json:"ranges,omitempty"`
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
	Items []ScaleProfileWithRanges `json:"items"`
	Total int                      `json:"total"`
	Page  int                      `json:"page"`
	Limit int                      `json:"limit"`
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

// StudentTermSubjectSummariesResponse wraps a list of summaries.
type StudentTermSubjectSummariesResponse struct {
	Items []StudentTermSubjectSummary `json:"items"`
}

// ── Student Subject Strand Summaries ───────────────────────────────────────

// StudentSubjectStrandSummary represents the rubric-only sub-strand level
// summary for a single student, term, and sub-strand. It counts how many
// performance indicators were awarded at each CBC level and computes a
// mastery_percentage.
//
// This table only ever has data where the subject was assessed via RUBRIC
// sessions. For subjects taught purely quantitatively, has_data stays false
// rather than showing a misleading 0%.
type StudentSubjectStrandSummary struct {
	ID                     string   `json:"id"`
	TenantID               string   `json:"-"`
	SchoolID               string   `json:"-"`
	StudentID              string   `json:"student_id"`
	AcademicTermID         string   `json:"academic_term_id"`
	LearningAreaID         string   `json:"learning_area_id"`
	StrandID               string   `json:"strand_id"`
	SubStrandID            string   `json:"sub_strand_id"`
	MasteryPercentage      *float64 `json:"mastery_percentage,omitempty"`
	ExceedingCount         int      `json:"exceeding_count"`
	MeetingCount           int      `json:"meeting_count"`
	ApproachingCount       int      `json:"approaching_count"`
	BelowCount             int      `json:"below_count"`
	MappedPerformanceLevel *string  `json:"mapped_performance_level,omitempty"`
	RequiresRemediation    bool     `json:"requires_remediation"`
	HasData                bool     `json:"has_data"`
	LastRefreshedAt        string   `json:"last_refreshed_at"`
}

// StudentSubjectStrandSummariesResponse wraps a list of strand summaries.
type StudentSubjectStrandSummariesResponse struct {
	Items []StudentSubjectStrandSummary `json:"items"`
	Total int                           `json:"total"`
}

// ── Student Term Overall Summary ─────────────────────────────────────────

// StudentTermOverallSummary represents the term-level rollup across all
// learning areas for a student. It aggregates the per-subject summaries
// and applies KNEC weighting formulas when the term is a final exam term
// (G6→KPSEA, G9→KJSEA, G12→KSSEA).
type StudentTermOverallSummary struct {
	ID                      string   `json:"id"`
	TenantID                string   `json:"-"`
	SchoolID                string   `json:"-"`
	StudentID               string   `json:"student_id"`
	AcademicTermID          string   `json:"academic_term_id"`
	SubjectsAssessedCount   int      `json:"subjects_assessed_count"`
	OverallMeanPercentage   *float64 `json:"overall_mean_percentage,omitempty"`
	OverallPerformanceLevel *string  `json:"overall_performance_level,omitempty"`
	ExceedingCount          int      `json:"exceeding_count"`
	MeetingCount            int      `json:"meeting_count"`
	ApproachingCount        int      `json:"approaching_count"`
	BelowCount              int      `json:"below_count"`
	IsWeightedExamScore     bool     `json:"is_weighted_exam_score"`
	HeadteacherRemark       *string  `json:"headteacher_remark,omitempty"`
	LastRefreshedAt         string   `json:"last_refreshed_at"`
}

// StudentTermOverallSummaryResponse wraps a single overall summary.
type StudentTermOverallSummaryResponse struct {
	Data StudentTermOverallSummary `json:"data"`
}

// StudentTermOverallSummariesListResponse wraps a list of overall summaries.
type StudentTermOverallSummariesListResponse struct {
	Items []StudentTermOverallSummary `json:"items"`
}

// SetHeadteacherRemarkPayload is the request body for setting the headteacher remark.
type SetHeadteacherRemarkPayload struct {
	Remark *string `json:"remark"`
}

// ── Student Performance Projections ─────────────────────────────────────

// StudentPerformanceProjection represents the computed performance projection
// for a student in a term, optionally scoped to a specific learning area.
//
// Projections are computed via periodic batch (once per term close), NOT
// incrementally. A single new score should not reshuffle a trend line.
type StudentPerformanceProjection struct {
	ID                        string   `json:"id"`
	TenantID                  string   `json:"-"`
	SchoolID                  string   `json:"-"`
	StudentID                 string   `json:"student_id"`
	AcademicTermID            string   `json:"academic_term_id"`
	LearningAreaID            *string  `json:"learning_area_id,omitempty"`
	MomentumScore             *float64 `json:"momentum_score,omitempty"`
	ProjectedScore            *float64 `json:"projected_score,omitempty"`
	ProjectedPerformanceLevel *string  `json:"projected_performance_level,omitempty"`
	TargetGapPoints           *float64 `json:"target_gap_points,omitempty"`
	RiskLevel                 string   `json:"risk_level"`
	ConfidencePercentage      *float64 `json:"confidence_percentage,omitempty"`
	LastRefreshedAt           string   `json:"last_refreshed_at"`
	CreatedAt                 string   `json:"created_at,omitempty"`
	UpdatedAt                 string   `json:"updated_at,omitempty"`
}

// PerformanceProjectionResponse wraps a single projection.
type PerformanceProjectionResponse struct {
	Data StudentPerformanceProjection `json:"data"`
}

// PerformanceProjectionListResponse wraps a list of projections.
type PerformanceProjectionListResponse struct {
	Items []StudentPerformanceProjection `json:"items"`
	Total int                            `json:"total"`
}

// RefreshProjectionsRequest is the request body for triggering a projection refresh.
type RefreshProjectionsRequest struct {
	AcademicTermID string `json:"academic_term_id"`
}

// RefreshProjectionsResponse is returned after triggering a projection refresh.
type RefreshProjectionsResponse struct {
	Message string `json:"message"`
	TermID  string `json:"term_id"`
}

// SetTeacherRemarkPayload is the request body for setting a teacher remark.
type SetTeacherRemarkPayload struct {
	Remark *string `json:"remark"`
}

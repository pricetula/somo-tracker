package assessment

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

var (
	ErrNotFound      = fmt.Errorf("assessment not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("assessment already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid assessment input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("assessment conflict: %w", middleware.ErrConflict)
)

// Module-specific sentinels.
var (
	ErrGradeLevelMismatch      = fmt.Errorf("indicator grade level mismatch: %w", middleware.ErrInvalidInput)
	ErrIndicatorLinked         = fmt.Errorf("indicator already linked: %w", middleware.ErrConflict)
	ErrInvalidRubricLevel      = fmt.Errorf("invalid rubric level: %w", middleware.ErrInvalidInput)
	ErrScoreTypeMismatch       = fmt.Errorf("score type mismatch: %w", middleware.ErrInvalidInput)
	ErrStudentNotInClass       = fmt.Errorf("student not in class: %w", middleware.ErrInvalidInput)
	ErrIndicatorNotInBP        = fmt.Errorf("indicator not in blueprint: %w", middleware.ErrInvalidInput)
	ErrSessionLocked           = fmt.Errorf("session is locked for edits: %w", middleware.ErrForbidden)
	ErrInvalidStatusTransition = fmt.Errorf("invalid status transition: %w", middleware.ErrInvalidInput)
	ErrNoResultsToSubmit       = fmt.Errorf("session has no results: %w", middleware.ErrInvalidInput)
	ErrRejectionReasonRequired = fmt.Errorf("rejection reason is required: %w", middleware.ErrInvalidInput)
	ErrNotSessionOwner         = fmt.Errorf("not the session owner: %w", middleware.ErrForbidden)
	ErrNotAdmin                = fmt.Errorf("admin role required: %w", middleware.ErrForbidden)
	ErrBatchPublishEmpty       = fmt.Errorf("no session IDs provided for batch publish: %w", middleware.ErrInvalidInput)
)

// ============================================================================
// Session Status
// ============================================================================

// ValidSessionStatuses is the set of valid cbc_session_status values.
var ValidSessionStatuses = map[string]bool{
	"DRAFT":          true,
	"PENDING_REVIEW": true,
	"APPROVED":       true,
	"PUBLISHED":      true,
	"REJECTED":       true,
}

// AllowedTransitions defines the valid state machine for session status.
// Key = current status, value = set of allowed next statuses.
var AllowedTransitions = map[string]map[string]bool{
	"DRAFT":          {"PENDING_REVIEW": true},
	"REJECTED":       {"PENDING_REVIEW": true},
	"PENDING_REVIEW": {"APPROVED": true, "REJECTED": true},
	"APPROVED":       {"PUBLISHED": true},
	"PUBLISHED":      {}, // terminal state
}

// ============================================================================
// Domain Models
// ============================================================================

// AssessmentBlueprint represents a per-school assessment plan for a grade/term.
type AssessmentBlueprint struct {
	ID           string `json:"id"`
	TenantID     string `json:"-"`
	SchoolID     string `json:"school_id"`
	Title        string `json:"title"`
	Type         string `json:"type"`        // cbc_assessment_type enum
	GradeLevel   string `json:"grade_level"` // cbc_grade_level enum
	AcademicYear int    `json:"academic_year"`
	Term         int    `json:"term"` // 1-3
	CreatedAt    string `json:"created_at"`
}

// BlueprintDetail extends AssessmentBlueprint with linked performance indicators.
type BlueprintDetail struct {
	AssessmentBlueprint
	Indicators []LinkedIndicator `json:"indicators"`
}

// LinkedIndicator represents a performance indicator linked to a blueprint.
type LinkedIndicator struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// AssessmentSession represents an assessment administered to a class on a specific date.
type AssessmentSession struct {
	ID                  string  `json:"id"`
	TenantID            string  `json:"-"`
	BlueprintID         string  `json:"blueprint_id"`
	ClassID             string  `json:"class_id"`
	AssessedByUserID    string  `json:"assessed_by_user_id"`
	DateAdministered    string  `json:"date_administered"` // "YYYY-MM-DD"
	KNECUploadReference *string `json:"knec_upload_reference,omitempty"`
	Status              string  `json:"status"` // cbc_session_status
	SubmittedAt         *string `json:"submitted_at,omitempty"`
	ReviewedByUserID    *string `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt          *string `json:"reviewed_at,omitempty"`
	RejectionReason     *string `json:"rejection_reason,omitempty"`
	PublishedByUserID   *string `json:"published_by_user_id,omitempty"`
	PublishedAt         *string `json:"published_at,omitempty"`
	CreatedAt           string  `json:"created_at"`
}

// SessionDetail extends AssessmentSession with nested rubric results.
type SessionDetail struct {
	AssessmentSession
	Results []LearnerRubricResult `json:"results"`
}

// SessionAdminView extends AssessmentSession with blueprint and teacher info
// for admin queue listings.
type SessionAdminView struct {
	AssessmentSession
	BlueprintTitle string `json:"blueprint_title"`
	BlueprintType  string `json:"blueprint_type"`
	GradeLevel     string `json:"grade_level"`
	AcademicYear   int    `json:"academic_year"`
	Term           int    `json:"term"`
	ClassName      string `json:"class_name"`
	StreamName     string `json:"stream_name"`
	TeacherName    string `json:"teacher_name"`
	ResultCount    int    `json:"result_count"`
}

// LearnerRubricResult represents a single student's rubric outcome for one
// performance indicator within an assessment session.
type LearnerRubricResult struct {
	ID                      string  `json:"id"`
	SessionID               string  `json:"session_id"`
	StudentID               string  `json:"student_id"`
	IndicatorID             string  `json:"indicator_id"`
	ScoreType               string  `json:"score_type"`          // "Numeric_Raw" | "Rubric_Direct"
	RawScore                *string `json:"raw_score,omitempty"` // NUMERIC(5,2)
	RubricLevel             string  `json:"rubric_level"`        // EE | ME | AE | BE
	TeacherObservationNotes *string `json:"teacher_observation_notes,omitempty"`
}

// AssessmentWeightConfig represents a KNEC-mandated national grading weight.
type AssessmentWeightConfig struct {
	ID                 string `json:"id"`
	GradeLevel         string `json:"grade_level"`
	AssessmentTypeCode string `json:"assessment_type_code"`
	TargetExam         string `json:"target_exam"`
	WeightPercent      string `json:"weight_percent"` // NUMERIC(5,2) as string
	EffectiveFrom      int    `json:"effective_from"`
}

// ============================================================================
// Request / Response Payloads
// ============================================================================

type CreateBlueprintPayload struct {
	Title        string `json:"title"`
	Type         string `json:"type"`
	GradeLevel   string `json:"grade_level"`
	AcademicYear int    `json:"academic_year"`
	Term         int    `json:"term"`
}

type UpdateBlueprintPayload struct {
	Title *string `json:"title,omitempty"`
	Type  *string `json:"type,omitempty"`
}

type ListBlueprintsQuery struct {
	SchoolID     string
	GradeLevel   string
	Term         int
	AcademicYear int
}

type CreateBlueprintResponse struct {
	ID string `json:"id"`
}

type ListBlueprintsResponse struct {
	Items []AssessmentBlueprint `json:"items"`
}

type BlueprintDetailResponse struct {
	Data BlueprintDetail `json:"data"`
}

type LinkIndicatorPayload struct {
	IndicatorIDs []string `json:"indicator_ids"`
}

type ListWeightConfigsQuery struct {
	GradeLevel string
	TargetExam string
}

type ListWeightConfigsResponse struct {
	Items []AssessmentWeightConfig `json:"items"`
}

// ============================================================================
// Session & Result Payloads
// ============================================================================

type CreateSessionPayload struct {
	BlueprintID      string `json:"blueprint_id"`
	ClassID          string `json:"class_id"`
	DateAdministered string `json:"date_administered"` // "YYYY-MM-DD"
}

type UpdateSessionPayload struct {
	DateAdministered    *string `json:"date_administered,omitempty"`
	KNECUploadReference *string `json:"knec_upload_reference,omitempty"`
}

type ListSessionsQuery struct {
	ClassID     string
	BlueprintID string
	Status      string
	Page        int
	PageSize    int
}

// AdminQueuesQuery captures filtering for admin queue views.
type AdminQueuesQuery struct {
	ClassID      string
	GradeLevel   string
	Term         int
	AcademicYear int
	Status       string // PENDING_REVIEW or APPROVED
	Page         int
	PageSize     int
}

// ParentSessionsQuery captures filtering for parent-facing results.
type ParentSessionsQuery struct {
	StudentID    string
	AcademicYear int
	Term         int
	Page         int
	PageSize     int
}

type CreateSessionResponse struct {
	ID string `json:"id"`
}

type ListSessionsResponse struct {
	Items      []AssessmentSession `json:"items"`
	TotalCount int                 `json:"total_count"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
}

type SessionDetailResponse struct {
	Data SessionDetail `json:"data"`
}

type BatchUpsertResultInput struct {
	StudentID               string  `json:"student_id"`
	IndicatorID             string  `json:"indicator_id"`
	ScoreType               string  `json:"score_type"`
	RawScore                *string `json:"raw_score,omitempty"`
	RubricLevel             string  `json:"rubric_level"`
	TeacherObservationNotes *string `json:"teacher_observation_notes,omitempty"`
}

type BatchUpsertResultsPayload struct {
	Results []BatchUpsertResultInput `json:"results"`
}

type ListResultsResponse struct {
	Items      []LearnerRubricResult `json:"items"`
	TotalCount int                   `json:"total_count"`
}

// ============================================================================
// Status Transition Payloads
// ============================================================================

type SubmitForReviewPayload struct {
	SessionID string `json:"session_id"`
}

type ApproveSessionPayload struct {
	SessionID string `json:"session_id"`
}

type RejectSessionPayload struct {
	SessionID       string `json:"session_id"`
	RejectionReason string `json:"rejection_reason"`
}

type PublishSessionPayload struct {
	SessionID string `json:"session_id"`
}

type BatchPublishPayload struct {
	SessionIDs []string `json:"session_ids"`
}

type BatchPublishResponse struct {
	PublishedCount int      `json:"published_count"`
	FailedIDs      []string `json:"failed_ids,omitempty"`
}

type StatusTransitionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type AdminQueuesResponse struct {
	Items      []SessionAdminView `json:"items"`
	TotalCount int                `json:"total_count"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
}

type ParentPublishedResultsResponse struct {
	Items      []ParentSessionResultView `json:"items"`
	TotalCount int                       `json:"total_count"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
}

// ParentSessionResultView is the parent-facing view of a published result.
// Never exposes raw scores or computed averages per CBC compliance rules.
type ParentSessionResultView struct {
	SessionID               string  `json:"session_id"`
	DateAdministered        string  `json:"date_administered"`
	LearningAreaName        string  `json:"learning_area_name"`
	BlueprintTitle          string  `json:"blueprint_title"`
	BlueprintType           string  `json:"blueprint_type"`
	RubricLevel             string  `json:"rubric_level"`
	IndicatorDescription    string  `json:"indicator_description"`
	TeacherObservationNotes *string `json:"teacher_observation_notes,omitempty"`
}

// ============================================================================
// LearningAreaResolver — cross-domain interface for grade level validation
// ============================================================================

// LearningAreaResolver resolves a performance indicator's education level for
// grade level validation. The curriculum repository implements this interface.
type LearningAreaResolver interface {
	GetPerformanceIndicatorEducationLevel(ctx context.Context, indicatorID string) (string, error)
}

// ============================================================================
// ClassStudentResolver — cross-domain interface for student membership validation
// ============================================================================

// ClassStudentResolver checks whether a student is enrolled in a given class.
// The students or cbcclasses repository implements this interface.
type ClassStudentResolver interface {
	IsStudentInClass(ctx context.Context, studentID, classID string) (bool, error)
}

// ============================================================================
// BlueprintIndicatorResolver — cross-domain interface for indicator validation
// ============================================================================

// BlueprintIndicatorResolver checks whether a performance indicator is linked
// to a blueprint. The assessment repository implements this interface.
type BlueprintIndicatorResolver interface {
	IsIndicatorLinked(ctx context.Context, blueprintID, indicatorID string) (bool, error)
}

// ============================================================================
// Repository Interface
// ============================================================================

// Repository defines the contract for assessment persistence.
type Repository interface {
	// Blueprints
	CreateBlueprint(ctx context.Context, bp *AssessmentBlueprint) (string, error)
	GetBlueprintByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error)
	ListBlueprints(ctx context.Context, tenantID string, query ListBlueprintsQuery) ([]AssessmentBlueprint, error)
	UpdateBlueprint(ctx context.Context, bp *AssessmentBlueprint) error
	DeleteBlueprint(ctx context.Context, id, tenantID, schoolID string) error

	// Blueprint Detail (with indicators)
	GetBlueprintDetail(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error)

	// Blueprint ↔ Indicator Linking
	LinkIndicators(ctx context.Context, blueprintID string, indicatorIDs []string) error
	UnlinkIndicator(ctx context.Context, blueprintID, indicatorID string) error
	IsIndicatorLinked(ctx context.Context, blueprintID, indicatorID string) (bool, error)
	ListBlueprintIndicators(ctx context.Context, blueprintID string) ([]LinkedIndicator, error)

	// Sessions
	CreateSession(ctx context.Context, s *AssessmentSession) (string, error)
	GetSessionByID(ctx context.Context, id, tenantID string) (*AssessmentSession, error)
	ListSessions(ctx context.Context, tenantID string, query ListSessionsQuery) ([]AssessmentSession, int, error)
	UpdateSession(ctx context.Context, s *AssessmentSession) error
	DeleteSession(ctx context.Context, id, tenantID string) error

	// Session Detail (with results)
	GetSessionDetail(ctx context.Context, id, tenantID string) (*SessionDetail, error)

	// Session Status Transitions
	SubmitForReview(ctx context.Context, sessionID, tenantID string, submittedAt string) error
	ApproveSession(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt string) error
	RejectSession(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt, rejectionReason string) error
	PublishSession(ctx context.Context, sessionID, tenantID, publisherUserID, publishedAt string) error
	PublishSessions(ctx context.Context, sessionIDs []string, tenantID, publisherUserID, publishedAt string) (int, error)

	// Results
	BatchUpsertResults(ctx context.Context, sessionID, tenantID string, results []LearnerRubricResult) (int, error)
	ListResults(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error)

	// Weight Configs (read-only)
	ListWeightConfigs(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error)

	// Admin Queues
	ListSessionsForAdminQueue(ctx context.Context, tenantID string, query AdminQueuesQuery) ([]SessionAdminView, int, error)

	// Parent-Facing
	ListPublishedResultsForStudent(ctx context.Context, tenantID string, query ParentSessionsQuery) ([]ParentSessionResultView, int, error)
}

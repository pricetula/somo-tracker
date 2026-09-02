package assessments

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors.
var (
	ErrNotFound          = xerrors.NotFound("assessment session not found")
	ErrAlreadyExists     = xerrors.AlreadyExists("assessment session already exists")
	ErrInvalidInput      = xerrors.InvalidInput("invalid assessment input")
	ErrUnauthorized      = xerrors.Unauthorized("unauthorized")
	ErrForbidden         = xerrors.Forbidden("forbidden")
	ErrConflict          = xerrors.Conflict("assessment session conflict")
	ErrInvalidStatus     = xerrors.Conflict("invalid status transition")
	ErrMaxPointsLocked   = xerrors.Conflict("max_points is immutable after student scores exist")
	ErrScoreOutOfRange   = xerrors.Conflict("score out of range")
	ErrNotEnrolledActive = xerrors.Conflict("student not actively enrolled")
)

// Repository defines the contract for assessment session persistence.
type Repository interface {
	List(ctx context.Context, filter SessionListFilter) (*SessionListResult, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error)
	Create(ctx context.Context, params CreateSessionParams) (*AssessmentSession, error)
	Update(ctx context.Context, params UpdateSessionParams) (*AssessmentSession, error)
	Delete(ctx context.Context, id, tenantID, schoolID string) error
	Submit(ctx context.Context, id, tenantID, schoolID, userID string) error
	Approve(ctx context.Context, id, tenantID, schoolID, userID string) error
	Reject(ctx context.Context, id, tenantID, schoolID, userID, comment string) error

	// Scores
	UpsertScores(ctx context.Context, sessionID, tenantID string, scores []ScoreEntryPayload) (int, error)
	ListScores(ctx context.Context, sessionID, tenantID string, page, limit int) (*ScoreListResult, error)

	// Grading scale profiles
	ListGradingScaleProfiles(ctx context.Context, tenantID, schoolID string) ([]map[string]interface{}, error)

	// Rubric outcomes
	UpsertRubricOutcomes(ctx context.Context, sessionID, tenantID string, entries []RubricEntryPayload) (int, error)
	ListRubricOutcomes(ctx context.Context, sessionID, tenantID string) ([]RubricOutcome, error)
}

// AssessmentSession represents a CBC assessment session.
type AssessmentSession struct {
	ID                    string     `json:"id"`
	TenantID              string     `json:"tenant_id"`
	SchoolID              string     `json:"school_id"`
	ClassID               string     `json:"class_id"`
	LearningAreaID        string     `json:"learning_area_id"`
	AcademicTermID        string     `json:"academic_term_id"`
	AcademicYearID        string     `json:"academic_year_id"`
	Name                  string     `json:"name"`
	EvaluationMethod      string     `json:"evaluation_method"` // QUANTITATIVE | RUBRIC
	MaxPoints             *float64   `json:"max_points,omitempty"`
	GradingScaleProfileID *string    `json:"grading_scale_profile_id,omitempty"`
	Status                string     `json:"status"` // DRAFT | PENDING_APPROVAL | PUBLISHED
	RejectionComment      *string    `json:"rejection_comment,omitempty"`
	SubmittedBy           *string    `json:"submitted_by,omitempty"`
	ApprovedBy            *string    `json:"approved_by,omitempty"`
	ScheduledDate         *time.Time `json:"scheduled_date,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	CreatedBy             string     `json:"created_by"`
}

// SessionListFilter holds filtering and pagination params for listing sessions.
type SessionListFilter struct {
	TenantID         string
	SchoolID         string
	ClassID          string
	LearningAreaID   string
	AcademicTermID   string
	Status           string
	EvaluationMethod string
	Page             int
	Limit            int
}

// SessionListResult holds the paginated response for session listing.
type SessionListResult struct {
	Items []AssessmentSession `json:"items"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// CreateSessionPayload is the request body for POST /api/v1/assessments/sessions.
type CreateSessionPayload struct {
	ClassID               string   `json:"class_id"`
	LearningAreaID        string   `json:"learning_area_id"`
	AcademicTermID        string   `json:"academic_term_id"`
	Name                  string   `json:"name"`
	EvaluationMethod      string   `json:"evaluation_method"`
	MaxPoints             *float64 `json:"max_points,omitempty"`
	GradingScaleProfileID *string  `json:"grading_scale_profile_id,omitempty"`
	ScheduledDate         *string  `json:"scheduled_date,omitempty"` // YYYY-MM-DD
}

// CreateSessionParams holds validated params for creating a session.
type CreateSessionParams struct {
	TenantID              string
	SchoolID              string
	AcademicYearID        string
	ClassID               string
	LearningAreaID        string
	AcademicTermID        string
	Name                  string
	EvaluationMethod      string
	MaxPoints             *float64
	GradingScaleProfileID *string
	ScheduledDate         *time.Time
	CreatedBy             string
}

// UpdateSessionPayload is the request body for PUT /api/v1/assessments/sessions/:id.
type UpdateSessionPayload struct {
	Name                  string   `json:"name"`
	EvaluationMethod      string   `json:"evaluation_method"`
	MaxPoints             *float64 `json:"max_points,omitempty"`
	GradingScaleProfileID *string  `json:"grading_scale_profile_id,omitempty"`
	ScheduledDate         *string  `json:"scheduled_date,omitempty"`
}

// UpdateSessionParams holds validated params for updating a session.
type UpdateSessionParams struct {
	ID                    string
	TenantID              string
	SchoolID              string
	Name                  string
	EvaluationMethod      string
	MaxPoints             *float64
	GradingScaleProfileID *string
	ScheduledDate         *time.Time
}

// RejectSessionPayload is the request body for POST /api/v1/assessments/sessions/:id/reject.
type RejectSessionPayload struct {
	Comment string `json:"comment"`
}

// ─── Quantitative Scores ─────────────────────────────────────────────────

// StudentScore represents a single student's score for a session.
type StudentScore struct {
	ID                    string    `json:"id"`
	TenantID              string    `json:"tenant_id"`
	SessionID             string    `json:"session_id"`
	StudentID             string    `json:"student_id"`
	RawScore              *float64  `json:"raw_score,omitempty"`
	CalculatedPercentage  *float64  `json:"calculated_percentage,omitempty"`
	FinalPerformanceLevel *string   `json:"final_performance_level,omitempty"`
	EnrollmentStatus      string    `json:"enrollment_status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// ScoreListResult holds paginated scores for a session.
type ScoreListResult struct {
	Items []StudentScore `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// BatchScorePayload is the request body for POST /api/v1/assessments/sessions/:id/scores.
type BatchScorePayload struct {
	Scores []ScoreEntryPayload `json:"scores"`
}

// ScoreEntryPayload is a single student's score entry.
type ScoreEntryPayload struct {
	StudentID string   `json:"student_id"`
	RawScore  *float64 `json:"raw_score"` // nil means not yet entered
}

// ─── Rubric Outcome Grades ───────────────────────────────────────────────

type RubricOutcome struct {
	ID                     string    `json:"id"`
	TenantID               string    `json:"tenant_id"`
	SessionID              string    `json:"session_id"`
	StudentID              string    `json:"student_id"`
	PerformanceIndicatorID string    `json:"performance_indicator_id"`
	AwardedLevel           string    `json:"awarded_level"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type RubricBatchPayload struct {
	Grading []RubricEntryPayload `json:"grading"`
}

type RubricEntryPayload struct {
	StudentID              string `json:"student_id"`
	PerformanceIndicatorID string `json:"performance_indicator_id"`
	AwardedLevel           string `json:"awarded_level"`
}

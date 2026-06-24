package imports

import (
	"context"
	"errors"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("imports not found")
	ErrAlreadyExists = errors.New("imports already exists")
	ErrInvalidInput  = errors.New("invalid imports input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("imports conflict")
)

// ─── Allowed roles for bulk staff invitation ──────────────────────────────

var AllowedRoles = map[string]bool{
	"SCHOOL_ADMIN": true,
	"NURSE":        true,
	"FINANCE":      true,
	"TEACHER":      true,
}

// ─── ImportJob ────────────────────────────────────────────────────────────

type ImportJob struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	SchoolID          string     `json:"school_id"`
	Role              string     `json:"role"`
	CreatedBy         *string    `json:"created_by,omitempty"`
	Status            string     `json:"status"`
	TotalRecords      int        `json:"total_records"`
	ProcessedRecords  int        `json:"processed_records"`
	SuccessCount      int        `json:"success_count"`
	FailedCount       int        `json:"failed_count"`
	ParentImportJobID *string    `json:"parent_import_job_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// ─── ImportStaffRecord — a single invite row from the client ─────────────

type ImportStaffRecord struct {
	TempID             string `json:"temp_id"`
	Email              string `json:"email"`
	FullName           string `json:"full_name"`
	Phone              string `json:"phone,omitempty"`
	RegistrationNumber string `json:"registration_number,omitempty"`
}

// ─── HTTP request/response types ─────────────────────────────────────────

type StartImportRequest struct {
	Role              string              `json:"role"`
	Records           []ImportStaffRecord `json:"records"`
	ParentImportJobID *string             `json:"parent_import_job_id,omitempty"`
}

type StartImportResponse struct {
	ImportJobID string `json:"import_job_id"`
	Status      string `json:"status"`
	Total       int    `json:"total"`
}

type TrackImportResponse struct {
	Job           ImportJob `json:"job"`
	FailedRecords int       `json:"failed_records"`
}

type ListFailedInvitationsResponse struct {
	Invitations []FailedInvitation `json:"invitations"`
}

type FailedInvitation struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	FullName     *string `json:"full_name,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

type ImportProgressEvent struct {
	Type             string `json:"type"`
	ImportJobID      string `json:"import_job_id"`
	Status           string `json:"status"`
	ProcessedRecords int    `json:"processed_records"`
	SuccessCount     int    `json:"success_count"`
	FailedCount      int    `json:"failed_count"`
	TotalRecords     int    `json:"total_records"`
}

// ─── SSE Event types ─────────────────────────────────────────────────────

const (
	EventProgress = "import_progress"
	EventFinished = "import_finished"
	EventError    = "import_error"
)

// ─── Redis keys ──────────────────────────────────────────────────────────

const (
	RedisChannelProgress = "import:progress:"
	RedisKeyJobStatus    = "import:job:"
)

// ─── Repository interface ───────────────────────────────────────────────

// TaskEnqueuer abstracts the Asynq client dependency.
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// ProgressPublisher abstracts the Redis pub/sub dependency.
type ProgressPublisher interface {
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
}

// SSEPubSubClient abstracts the Redis methods needed by the SSE handler.
type SSEPubSubClient interface {
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
	Ping(ctx context.Context) *redis.StatusCmd
}

// Repository defines the contract for import job persistence.
type Repository interface {
	CreateImportJob(ctx context.Context, job *ImportJob) error
	GetImportJob(ctx context.Context, jobID string) (*ImportJob, error)
	UpdateImportJobStatus(ctx context.Context, id, status string, processed, successCount, failedCount int) error
	SetImportJobStarted(ctx context.Context, id string) error
	SetImportJobCompleted(ctx context.Context, id string, hasErrors bool) error
	SetImportJobFailed(ctx context.Context, id string) error
	BulkInsertInvitations(ctx context.Context, records []ImportStaffRecord, tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string) (map[string]string, []FailedInsertion, error) // returns map[temp_id]invitation_id
	RecordImportFailure(ctx context.Context, jobID, rawPayloadJSON, errMsg string) error
	GetFailedInvitationsByJob(ctx context.Context, jobID string) ([]FailedInvitation, error)
	GetInvitationStytchMemberID(ctx context.Context, id string) (string, error)
	SetInvitationStytchMemberID(ctx context.Context, id, stytchMemberID string) error
	SetInvitationFailed(ctx context.Context, id, errorMessage string, attemptCount int) error
	// BulkRecordImportFailure inserts multiple failure records in a single query.
	BulkRecordImportFailure(ctx context.Context, jobID string, records []ImportStaffRecord, errMsg string) error

	// BulkUpdateInvitations updates existing invitation rows by ID (correction resubmit).
	BulkUpdateInvitations(ctx context.Context, records []ImportStaffRecord, role, jobID string, now time.Time) (int, error)
	GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error)
	GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error)
	// GetPendingStage2Records returns invitations for a job that haven't yet been
	// sent to Stytch (no stytch_member_id). Used to resume Stage 2 on task retry.
	GetPendingStage2Records(ctx context.Context, jobID string) ([]Stage2Record, error)
}

// ─── Constants ───────────────────────────────────────────────────────────

const (
	MaxRecordsPerImport = 5000
	BatchSize           = 200
	StytchConcurrency   = 8
	StytchMaxRetries    = 3
	InvitationTTL       = 7 * 24 * time.Hour
	// TaskTimeout is the maximum wall-clock time for a single Asynq task
	// invocation. 45 minutes accounts for 5000 records at 8 concurrent
	// Stytch workers with occasional retries.
	TaskTimeout = 45 * time.Minute
)

// ─── Error types ─────────────────────────────────────────────────────────

type ErrorBody struct {
	Error   string `json:"code"`
	Message string `json:"message,omitempty"`
}

package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Sentinel Domain Errors
// ============================================================================

// MaxImportBodyBytes returns the maximum request body size (in bytes)
// for the student import endpoint. Sized to accommodate MaxImportRows
// worth of realistic student data with headroom.
func MaxImportBodyBytes() int {
	return 15 * 1024 * 1024 // 15 MB
}

var (
	ErrNotFound         = fmt.Errorf("imports not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists    = fmt.Errorf("imports already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput     = fmt.Errorf("invalid imports input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized     = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden        = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict         = fmt.Errorf("imports conflict: %w", middleware.ErrConflict)
	ErrDuplicateJob     = fmt.Errorf("duplicate import job: %w", middleware.ErrConflict)
	ErrImportInProgress = fmt.Errorf("import already in progress: %w", middleware.ErrConflict)
	ErrJobTypeMismatch  = fmt.Errorf("job type mismatch: %w", middleware.ErrInvalidInput)
	ErrNotCancellable   = fmt.Errorf("import job is not cancellable: %w", middleware.ErrConflict)
)

// ImportInProgressError is returned when a CreateJob request is rejected because
// another job for the same school is already active (processing or cancelling).
// It carries the active job's ID so the frontend can redirect to its progress.
type ImportInProgressError struct {
	ActiveJobID uuid.UUID
}

func (e *ImportInProgressError) Error() string {
	return fmt.Sprintf("import already in progress for school: active job %s", e.ActiveJobID)
}

func (e *ImportInProgressError) Unwrap() error { return ErrImportInProgress }

// ============================================================================
// ImportJobType maps to the import_job_type DB enum.
// ============================================================================

type ImportJobType string

const (
	ImportJobTypeStaffInvite   ImportJobType = "STAFF_INVITE"
	ImportJobTypeStudentImport ImportJobType = "STUDENT_IMPORT"
	ImportJobTypeParentInvite  ImportJobType = "PARENT_INVITE"
)

// ============================================================================
// ImportJobStatus maps to the import_job_status DB enum.
// ============================================================================

type ImportJobStatus string

const (
	ImportJobStatusPending             ImportJobStatus = "pending"
	ImportJobStatusProcessing          ImportJobStatus = "processing"
	ImportJobStatusCompleted           ImportJobStatus = "completed"
	ImportJobStatusFailed              ImportJobStatus = "failed"
	ImportJobStatusCancelled           ImportJobStatus = "cancelled"
	ImportJobStatusCancelling          ImportJobStatus = "cancelling"
	ImportJobStatusCompletedWithErrors ImportJobStatus = "completed_with_errors"
)

// ============================================================================
// ImportStagingStatus maps to the import_staging_status DB enum.
// ============================================================================

type ImportStagingStatus string

const (
	ImportStagingStatusPending   ImportStagingStatus = "pending"
	ImportStagingStatusSucceeded ImportStagingStatus = "succeeded"
	ImportStagingStatusFailed    ImportStagingStatus = "failed"
)

// ImportChunkStatus maps to the import_chunk_status DB enum.

type ImportChunkStatus string

const (
	ImportChunkStatusPending    ImportChunkStatus = "pending"
	ImportChunkStatusProcessing ImportChunkStatus = "processing"
	ImportChunkStatusCompleted  ImportChunkStatus = "completed"
	ImportChunkStatusCancelled  ImportChunkStatus = "cancelled"
)

// ============================================================================
// ImportFailureType maps to the import_failure_type DB enum.
// ============================================================================

type ImportFailureType string

const (
	ImportFailureSchemaValidation         ImportFailureType = "SCHEMA_VALIDATION"
	ImportFailureDBConstraint             ImportFailureType = "DATABASE_CONSTRAINT"
	ImportFailureBusinessRule             ImportFailureType = "BUSINESS_RULE_VIOLATION"
	ImportFailureInvalidClassReference    ImportFailureType = "INVALID_CLASS_REFERENCE"
	ImportFailureDBConstraintViolation    ImportFailureType = "DB_CONSTRAINT_VIOLATION"
	ImportFailureDuplicateAdmissionNumber ImportFailureType = "DUPLICATE_ADMISSION_NUMBER"
	ImportFailureDuplicateUPINumber       ImportFailureType = "DUPLICATE_UPI_NUMBER"
	ImportFailureDuplicateKneCNumber      ImportFailureType = "DUPLICATE_KNEC_NUMBER"

	// Bulk invitation failure types
	ImportFailureDuplicateEmail     ImportFailureType = "DUPLICATE_EMAIL"
	ImportFailureInvalidEmailFormat ImportFailureType = "INVALID_EMAIL_FORMAT"
	ImportFailureStytchAPIError     ImportFailureType = "STYTCH_API_ERROR"
	ImportFailureInviteInsertFailed ImportFailureType = "INVITATION_INSERT_FAILED"
)

// ============================================================================
// Domain Models
// ============================================================================

// Job represents a bulk import job as stored in import_jobs.
type Job struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	SchoolID         uuid.UUID       `json:"school_id"`
	JobType          ImportJobType   `json:"job_type"`
	Role             *string         `json:"role,omitempty"`
	CreatedBy        *uuid.UUID      `json:"created_by,omitempty"`
	Status           ImportJobStatus `json:"status"`
	TotalRecords     int             `json:"total_records"`
	ProcessedRecords int             `json:"processed_records"`
	SuccessCount     int             `json:"success_count"`
	FailedCount      int             `json:"failed_count"`
	IDempotencyKey   *string         `json:"idempotency_key,omitempty"`
	PayloadHash      *string         `json:"payload_hash,omitempty"`
	TotalChunks      int             `json:"total_chunks"`
	ProcessedChunks  int             `json:"processed_chunks"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	LastProgressAt   *time.Time      `json:"last_progress_at,omitempty"`
}

// Chunk describes a contiguous range of staging rows to process as one unit.
type Chunk struct {
	JobID          uuid.UUID `json:"job_id"`
	ChunkIndex     int       `json:"chunk_index"`
	RowNumberStart int       `json:"row_number_start"`
	RowNumberEnd   int       `json:"row_number_end"`
}

// ChunkTaskPayload is the Asynq task payload for a chunk worker.
type ChunkTaskPayload struct {
	JobID          string `json:"job_id"`
	ChunkIndex     int    `json:"chunk_index"`
	RowNumberStart int    `json:"row_number_start"`
	RowNumberEnd   int    `json:"row_number_end"`
}

// RowFailure captures a single failed row during import processing.
type RowFailure struct {
	RowNumber    int               `json:"row_number"`
	RawPayload   json.RawMessage   `json:"raw_payload"`
	ErrorMessage string            `json:"error_message"`
	ErrorType    ImportFailureType `json:"error_type"`
}

// ValidatedRow is an opaque validated row — the engine does not inspect its contents.
// StagingRowID links this row back to the import_job_staging row for idempotent insert tracking.
type ValidatedRow struct {
	RawData      json.RawMessage
	StagingRowID uuid.UUID
}

// ChunkResult is returned by a chunk executor after processing.
type ChunkResult struct {
	Processed int
	Succeeded int
	Failed    int
	Failures  []RowFailure
}

// ProgressEvent is emitted via Redis Pub/Sub.
type ProgressEvent struct {
	JobID            string          `json:"job_id"`
	Status           ImportJobStatus `json:"status"`
	TotalRecords     int             `json:"total_records"`
	ProcessedRecords int             `json:"processed_records"`
	SuccessCount     int             `json:"success_count"`
	FailedCount      int             `json:"failed_count"`
	TotalChunks      int             `json:"total_chunks"`
	ProcessedChunks  int             `json:"processed_chunks"`
}

// ============================================================================
// ImportError — typed error for constraint violations and class reference
// failures, allowing the service layer to extract the failure type without
// inspecting raw driver errors.
// ============================================================================

type ImportError struct {
	Type    ImportFailureType
	Message string
}

func (e *ImportError) Error() string {
	return e.Message
}

// ============================================================================
// Importer Interface — implemented by domain packages (students, etc.)
// ============================================================================

type Importer interface {
	// JobType returns the import_job_type this importer handles.
	JobType() ImportJobType

	// Validate checks each raw row for schema-level correctness.
	// Returns validated rows and any rows that failed validation.
	Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]ValidatedRow, []RowFailure)

	// ResolveReferences enriches the validated rows with any cross-table
	// references that require DB lookups (e.g., grade_level + stream_name → class_id).
	// metadata contains the job-level metadata (e.g., academic_term_id, academic_year_id).
	// Returns resolved rows and any rows that failed resolution.
	// The engine calls this after Validate and before BulkInsert.
	ResolveReferences(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []ValidatedRow) ([]ValidatedRow, []RowFailure)

	// BulkInsert attempts to insert all validated & resolved rows in one multi-row INSERT.
	// Returns the number of rows inserted and any error.
	BulkInsert(ctx context.Context, tx pgx.Tx, rows []ValidatedRow) (inserted int, err error)

	// InsertOne inserts a single validated & resolved row inside a savepoint.
	InsertOne(ctx context.Context, tx pgx.Tx, row ValidatedRow) error
}

// ============================================================================
// Service Repository Interface (what the service needs from the DB layer)
// ============================================================================

type ServiceRepository interface {
	// CreateJob creates a new import_job row and returns the ID.
	// For jobs without an idempotency_key, this is a plain INSERT.
	// For jobs with an idempotency_key, use CreateJobIdempotent instead.
	CreateJob(ctx context.Context, job *Job) (uuid.UUID, error)

	// CreateJobIdempotent inserts a new import_job row with ON CONFLICT DO NOTHING.
	// Returns the newly created Job and true, or the existing Job and false.
	CreateJobIdempotent(ctx context.Context, job *Job, payloadHash string) (*Job, bool, error)

	// GetJobByID returns a single import job.
	GetJobByID(ctx context.Context, jobID uuid.UUID) (*Job, error)

	// InsertStagingRows inserts a batch of staging rows in one multi-row INSERT.
	InsertStagingRows(ctx context.Context, rows []StagingRow) error

	// GetStagingRows returns a range of staging rows for a chunk.
	GetStagingRows(ctx context.Context, jobID uuid.UUID, rowStart, rowEnd int) ([]StagingRow, error)

	// MarkStagingRows updates status and processed_at for a range of rows.
	MarkStagingRows(ctx context.Context, jobID uuid.UUID, rowStart, rowEnd int, status ImportStagingStatus) error

	// InsertFailures inserts multiple failure records.
	InsertFailures(ctx context.Context, jobID uuid.UUID, failures []RowFailure) error

	// ClaimChunk atomically transitions a chunk from 'pending' to 'processing'.
	// Returns the chunk ID if claimed, or uuid.Nil if another worker already claimed it.
	ClaimChunk(ctx context.Context, jobID uuid.UUID, chunkIndex int) (uuid.UUID, error)

	// InsertChunkRows inserts import_job_chunks rows for a job.
	InsertChunkRows(ctx context.Context, chunks []Chunk) error

	// AtomicChunkCompletion atomically transitions a chunk from 'processing' to 'completed'
	// and, only on success, increments job counters. If the chunk is already 'completed'
	// it is a no-op — counters are not re-incremented.
	// Returns the updated status and a boolean indicating terminal state.
	AtomicChunkCompletion(ctx context.Context, jobID uuid.UUID, chunkID uuid.UUID, chunkProcessed, chunkSuccess, chunkFailed int) (ImportJobStatus, bool, error)

	// CancelJob atomically transitions the job from 'processing' to 'cancelling'.
	// Returns the updated job if the transition succeeded, or nil if the job was
	// already in a terminal state or another status (race-safe).
	CancelJob(ctx context.Context, jobID uuid.UUID) (*Job, error)

	// CancelPendingChunk atomically transitions a chunk from 'pending' to 'cancelled'
	// if the parent job's status is 'cancelling'. No-op if the chunk was already
	// claimed or completed.
	CancelPendingChunk(ctx context.Context, jobID uuid.UUID, chunkIndex int) error

	// MarkStagingRowSucceeded sets a single staging row to 'succeeded' within the caller's
	// transaction/savepoint. Used alongside InsertOne for atomic insert+mark.
	MarkStagingRowSucceeded(ctx context.Context, tx pgx.Tx, stagingRowID uuid.UUID) error

	// UpdateJobStatus sets the job status (for job-level fail).
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status ImportJobStatus) error

	// GetJobStagingRowCount returns the number of staging rows for a job.
	GetJobStagingRowCount(ctx context.Context, jobID uuid.UUID) (int, error)

	// GetJobByIDempotencyKey finds a job by tenant_id and idempotency_key.
	GetJobByIDempotencyKey(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (*Job, error)

	// GetFailures returns paginated failure records for a job.
	GetFailures(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]RowFailure, int, error)

	// ListJobs returns paginated import jobs for a school, newest first.
	ListJobs(ctx context.Context, tenantID, schoolID uuid.UUID, limit, offset int) ([]Job, int, error)

	// GetActiveJobBySchoolID returns the currently active job (status 'processing' or
	// 'cancelling') for the given school, or ErrNotFound if none is active.
	GetActiveJobBySchoolID(ctx context.Context, schoolID uuid.UUID) (*Job, error)

	// CleanupStagingData deletes import_job_staging rows belonging to jobs that
	// reached a terminal status before the given cutoff time. Only terminal jobs
	// (completed, completed_with_errors, failed, cancelled) are eligible — jobs
	// still processing or cancelling are never touched. Returns the number of
	// rows deleted. Runs in batches of batchSize.
	CleanupStagingData(ctx context.Context, cutoff time.Time, batchSize int) (int, error)

	// CleanupFailureData is like CleanupStagingData but for import_job_failures.
	CleanupFailureData(ctx context.Context, cutoff time.Time, batchSize int) (int, error)

	// TouchLastProgressAt sets last_progress_at = NOW() for the given job.
	TouchLastProgressAt(ctx context.Context, jobID uuid.UUID) error
}

// StagingRow represents a row in import_job_staging.
type StagingRow struct {
	ID        uuid.UUID           `json:"id"`
	JobID     uuid.UUID           `json:"job_id"`
	TenantID  uuid.UUID           `json:"tenant_id"`
	SchoolID  uuid.UUID           `json:"school_id"`
	RowNumber int                 `json:"row_number"`
	RawData   json.RawMessage     `json:"raw_data"`
	Status    ImportStagingStatus `json:"status"`
}

// ============================================================================
// ImporterRegistry — populated at startup by domain module.go files.
// ============================================================================

// ImporterRegistry holds all registered Importers, keyed by ImportJobType.
var ImporterRegistry = map[ImportJobType]Importer{}

// RegisterImporter registers an Importer for its JobType.
// Panics if a duplicate JobType is registered.
func RegisterImporter(imp Importer) {
	jt := imp.JobType()
	if _, exists := ImporterRegistry[jt]; exists {
		panic(fmt.Sprintf("imports: duplicate importer registration for job type %q", jt))
	}
	ImporterRegistry[jt] = imp
}

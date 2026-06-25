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
	ErrNotFound       = errors.New("imports not found")
	ErrAlreadyExists  = errors.New("imports already exists")
	ErrInvalidInput   = errors.New("invalid imports input")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrConflict       = errors.New("imports conflict")
	ErrImportInFlight = errors.New("import already in progress")
	ErrPayloadTooBig  = errors.New("payload exceeds maximum size")
)

// ─── Student import validation constants ──────────────────────────────────

var ValidTerms = map[string]bool{
	"Term 1": true,
	"Term 2": true,
	"Term 3": true,
}

const (
	MaxStudentsPerImport = 5000
	EnrollmentChunkSize  = 500
)

// ─── Allowed roles for bulk staff invitation ──────────────────────────────

var AllowedRoles = map[string]bool{
	"SCHOOL_ADMIN": true,
	"NURSE":        true,
	"FINANCE":      true,
	"TEACHER":      true,
}

// ─── Student Import Types ────────────────────────────────────────────────

type StudentRecord struct {
	FullName             string `json:"full_name"`
	Gender               string `json:"gender"`
	DateOfBirth          string `json:"date_of_birth"`
	UPINumber            string `json:"upi_number"`
	KNECAssessmentNumber string `json:"knec_assessment_number"`
	CBCStudentParentsID  string `json:"cbc_student_parents_id"`
	ClassID              string `json:"class_id"`
}

type StartStudentImportRequest struct {
	AcademicYear string          `json:"academic_year"`
	Term         string          `json:"term"`
	Students     []StudentRecord `json:"students"`
}

type StartStudentImportResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type StagingRow struct {
	RowNumber int                    `json:"row_number"`
	RawData   map[string]interface{} `json:"raw_data"`
}

type ValidStudent struct {
	RowNumber            int
	FullName             string
	Gender               string
	DateOfBirth          *string
	UPINumber            *string
	KNECAssessmentNumber *string
	CBCStudentParentsID  *string
	ClassID              *string
	RawData              map[string]interface{}
}

type FailedRow struct {
	RawData      map[string]interface{}
	ErrorMessage string
}

type StudentResult struct {
	StudentID string
	ClassID   *string
}

// ProgressFrame is the SSE payload sent via Redis Pub/Sub for student imports.
type ProgressFrame struct {
	Status       string `json:"status"`
	Processed    int    `json:"processed"`
	Total        int    `json:"total"`
	SuccessCount int    `json:"success_count"`
	FailedCount  int    `json:"failed_count"`
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

// ─── StudentImportPayload — Asynq task payload (no row data) ────────────

type StudentImportPayload struct {
	JobID    string `json:"job_id"`
	TenantID string `json:"tenant_id"`
}

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

	// ─── Student Import Methods ─────────────────────────────────────────

	// CheckConcurrentImport returns true if there is already an active import
	// (pending or processing) for this tenant+school.
	CheckConcurrentImport(ctx context.Context, tenantID, schoolID string) (bool, error)

	// BulkInsertStaging inserts a batch of student records into import_job_staging.
	BulkInsertStaging(ctx context.Context, jobID, tenantID, schoolID string, records []StudentRecord, academicYear, term string) error

	// GetStagingRows loads all staging rows for a job, ordered by row_number.
	GetStagingRows(ctx context.Context, jobID string) ([]StagingRow, error)

	// GetValidClasses returns a set of valid class IDs for this tenant+school.
	GetValidClasses(ctx context.Context, tenantID, schoolID string, classIDs []string) (map[string]bool, error)

	// GetValidParentIDs returns a set of valid parent IDs (from cbc_parents) for this tenant.
	GetValidParentIDs(ctx context.Context, tenantID string, parentIDs []string) (map[string]bool, error)

	// BulkInsertStudents inserts students and returns their generated IDs with class_id.
	BulkInsertStudents(ctx context.Context, tenantID string, students []ValidStudent) ([]StudentResult, error)

	// BulkInsertEnrollments inserts enrollment rows for newly created students.
	BulkInsertEnrollments(ctx context.Context, tenantID, schoolID, academicTermID string, enrollments []StudentResult) error

	// ResolveAcademicTerm resolves academic_year name + term name to an academic_term_id UUID.
	ResolveAcademicTerm(ctx context.Context, tenantID, schoolID, academicYear, term string) (string, error)

	// BulkInsertFailures inserts failure rows in a single query (no per-row loop).
	BulkInsertFailures(ctx context.Context, jobID string, failures []FailedRow) error

	// PurgeStaging deletes all staging rows for a completed job.
	PurgeStaging(ctx context.Context, jobID string) error

	// GetImportJobStatus returns the current status and total_records of a job.
	GetImportJobStatus(ctx context.Context, jobID string) (status string, totalRecords int, schoolID string, err error)

	// GetAcademicYears returns all academic years for a tenant+school.
	GetAcademicYears(ctx context.Context, tenantID, schoolID string) ([]AcademicYearRecord, error)

	// GetAcademicPeriods returns all academic periods (terms) for a given academic year.
	GetAcademicPeriods(ctx context.Context, tenantID, schoolID, academicYearID string) ([]AcademicPeriodRecord, error)

	// ─── Lookup Methods for Student Import ──────────────────────────────

	// ListParents returns all active parents for a tenant+school.
	ListParents(ctx context.Context, tenantID, schoolID string) ([]ParentRecord, error)

	// ListClasses returns all active classes for a tenant+school.
	ListClasses(ctx context.Context, tenantID, schoolID string) ([]ClassRecord, error)

	// ListExistingStudents returns all existing students for a tenant+school
	// for duplicate detection during import.
	ListExistingStudents(ctx context.Context, tenantID, schoolID string) ([]ExistingStudentRecord, error)
}

// ─── Parent / Class Lookup Response Types ────────────────────────────

type ParentRecord struct {
	ID       string  `json:"id"`
	FullName string  `json:"full_name"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
}

type ClassRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ─── Existing Student Response Type ────────────────────────────────────

type ExistingStudentRecord struct {
	FullName    string  `json:"full_name"`
	DateOfBirth *string `json:"date_of_birth,omitempty"`
	UPINumber   *string `json:"upi_number,omitempty"`
}

// ─── Academic Year / Period Response Types ─────────────────────────────

type AcademicYearRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	IsCurrent bool   `json:"is_current"`
}

type AcademicPeriodRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TermNumber int    `json:"term_number"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	IsCurrent  bool   `json:"is_current"`
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

const (
	TypeProcessStudents = "task:import:students"
	RedisProgressPrefix = "import:progress:"
	RedisEventsPrefix   = "import:events:"
	RedisProgressTTL    = 86400 // 24 hours
)

// ─── Error types ─────────────────────────────────────────────────────────

type ErrorBody struct {
	Error   string `json:"code"`
	Message string `json:"message,omitempty"`
}

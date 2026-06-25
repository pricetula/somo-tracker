package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// Service contains business logic for the imports domain.
type Service struct {
	repo   Repository
	client TaskEnqueuer
	logger *zap.Logger
	cfg    config.Config
}

// NewService creates a new Service.
func NewService(repo Repository, client TaskEnqueuer, cfg config.Config, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		client: client,
		logger: logger,
		cfg:    cfg,
	}
}

// StytchOrgResolver resolves the Stytch org ID for a tenant.
// This is defined as an interface here so the handler can inject it.
type StytchOrgResolver interface {
	GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error)
}

// StartImport creates an import job and enqueues an Asynq task.
// If parentImportJobID is non-empty, this is a correction resubmit and the
// new job will be linked to the original import job for traceability.
func (s *Service) StartImport(
	ctx context.Context,
	tenantID, schoolID, userID, role string,
	records []ImportStaffRecord,
	resolver StytchOrgResolver,
	parentImportJobID string,
) (*StartImportResponse, error) {
	// Validate role
	if !AllowedRoles[role] {
		return nil, fmt.Errorf("invalid role: must be one of SCHOOL_ADMIN, NURSE, FINANCE")
	}

	// Validate records
	if len(records) == 0 {
		return nil, fmt.Errorf("at least one record is required")
	}
	if len(records) > MaxRecordsPerImport {
		return nil, fmt.Errorf("maximum %d records per import", MaxRecordsPerImport)
	}

	// Validate each record
	for _, rec := range records {
		if rec.Email == "" {
			return nil, fmt.Errorf("email is required for all records")
		}
		if rec.FullName == "" {
			return nil, fmt.Errorf("full_name is required for all records")
		}
		// TSC Number (registration_number) is mandatory for TEACHER role
		if role == "TEACHER" && rec.RegistrationNumber == "" {
			return nil, fmt.Errorf("registration_number (TSC Number) is required for all teacher records")
		}
	}

	// Resolve Stytch org ID
	stytchOrgID, err := resolver.GetTenantStytchOrgID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("resolve stytch org: %w", err)
	}

	// Create import job
	jobID := uuid.New().String()
	now := time.Now().UTC()

	createdBy := userID
	var parentJobID *string
	if parentImportJobID != "" {
		parentJobID = &parentImportJobID
	}
	job := &ImportJob{
		ID:                jobID,
		TenantID:          tenantID,
		SchoolID:          schoolID,
		Role:              role,
		CreatedBy:         &createdBy,
		Status:            "pending",
		TotalRecords:      len(records),
		ProcessedRecords:  0,
		SuccessCount:      0,
		FailedCount:       0,
		ParentImportJobID: parentJobID,
		CreatedAt:         now,
	}

	if err := s.repo.CreateImportJob(ctx, job); err != nil {
		return nil, fmt.Errorf("create import job: %w", err)
	}

	// Enqueue Asynq task
	payload := ProcessImportPayload{
		ImportJobID:       jobID,
		TenantID:          tenantID,
		SchoolID:          schoolID,
		Role:              role,
		Records:           records,
		StytchOrgID:       stytchOrgID,
		BackendURL:        s.cfg.BackendURL,
		ParentImportJobID: parentImportJobID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}

	task := asynq.NewTask(TypeProcessImport, payloadBytes,
		asynq.Queue("critical"),
		asynq.MaxRetry(3),
		asynq.Timeout(TaskTimeout),
	)

	info, err := s.client.Enqueue(task)
	if err != nil {
		s.logger.Error("failed to enqueue import task",
			zap.String("import_job_id", jobID),
			zap.Error(err),
		)
		// Job is created but task failed — still return the job ID
		// The user can retry via the frontend
		return &StartImportResponse{
			ImportJobID: jobID,
			Status:      "enqueue_failed",
			Total:       len(records),
		}, nil
	}

	s.logger.Info("import task enqueued",
		zap.String("import_job_id", jobID),
		zap.String("task_id", info.ID),
	)

	return &StartImportResponse{
		ImportJobID: jobID,
		Status:      "pending",
		Total:       len(records),
	}, nil
}

// GetImportJob retrieves the current state of an import job.
func (s *Service) GetImportJob(ctx context.Context, jobID string) (*TrackImportResponse, error) {
	job, err := s.repo.GetImportJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.GetImportJob: %w", err)
	}

	return &TrackImportResponse{
		Job: *job,
	}, nil
}

// GetFailedInvitations retrieves failed invitation records for a completed job.
func (s *Service) GetFailedInvitations(ctx context.Context, jobID string) (*ListFailedInvitationsResponse, error) {
	invitations, err := s.repo.GetFailedInvitationsByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.GetFailedInvitations: %w", err)
	}
	if invitations == nil {
		invitations = []FailedInvitation{}
	}
	return &ListFailedInvitationsResponse{Invitations: invitations}, nil
}

// ============================================================================
// Student Import Service
// ============================================================================

// StartStudentImport validates, creates a job, persists staging rows, and enqueues.
func (s *Service) StartStudentImport(ctx context.Context, tenantID, schoolID, userID, role string, req *StartStudentImportRequest) (*StartStudentImportResponse, error) {
	// Step 2 — Concurrent-import guard
	hasActive, err := s.repo.CheckConcurrentImport(ctx, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.StartStudentImport: check concurrent: %w", err)
	}
	if hasActive {
		return nil, fmt.Errorf("imports.Service.StartStudentImport: %w", ErrImportInFlight)
	}

	// Step 3 — Create job row
	jobID := uuid.New().String()
	now := time.Now().UTC()
	cb := userID
	job := &ImportJob{
		ID:               jobID,
		TenantID:         tenantID,
		SchoolID:         schoolID,
		Role:             role,
		CreatedBy:        &cb,
		Status:           "pending",
		TotalRecords:     len(req.Students),
		ProcessedRecords: 0,
		SuccessCount:     0,
		FailedCount:      0,
		CreatedAt:        now,
	}

	if err := s.repo.CreateImportJob(ctx, job); err != nil {
		return nil, fmt.Errorf("imports.Service.StartStudentImport: create job: %w", err)
	}

	// Step 4 — Persist staging rows (single bulk insert with academic_year & term stamped)
	if err := s.repo.BulkInsertStaging(ctx, jobID, tenantID, schoolID, req.Students, req.AcademicYear, req.Term); err != nil {
		return nil, fmt.Errorf("imports.Service.StartStudentImport: bulk insert staging: %w", err)
	}

	// Step 5 — Enqueue Asynq task (no row data, just job_id + tenant_id)
	payload := StudentImportPayload{
		JobID:    jobID,
		TenantID: tenantID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.StartStudentImport: marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeProcessStudents, payloadBytes,
		asynq.Queue("critical"),
		asynq.MaxRetry(1),
	)

	if _, err := s.client.Enqueue(task); err != nil {
		s.logger.Error("failed to enqueue student import task",
			zap.String("job_id", jobID),
			zap.Error(err),
		)
		// Job created + staging persisted, but enqueue failed.
		// Return the job ID so the user can check status later.
		return &StartStudentImportResponse{
			JobID:  jobID,
			Status: "enqueue_failed",
		}, nil
	}

	s.logger.Info("student import task enqueued",
		zap.String("job_id", jobID),
		zap.Int("total_records", len(req.Students)),
	)

	return &StartStudentImportResponse{
		JobID:  jobID,
		Status: "pending",
	}, nil
}

// GetImportJob retrieves an import job by ID for the student import track endpoint.
func (s *Service) GetImportJobByID(ctx context.Context, jobID string) (*ImportJob, error) {
	return s.repo.GetImportJob(ctx, jobID)
}

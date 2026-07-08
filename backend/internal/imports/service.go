package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// ============================================================================
// Constants
// ============================================================================

const (
	// ChunkSize is the target number of rows per chunk.
	ChunkSize = 100
)

// ============================================================================
// Service — Import Job State Manager
// ============================================================================

// Service handles import job lifecycle: creation, chunking, completion detection.
type Service struct {
	repo    ServiceRepository
	pool    *pgxpool.Pool
	asynq   *asynq.Client
	beginTx func(ctx context.Context) (pgx.Tx, error) // overridable for tests
}

// NewService creates a new Service.
func NewService(repo ServiceRepository, pools *database.Pools, asynqClient *asynq.Client) *Service {
	s := &Service{
		repo:  repo,
		pool:  pools.PG,
		asynq: asynqClient,
	}
	s.beginTx = func(ctx context.Context) (pgx.Tx, error) {
		return s.pool.Begin(ctx)
	}
	return s
}

// ============================================================================
// CreateJob — creates job + staging rows, enqueues chunk tasks
// ============================================================================

// CreateJobRequest is the input to create an import job.
type CreateJobRequest struct {
	TenantID       uuid.UUID
	SchoolID       uuid.UUID
	JobType        ImportJobType
	CreatedBy      uuid.UUID
	Role           *string
	Rows           []json.RawMessage
	IDempotencyKey *string
	Metadata       json.RawMessage
}

// CreateJobResponse is returned after a successful job creation.
// If IsReplay is true the response reflects a pre-existing job (idempotent
// replay) rather than a newly created one.
type CreateJobResponse struct {
	JobID        uuid.UUID       `json:"job_id"`
	TotalRecords int             `json:"total_records"`
	TotalChunks  int             `json:"total_chunks"`
	Status       ImportJobStatus `json:"status"`
	StreamToken  string          `json:"stream_token"`
	IsReplay     bool            `json:"is_replay"`
}

// computePayloadHash produces a deterministic hash of the row set.
// The result is stable for identical serialized rows and can be used
// to detect payload changes for idempotency checking.
func computePayloadHash(rows []json.RawMessage) string {
	data, err := json.Marshal(rows)
	if err != nil {
		// json.Marshal of []json.RawMessage should never fail under normal
		// conditions, but if it does, fall back to a simple length-based hash
		// so we still have deterministic behaviour.
		data = []byte(fmt.Sprintf("fallback:%d", len(rows)))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// CreateJob creates an import job, writes staging rows, and enqueues chunk tasks.
// When an idempotency_key is provided the method uses INSERT ... ON CONFLICT DO NOTHING
// for concurrent-safe deduplication. See the idempotency rules documented in
// docs/student-import-architecture.md.
func (s *Service) CreateJob(ctx context.Context, req CreateJobRequest) (*CreateJobResponse, error) {
	if len(req.Rows) == 0 {
		return nil, fmt.Errorf("imports.Service.CreateJob: no rows provided: %w", ErrInvalidInput)
	}

	// Calculate chunk partitioning
	totalRecords := len(req.Rows)
	totalChunks := (totalRecords + ChunkSize - 1) / ChunkSize

	// Build metadata
	if req.Metadata == nil {
		req.Metadata = json.RawMessage(`{}`)
	}

	// Compute payload hash for idempotency (stable, deterministic hash of rows)
	payloadHash := computePayloadHash(req.Rows)

	// ── Idempotent path: idempotency_key present ──────────────────────
	if req.IDempotencyKey != nil && *req.IDempotencyKey != "" {
		job := &Job{
			TenantID:       req.TenantID,
			SchoolID:       req.SchoolID,
			JobType:        req.JobType,
			Role:           req.Role,
			CreatedBy:      &req.CreatedBy,
			Status:         ImportJobStatusPending,
			TotalRecords:   totalRecords,
			IDempotencyKey: req.IDempotencyKey,
			TotalChunks:    totalChunks,
			Metadata:       req.Metadata,
		}

		createdJob, isNew, err := s.repo.CreateJobIdempotent(ctx, job, payloadHash)
		if err != nil {
			return nil, fmt.Errorf("imports.Service.CreateJob: idempotent insert: %w", err)
		}

		if isNew {
			// Newly created — proceed with regular flow (staging, chunking, enqueue)
			jobID := createdJob.ID

			// Build and write staging rows
			stagingRows := make([]StagingRow, len(req.Rows))
			for i, raw := range req.Rows {
				stagingRows[i] = StagingRow{
					JobID:     jobID,
					TenantID:  req.TenantID,
					SchoolID:  req.SchoolID,
					RowNumber: i,
					RawData:   raw,
					Status:    ImportStagingStatusPending,
				}
			}

			if err := s.repo.InsertStagingRows(ctx, stagingRows); err != nil {
				return nil, fmt.Errorf("imports.Service.CreateJob: insert staging: %w", err)
			}

			// Enqueue chunk tasks
			if err := s.enqueueChunks(ctx, jobID, totalChunks); err != nil {
				slog.ErrorContext(ctx, "imports.Service.CreateJob: enqueue chunks failed (staging rows written)",
					"job_id", jobID,
					"error", err,
				)
			}

			if err := s.repo.UpdateJobStatus(ctx, jobID, ImportJobStatusProcessing); err != nil {
				return nil, fmt.Errorf("imports.Service.CreateJob: update status: %w", err)
			}

			slog.Info("imports.Service.CreateJob: job created",
				"job_id", jobID,
				"job_type", req.JobType,
				"total_records", totalRecords,
				"total_chunks", totalChunks,
				"idempotency_key", *req.IDempotencyKey,
			)

			return &CreateJobResponse{
				JobID:        jobID,
				TotalRecords: totalRecords,
				TotalChunks:  totalChunks,
				Status:       ImportJobStatusProcessing,
				IsReplay:     false,
			}, nil
		}

		// INSERT conflicted — job with this key already exists
		// Fetch the existing job to compare payload_hash
		existing, err := s.repo.GetJobByIDempotencyKey(ctx, req.TenantID, *req.IDempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("imports.Service.CreateJob: fetch existing: %w", err)
		}

		// Compare payload hashes
		existingHash := ""
		if existing.PayloadHash != nil {
			existingHash = *existing.PayloadHash
		}

		if existingHash == payloadHash {
			slog.Info("imports.Service.CreateJob: idempotent replay",
				"existing_job_id", existing.ID,
				"idempotency_key", *req.IDempotencyKey,
			)
			return &CreateJobResponse{
				JobID:        existing.ID,
				TotalRecords: existing.TotalRecords,
				TotalChunks:  existing.TotalChunks,
				Status:       existing.Status,
				IsReplay:     true,
			}, nil
		}

		// Same key, different payload — this is an error
		slog.Warn("imports.Service.CreateJob: idempotency key reused with different payload",
			"existing_job_id", existing.ID,
			"idempotency_key", *req.IDempotencyKey,
			"existing_hash", existingHash,
			"new_hash", payloadHash,
		)
		return nil, fmt.Errorf("imports.Service.CreateJob: %w", ErrDuplicateJob)
	}

	// ── Non-idempotent path: no idempotency_key ───────────────────────
	job := &Job{
		TenantID:        req.TenantID,
		SchoolID:        req.SchoolID,
		JobType:         req.JobType,
		Role:            req.Role,
		CreatedBy:       &req.CreatedBy,
		Status:          ImportJobStatusPending,
		TotalRecords:    totalRecords,
		IDempotencyKey:  nil,
		TotalChunks:     totalChunks,
		ProcessedChunks: 0,
		Metadata:        req.Metadata,
	}

	jobID, err := s.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.CreateJob: create job: %w", err)
	}
	job.ID = jobID

	// Build staging rows
	stagingRows := make([]StagingRow, len(req.Rows))
	for i, raw := range req.Rows {
		stagingRows[i] = StagingRow{
			JobID:     jobID,
			TenantID:  req.TenantID,
			SchoolID:  req.SchoolID,
			RowNumber: i,
			RawData:   raw,
			Status:    ImportStagingStatusPending,
		}
	}

	// Write all staging rows in one batch
	if err := s.repo.InsertStagingRows(ctx, stagingRows); err != nil {
		return nil, fmt.Errorf("imports.Service.CreateJob: insert staging: %w", err)
	}

	// Enqueue chunk tasks
	if err := s.enqueueChunks(ctx, jobID, totalChunks); err != nil {
		slog.ErrorContext(ctx, "imports.Service.CreateJob: enqueue chunks failed (staging rows written)",
			"job_id", jobID,
			"error", err,
		)
	}

	// Set status to processing
	if err := s.repo.UpdateJobStatus(ctx, jobID, ImportJobStatusProcessing); err != nil {
		return nil, fmt.Errorf("imports.Service.CreateJob: update status: %w", err)
	}

	slog.Info("imports.Service.CreateJob: job created",
		"job_id", jobID,
		"job_type", req.JobType,
		"total_records", totalRecords,
		"total_chunks", totalChunks,
	)

	return &CreateJobResponse{
		JobID:        jobID,
		TotalRecords: totalRecords,
		TotalChunks:  totalChunks,
		Status:       ImportJobStatusProcessing,
		IsReplay:     false,
	}, nil
}

// enqueueChunks creates Asynq tasks for each chunk with deterministic IDs.
func (s *Service) enqueueChunks(ctx context.Context, jobID uuid.UUID, totalChunks int) error {
	for i := 0; i < totalChunks; i++ {
		rowStart := i * ChunkSize
		// Last chunk may be smaller — row count is handled by the executor when it queries

		payload := ChunkTaskPayload{
			JobID:          jobID.String(),
			ChunkIndex:     i,
			RowNumberStart: rowStart,
			RowNumberEnd:   rowStart + ChunkSize,
		}

		task := asynq.NewTask("imports:process_chunk", toBytes(payload),
			asynq.Queue("imports"),
			asynq.MaxRetry(3),
			asynq.Unique(24*time.Hour), // 24h unique TTL
		)

		// Use a deterministic task ID for idempotent redelivery
		taskID := fmt.Sprintf("import:%s:chunk:%d", jobID, i)

		if _, err := s.asynq.Enqueue(task, asynq.TaskID(taskID)); err != nil {
			return fmt.Errorf("enqueue chunk %d: %w", i, err)
		}
	}
	return nil
}

// ============================================================================
// ProcessChunk — executed by the Asynq worker
// ============================================================================

// ProcessChunk handles a single chunk execution. This is the core of the
// chunk executor.
func (s *Service) ProcessChunk(ctx context.Context, payload ChunkTaskPayload) error {
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: parse job id: %w", err)
	}

	// Look up the job to find its type and tenant/school
	job, err := s.repo.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: get job: %w", err)
	}

	// Find the right Importer
	imp, ok := ImporterRegistry[job.JobType]
	if !ok {
		return fmt.Errorf("imports.Service.ProcessChunk: no importer registered for job type %q", job.JobType)
	}

	// Load staging rows for this chunk
	rows, err := s.repo.GetStagingRows(ctx, jobID, payload.RowNumberStart, payload.RowNumberEnd)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: get staging rows: %w", err)
	}

	if len(rows) == 0 {
		// No rows to process — mark chunk as completed with 0 results
		_, _, err := s.repo.AtomicChunkCompletion(ctx, jobID, 0, 0, 0)
		return err
	}

	// Extract raw JSON from staging rows
	rawRows := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		rawRows[i] = row.RawData
	}

	// Step 1: Validate all rows
	validated, validationFailures := imp.Validate(ctx, job.TenantID, job.SchoolID, rawRows)

	// Map validation failures to row numbers (using staging row indices)
	var allFailures []RowFailure
	for _, f := range validationFailures {
		// Find the actual row_number in staging
		rowNum := f.RowNumber
		if rowNum < len(rows) {
			f.RowNumber = rows[rowNum].RowNumber
		}
		allFailures = append(allFailures, f)
	}

	// Step 1b: Resolve cross-table references (grade/stream → class_id, etc.)
	resolved, resolveFailures := imp.ResolveReferences(ctx, job.TenantID, job.SchoolID, job.Metadata, validated)
	allFailures = append(allFailures, resolveFailures...)

	// Step 2: Attempt bulk insert
	successCount := 0
	dbFailures := 0

	if len(resolved) > 0 {
		// Start a transaction for this chunk
		tx, err := s.beginTx(ctx)
		if err != nil {
			return fmt.Errorf("imports.Service.ProcessChunk: begin tx: %w", err)
		}
		// Use deferred rollback pattern
		defer func() {
			if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
				slog.ErrorContext(ctx, "imports.Service.ProcessChunk: deferred rollback",
					"job_id", jobID,
					"error", err,
				)
			}
		}()

		inserted, err := imp.BulkInsert(ctx, tx, resolved)
		if err != nil {
			// Bulk insert failed — fall back to per-row savepoint inserts
			dbFailures, err = s.insertWithSavepoints(ctx, tx, imp, resolved, rows, &allFailures)
			if err != nil {
				return fmt.Errorf("imports.Service.ProcessChunk: savepoint fallback: %w", err)
			}
			successCount = len(resolved) - dbFailures
		} else {
			successCount = inserted
		}

		// Commit the transaction
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("imports.Service.ProcessChunk: commit tx: %w", err)
		}
	}

	// Step 3: Write failures
	if len(allFailures) > 0 {
		if err := s.repo.InsertFailures(ctx, jobID, allFailures); err != nil {
			return fmt.Errorf("imports.Service.ProcessChunk: insert failures: %w", err)
		}
	}

	// Step 4: Mark staging rows as processed
	if err := s.repo.MarkStagingRows(ctx, jobID, payload.RowNumberStart, payload.RowNumberEnd,
		ImportStagingStatusSucceeded); err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: mark staging rows: %w", err)
	}

	// Step 5: Atomic chunk completion
	chunkProcessed := len(rows)
	newStatus, _, err := s.repo.AtomicChunkCompletion(ctx, jobID,
		chunkProcessed, successCount, len(allFailures))
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: atomic completion: %w", err)
	}

	slog.InfoContext(ctx, "imports.Service.ProcessChunk: chunk completed",
		"job_id", jobID,
		"chunk", payload.ChunkIndex,
		"processed", chunkProcessed,
		"succeeded", successCount,
		"failed", len(allFailures),
		"status", newStatus,
	)

	return nil
}

// insertWithSavepoints falls back to per-row inserts when bulk insert fails.
// Returns the number of rows that failed.
func (s *Service) insertWithSavepoints(
	ctx context.Context,
	tx pgx.Tx,
	imp Importer,
	validated []ValidatedRow,
	stagingRows []StagingRow,
	failures *[]RowFailure,
) (int, error) {
	failedCount := 0

	for i, vRow := range validated {
		_, err := tx.Exec(ctx, "SAVEPOINT import_row")
		if err != nil {
			return 0, fmt.Errorf("savepoint: %w", err)
		}

		err = imp.InsertOne(ctx, tx, vRow)
		if err != nil {
			// Check for duplicate UPI, unique constraint, etc.
			*failures = append(*failures, RowFailure{
				RowNumber:    stagingRows[i].RowNumber,
				RawPayload:   vRow.RawData,
				ErrorMessage: err.Error(),
				ErrorType:    ImportFailureDBConstraint,
			})
			failedCount++

			_, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT import_row")
			if rbErr != nil {
				return 0, fmt.Errorf("rollback savepoint: %w", rbErr)
			}
		} else {
			_, relErr := tx.Exec(ctx, "RELEASE SAVEPOINT import_row")
			if relErr != nil {
				return 0, fmt.Errorf("release savepoint: %w", relErr)
			}
		}
	}

	return failedCount, nil
}

// ============================================================================
// GetJob — retrieves current job state
// ============================================================================

func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	job, err := s.repo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.GetJob: %w", err)
	}
	return job, nil
}

// ============================================================================
// GetFailures — retrieves paginated failure records for a job
// ============================================================================

func (s *Service) GetFailures(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]RowFailure, int, error) {
	if limit <= 0 || limit > 5000 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetFailures(ctx, jobID, limit, offset)
}

// ============================================================================
// helpers
// ============================================================================

func toBytes(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func isNotFound(err error) bool {
	return err != nil && (err.Error() == "no rows in result set" ||
		errors.Is(err, ErrNotFound))
}

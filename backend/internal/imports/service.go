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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// ============================================================================
// Constants
// ============================================================================

const (
	// ChunkSize is the target number of rows per chunk.
	ChunkSize = 100

	// MaxImportRows is the maximum number of rows allowed in a single
	// import request. This limit prevents one import from consuming
	// excessive memory, Asynq workers, and DB connections.
	// 5000 rows at ~1KB each ≈ 5MB raw payload, well under the 15MB
	// request body limit. Schools with larger enrollment should split
	// imports into multiple files.
	MaxImportRows = 5000

	// maxImportBodyBytes is the maximum request body size (in bytes)
	// for the import endpoint. Set to 15MB with generous headroom:
	// 5000 rows × ~2KB/row ≈ 10MB + 50% margin.
	// maxImportBodyBytes = 15 * 1024 * 1024 // 15 MB
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
// CreateJob — creates job + staging rows + chunk rows + enqueues chunk tasks
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

// CreateJob creates an import job, writes staging rows, chunk rows, and enqueues chunk tasks.
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

	// Build chunk definitions (used in both paths)
	chunks := make([]Chunk, totalChunks)
	for i := 0; i < totalChunks; i++ {
		chunks[i] = Chunk{
			ChunkIndex:     i,
			RowNumberStart: i * ChunkSize,
			RowNumberEnd:   (i * ChunkSize) + ChunkSize,
		}
	}
	// Ensure the last chunk's RowNumberEnd doesn't exceed max int
	if totalChunks > 0 {
		last := &chunks[totalChunks-1]
		if last.RowNumberEnd > totalRecords {
			last.RowNumberEnd = totalRecords
		}
	}

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
			// Check for unique constraint violation from the partial index
			// uq_import_jobs_one_active_per_school (a different job is already
			// active for this school).
			if isUniqueConstraintViolation(err) {
				activeJob, lookupErr := s.repo.GetActiveJobBySchoolID(ctx, req.SchoolID)
				if lookupErr != nil {
					return nil, fmt.Errorf("imports.Service.CreateJob: idempotent conflict but cannot look up active job: %w", lookupErr)
				}
				return nil, &ImportInProgressError{ActiveJobID: activeJob.ID}
			}
			return nil, fmt.Errorf("imports.Service.CreateJob: idempotent insert: %w", err)
		}

		if isNew {
			// Newly created — proceed with regular flow (staging, chunking, enqueue)
			jobID := createdJob.ID
			// Assign job ID to chunks
			for i := range chunks {
				chunks[i].JobID = jobID
			}

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

			// Write chunk rows
			if err := s.repo.InsertChunkRows(ctx, chunks); err != nil {
				return nil, fmt.Errorf("imports.Service.CreateJob: insert chunks: %w", err)
			}

			// Enqueue chunk tasks
			if err := s.enqueueChunks(ctx, jobID, totalChunks); err != nil {
				slog.ErrorContext(ctx, "imports.Service.CreateJob: enqueue chunks failed (staging rows written)",
					"job_id", jobID,
					"error", err,
				)
			}

			if err := s.repo.UpdateJobStatus(ctx, jobID, ImportJobStatusProcessing); err != nil {
				// Check for unique constraint violation from the partial index
				// uq_import_jobs_one_active_per_school — another job was activated
				// for this school between our insert and this UPDATE.
				if isUniqueConstraintViolation(err) {
					activeJob, lookupErr := s.repo.GetActiveJobBySchoolID(ctx, req.SchoolID)
					if lookupErr != nil {
						return nil, fmt.Errorf("imports.Service.CreateJob: update status conflict but cannot look up active job: %w", lookupErr)
					}
					return nil, &ImportInProgressError{ActiveJobID: activeJob.ID}
				}
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
		// Check for unique constraint violation from the partial index
		// uq_import_jobs_one_active_per_school. Rely on the DB constraint
		// as the source of truth (not a check-then-insert pattern).
		if isUniqueConstraintViolation(err) {
			activeJob, lookupErr := s.repo.GetActiveJobBySchoolID(ctx, req.SchoolID)
			if lookupErr != nil {
				return nil, fmt.Errorf("imports.Service.CreateJob: conflict but cannot look up active job: %w", lookupErr)
			}
			return nil, &ImportInProgressError{ActiveJobID: activeJob.ID}
		}
		return nil, fmt.Errorf("imports.Service.CreateJob: create job: %w", err)
	}
	job.ID = jobID

	// Assign job ID to chunks
	for i := range chunks {
		chunks[i].JobID = jobID
	}

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

	// Write chunk rows
	if err := s.repo.InsertChunkRows(ctx, chunks); err != nil {
		return nil, fmt.Errorf("imports.Service.CreateJob: insert chunks: %w", err)
	}

	// Enqueue chunk tasks
	if err := s.enqueueChunks(ctx, jobID, totalChunks); err != nil {
		slog.ErrorContext(ctx, "imports.Service.CreateJob: enqueue chunks failed (staging rows written)",
			"job_id", jobID,
			"error", err,
		)
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
// CancelJob — requests cancellation of an import job
// ============================================================================

// CancelJob atomically transitions a job from 'processing' to 'cancelling'.
// Returns the updated job with the new status. Returns ErrNotCancellable if
// the job is not in a cancellable state (already terminal or not yet started).
// The caller must verify tenant/school access before calling this method.
func (s *Service) CancelJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	job, err := s.repo.CancelJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.CancelJob: %w", err)
	}

	slog.InfoContext(ctx, "imports.Service.CancelJob: job cancellation requested",
		"job_id", jobID,
		"status", job.Status,
	)
	return job, nil
}

// ============================================================================
// ProcessChunk — executed by the Asynq worker (idempotent against redelivery)
// ============================================================================

// ProcessChunk handles a single chunk execution. This is the core of the
// chunk executor. It is safe against redelivery: chunk claiming, pending-only
// row filtering, atomic insert+mark, and idempotent chunk completion ensure
// a chunk is never processed twice in a way that duplicates data.
//
// Cancellation-aware: before claiming a chunk, checks the parent job status.
// If the job is 'cancelling', the chunk is marked 'cancelled' instead of
// claimed, and no rows are processed.
func (s *Service) ProcessChunk(ctx context.Context, payload ChunkTaskPayload) error {
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: parse job id: %w", err)
	}

	// ── Step 0: Check job status for cancellation ─────────────────────────
	// Before attempting to claim the chunk, check if the parent job has been
	// cancelled. If so, mark this chunk as 'cancelled' and exit without
	// processing any rows. This is a cooperative cancellation — chunks already
	// claimed/processing are allowed to finish normally.
	job, err := s.repo.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: get job: %w", err)
	}
	if job.Status == ImportJobStatusCancelling {
		// Atomically cancel this pending chunk
		if err := s.repo.CancelPendingChunk(ctx, jobID, payload.ChunkIndex); err != nil {
			return fmt.Errorf("imports.Service.ProcessChunk: cancel pending chunk: %w", err)
		}
		slog.InfoContext(ctx, "imports.Service.ProcessChunk: chunk cancelled (job cancelled)",
			"job_id", jobID,
			"chunk", payload.ChunkIndex,
		)
		return nil
	}

	// ── Step 1: Atomically claim the chunk ────────────────────────────────
	chunkID, err := s.repo.ClaimChunk(ctx, jobID, payload.ChunkIndex)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: claim chunk: %w", err)
	}
	if chunkID == uuid.Nil {
		// Another attempt already claimed or completed this chunk — safe to skip
		slog.InfoContext(ctx, "imports.Service.ProcessChunk: chunk already claimed or completed, skipping",
			"job_id", jobID,
			"chunk", payload.ChunkIndex,
		)
		return nil
	}

	// Look up the job again (we already have it from the cancellation check above,
	// but re-fetch for clarity in case the status changed between steps)
	job, err = s.repo.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: get job: %w", err)
	}

	// Find the right Importer
	imp, ok := ImporterRegistry[job.JobType]
	if !ok {
		return fmt.Errorf("imports.Service.ProcessChunk: no importer registered for job type %q", job.JobType)
	}

	// ── Step 2: Load staging rows (pending only) ─────────────────────────
	rows, err := s.repo.GetStagingRows(ctx, jobID, payload.RowNumberStart, payload.RowNumberEnd)
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: get staging rows: %w", err)
	}

	if len(rows) == 0 {
		// No rows to process — mark chunk as completed with 0 results
		_, _, err := s.repo.AtomicChunkCompletion(ctx, jobID, chunkID, 0, 0, 0)
		return err
	}

	// Extract raw JSON from staging rows
	rawRows := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		rawRows[i] = row.RawData
	}

	// Step 3: Validate all rows
	validated, validationFailures := imp.Validate(ctx, job.TenantID, job.SchoolID, rawRows)

	// Attach staging row IDs to validated rows, and map validation failures
	var allFailures []RowFailure
	for _, f := range validationFailures {
		rowNum := f.RowNumber
		if rowNum < len(rows) {
			f.RowNumber = rows[rowNum].RowNumber
		}
		allFailures = append(allFailures, f)
	}
	for i := range validated {
		if i < len(rows) {
			validated[i].StagingRowID = rows[i].ID
		}
	}

	// Step 4: Resolve cross-table references
	resolved, resolveFailures := imp.ResolveReferences(ctx, job.TenantID, job.SchoolID, job.Metadata, validated)
	allFailures = append(allFailures, resolveFailures...)

	// Step 5: Attempt bulk insert with savepoint fallback
	successCount := 0
	dbFailures := 0

	if len(resolved) > 0 {
		tx, err := s.beginTx(ctx)
		if err != nil {
			return fmt.Errorf("imports.Service.ProcessChunk: begin tx: %w", err)
		}
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

	// Step 6: Write failures
	if len(allFailures) > 0 {
		if err := s.repo.InsertFailures(ctx, jobID, allFailures); err != nil {
			return fmt.Errorf("imports.Service.ProcessChunk: insert failures: %w", err)
		}
	}

	// Step 7: Atomic chunk completion (idempotent — guarded by chunk status)
	// Use the total expected row count for the chunk (not just pending rows)
	// so that processed_records reflects the full chunk, including rows that
	// were already committed by a prior crashed attempt.
	chunkTotal := payload.RowNumberEnd - payload.RowNumberStart
	// Clamp to the job's total records for the last (partial) chunk
	if job.TotalRecords > 0 && chunkTotal > job.TotalRecords-payload.RowNumberStart {
		chunkTotal = job.TotalRecords - payload.RowNumberStart
	}
	newStatus, _, err := s.repo.AtomicChunkCompletion(ctx, jobID, chunkID,
		chunkTotal, successCount, len(allFailures))
	if err != nil {
		return fmt.Errorf("imports.Service.ProcessChunk: atomic completion: %w", err)
	}

	slog.InfoContext(ctx, "imports.Service.ProcessChunk: chunk completed",
		"job_id", jobID,
		"chunk", payload.ChunkIndex,
		"processed", chunkTotal,
		"succeeded", successCount,
		"failed", len(allFailures),
		"status", newStatus,
	)

	return nil
}

// insertWithSavepoints falls back to per-row inserts when bulk insert fails.
// Each savepoint atomically inserts the student + marks the staging row 'succeeded'.
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
			// Use the typed error from InsertOne if available, otherwise
			// fall back to a generic DB constraint failure.
			var impErr *ImportError
			errorType := ImportFailureDBConstraint
			errorMsg := err.Error()
			if errors.As(err, &impErr) {
				errorType = impErr.Type
				errorMsg = impErr.Message
			}

			*failures = append(*failures, RowFailure{
				RowNumber:    stagingRows[i].RowNumber,
				RawPayload:   vRow.RawData,
				ErrorMessage: errorMsg,
				ErrorType:    errorType,
			})
			failedCount++

			_, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT import_row")
			if rbErr != nil {
				return 0, fmt.Errorf("rollback savepoint: %w", rbErr)
			}
		} else {
			// Mark staging row as succeeded within the same savepoint
			// (atomic insert + mark — no code path where insert commits but staging is left pending)
			if err := s.repo.MarkStagingRowSucceeded(ctx, tx, vRow.StagingRowID); err != nil {
				return 0, fmt.Errorf("mark staging row succeeded: %w", err)
			}

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

// isUniqueConstraintViolation checks if an error is a Postgres unique constraint
// violation (SQLSTATE 23505). Used to detect conflicts on the partial unique index
// uq_import_jobs_one_active_per_school without parsing error strings.
func isUniqueConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// ============================================================================
// GetActiveJobBySchool — returns the currently active job for a school
// ============================================================================

func (s *Service) GetActiveJobBySchool(ctx context.Context, schoolID uuid.UUID) (*Job, error) {
	job, err := s.repo.GetActiveJobBySchoolID(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("imports.Service.GetActiveJobBySchool: %w", err)
	}
	return job, nil
}

// ============================================================================
// helpers
// ============================================================================

func toBytes(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

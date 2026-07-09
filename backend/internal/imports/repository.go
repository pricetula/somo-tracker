package imports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// ============================================================================
// PgRepository — Postgres-backed implementation of ServiceRepository
// ============================================================================

type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// Compile-time interface checks.
var _ ServiceRepository = (*PgRepository)(nil)

// ─── CreateJob ─────────────────────────────────────────────────────────────

func (r *PgRepository) CreateJob(ctx context.Context, job *Job) (uuid.UUID, error) {
	query := `
		INSERT INTO import_jobs (tenant_id, school_id, job_type, role, created_by, status,
		                         total_records, idempotency_key, payload_hash, total_chunks, metadata)
		VALUES ($1, $2, $3, $4, $5, 'processing'::import_job_status, $6, $7, $8, $9, $10)
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		job.TenantID,
		job.SchoolID,
		job.JobType,
		job.Role,
		job.CreatedBy,
		job.TotalRecords,
		job.IDempotencyKey,
		job.PayloadHash,
		job.TotalChunks,
		job.Metadata,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("imports.Repository.CreateJob: %w", err)
	}
	return id, nil
}

// ─── CreateJobIdempotent ─────────────────────────────────────────────────────

// CreateJobIdempotent inserts a new import_job row or returns the existing one
// when a conflict on (tenant_id, school_id, idempotency_key) occurs.
// bool is true when a new row was created, false when the existing one is returned.
func (r *PgRepository) CreateJobIdempotent(ctx context.Context, job *Job, payloadHash string) (*Job, bool, error) {
	query := `
		INSERT INTO import_jobs (tenant_id, school_id, job_type, role, created_by, status,
		                         total_records, idempotency_key, payload_hash, total_chunks, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL
		DO NOTHING
		RETURNING id, tenant_id, school_id, job_type, role, created_by, status,
		          total_records, processed_records, success_count, failed_count,
		          idempotency_key, payload_hash, total_chunks, processed_chunks, metadata,
		          created_at, started_at, completed_at, last_progress_at
	`
	var j Job
	var role, idempotencyKey, payloadHashOut *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		job.TenantID,
		job.SchoolID,
		job.JobType,
		job.Role,
		job.CreatedBy,
		job.Status,
		job.TotalRecords,
		job.IDempotencyKey,
		payloadHash,
		job.TotalChunks,
		job.Metadata,
	).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKey, &payloadHashOut, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.LastProgressAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Conflict — row already exists, caller should fetch by idempotency key
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("imports.Repository.CreateJobIdempotent: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKey
	j.PayloadHash = payloadHashOut
	j.CreatedBy = createdBy
	return &j, true, nil
}

// ─── GetJobByID ────────────────────────────────────────────────────────────

func (r *PgRepository) GetJobByID(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		SELECT id, tenant_id, school_id, job_type, role, created_by, status,
		       total_records, processed_records, success_count, failed_count,
		       idempotency_key, payload_hash, total_chunks, processed_chunks, metadata,
		       created_at, started_at, completed_at, last_progress_at
		FROM import_jobs
		WHERE id = $1
	`
	var j Job
	var role, idempotencyKey, payloadHash *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKey, &payloadHash, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.LastProgressAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("imports.Repository.GetJobByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("imports.Repository.GetJobByID: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKey
	j.PayloadHash = payloadHash
	j.CreatedBy = createdBy
	return &j, nil
}

// ─── InsertStagingRows ─────────────────────────────────────────────────────

func (r *PgRepository) InsertStagingRows(ctx context.Context, rows []StagingRow) error {
	if len(rows) == 0 {
		return nil
	}

	// Build a batch insert with value placeholders
	valueStrs := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*5)

	for i, row := range rows {
		base := i * 5
		valueStrs[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5)
		args = append(args, row.JobID, row.TenantID, row.SchoolID, row.RowNumber, row.RawData)
	}

	query := fmt.Sprintf(`
		INSERT INTO import_job_staging (job_id, tenant_id, school_id, row_number, raw_data)
		VALUES %s
	`, strings.Join(valueStrs, ", "))

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.InsertStagingRows: %w", err)
	}
	return nil
}

// ─── InsertChunkRows ──────────────────────────────────────────────────────

func (r *PgRepository) InsertChunkRows(ctx context.Context, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	valueStrs := make([]string, len(chunks))
	args := make([]interface{}, 0, len(chunks)*4)

	for i, c := range chunks {
		base := i * 4
		valueStrs[i] = fmt.Sprintf("($%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4)
		args = append(args, c.JobID, c.ChunkIndex, c.RowNumberStart, c.RowNumberEnd)
	}

	query := fmt.Sprintf(`
		INSERT INTO import_job_chunks (job_id, chunk_index, row_start, row_end)
		VALUES %s
	`, strings.Join(valueStrs, ", "))

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.InsertChunkRows: %w", err)
	}
	return nil
}

// ─── GetStagingRows ────────────────────────────────────────────────────────

func (r *PgRepository) GetStagingRows(ctx context.Context, jobID uuid.UUID, rowStart, rowEnd int) ([]StagingRow, error) {
	// Filter to 'pending' only — rows already succeeded/failed from a prior
	// partial attempt are skipped for redelivery safety.
	query := `
		SELECT id, job_id, tenant_id, school_id, row_number, raw_data, status
		FROM import_job_staging
		WHERE job_id = $1 AND row_number >= $2 AND row_number < $3 AND status = 'pending'
		ORDER BY row_number ASC
	`
	rows, err := r.pool.Query(ctx, query, jobID, rowStart, rowEnd)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.GetStagingRows: %w", err)
	}
	defer rows.Close()

	var result []StagingRow
	for rows.Next() {
		var sr StagingRow
		var status string
		if err := rows.Scan(&sr.ID, &sr.JobID, &sr.TenantID, &sr.SchoolID, &sr.RowNumber, &sr.RawData, &status); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetStagingRows: scan: %w", err)
		}
		sr.Status = ImportStagingStatus(status)
		result = append(result, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.GetStagingRows: rows: %w", err)
	}
	if result == nil {
		result = []StagingRow{}
	}
	return result, nil
}

// ─── MarkStagingRows ───────────────────────────────────────────────────────

func (r *PgRepository) MarkStagingRows(ctx context.Context, jobID uuid.UUID, rowStart, rowEnd int, status ImportStagingStatus) error {
	query := `
		UPDATE import_job_staging
		SET status = $1, processed_at = NOW()
		WHERE job_id = $2 AND row_number >= $3 AND row_number < $4
	`
	_, err := r.pool.Exec(ctx, query, string(status), jobID, rowStart, rowEnd)
	if err != nil {
		return fmt.Errorf("imports.Repository.MarkStagingRows: %w", err)
	}
	return nil
}

// ─── MarkStagingRowSucceeded ────────────────────────────────────────────────

func (r *PgRepository) MarkStagingRowSucceeded(ctx context.Context, tx pgx.Tx, stagingRowID uuid.UUID) error {
	query := `
		UPDATE import_job_staging
		SET status = 'succeeded', processed_at = NOW()
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, stagingRowID)
	if err != nil {
		return fmt.Errorf("imports.Repository.MarkStagingRowSucceeded: %w", err)
	}
	return nil
}

// ─── InsertFailures ────────────────────────────────────────────────────────

func (r *PgRepository) InsertFailures(ctx context.Context, jobID uuid.UUID, failures []RowFailure) error {
	if len(failures) == 0 {
		return nil
	}

	valueStrs := make([]string, len(failures))
	args := make([]interface{}, 0, len(failures)*5)

	for i, f := range failures {
		base := i * 5
		valueStrs[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5)
		args = append(args, jobID, f.RawPayload, f.ErrorMessage, string(f.ErrorType), f.RowNumber)
	}

	query := fmt.Sprintf(`
		INSERT INTO import_job_failures (import_job_id, raw_payload, error_message, error_type, row_number)
		VALUES %s
	`, strings.Join(valueStrs, ", "))

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.InsertFailures: %w", err)
	}
	return nil
}

// ─── CancelJob ────────────────────────────────────────────────────────────

func (r *PgRepository) CancelJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		UPDATE import_jobs
		SET status = 'cancelling'::import_job_status
		WHERE id = $1 AND status = 'processing'::import_job_status
		RETURNING id, tenant_id, school_id, job_type, role, created_by, status,
		          total_records, processed_records, success_count, failed_count,
		          idempotency_key, payload_hash, total_chunks, processed_chunks, metadata,
		          created_at, started_at, completed_at, last_progress_at
	`
	var j Job
	var role, idempotencyKey, payloadHash *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKey, &payloadHash, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.LastProgressAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("imports.Repository.CancelJob: %w", ErrNotCancellable)
		}
		return nil, fmt.Errorf("imports.Repository.CancelJob: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKey
	j.PayloadHash = payloadHash
	j.CreatedBy = createdBy
	return &j, nil
}

// ─── CancelPendingChunk ────────────────────────────────────────────────────

func (r *PgRepository) CancelPendingChunk(ctx context.Context, jobID uuid.UUID, chunkIndex int) error {
	query := `
		UPDATE import_job_chunks
		SET status = 'cancelled'
		WHERE job_id = $1 AND chunk_index = $2 AND status = 'pending'
	`
	_, err := r.pool.Exec(ctx, query, jobID, chunkIndex)
	if err != nil {
		return fmt.Errorf("imports.Repository.CancelPendingChunk: %w", err)
	}
	return nil
}

// ─── ClaimChunk ─────────────────────────────────────────────────────────────

func (r *PgRepository) ClaimChunk(ctx context.Context, jobID uuid.UUID, chunkIndex int) (uuid.UUID, error) {
	query := `
		UPDATE import_job_chunks
		SET status = 'processing', claimed_at = NOW()
		WHERE job_id = $1 AND chunk_index = $2 AND status = 'pending'
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query, jobID, chunkIndex).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Chunk already claimed or completed — not an error, just no-op
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("imports.Repository.ClaimChunk: %w", err)
	}
	return id, nil
}

// ─── AtomicChunkCompletion ─────────────────────────────────────────────────

func (r *PgRepository) AtomicChunkCompletion(ctx context.Context, jobID uuid.UUID, chunkID uuid.UUID, chunkProcessed, chunkSuccess, chunkFailed int) (ImportJobStatus, bool, error) {
	// First: transition the chunk from 'processing' to 'completed'.
	// Only if that succeeds, increment job counters.
	// If the chunk is already 'completed', this is a no-op.
	query := `
		WITH chunk_update AS (
			UPDATE import_job_chunks
			SET status = 'completed', completed_at = NOW()
			WHERE id = $1 AND status = 'processing'
			RETURNING id, job_id
		)
		UPDATE import_jobs
		SET
		    processed_records = processed_records + $2,
		    processed_chunks  = processed_chunks + 1,
		    success_count     = success_count + $3,
		    failed_count      = failed_count + $4,
		    last_progress_at  = NOW(),
		    status = CASE
		        -- If the job is being cancelled and all chunks are done, transition to 'cancelled'
		        WHEN processed_chunks + 1 = total_chunks AND import_jobs.status = 'cancelling'::import_job_status THEN 'cancelled'::import_job_status
		        WHEN processed_chunks + 1 = total_chunks AND failed_count + $4 = 0 THEN 'completed'::import_job_status
		        WHEN processed_chunks + 1 = total_chunks THEN 'completed_with_errors'::import_job_status
		        ELSE import_jobs.status
		    END,
		    completed_at = CASE WHEN processed_chunks + 1 = total_chunks THEN NOW() ELSE completed_at END
		FROM chunk_update
		WHERE import_jobs.id = chunk_update.job_id
		RETURNING import_jobs.status, import_jobs.processed_chunks, import_jobs.total_chunks
	`
	var status string
	var processedChunks, totalChunks int
	err := r.pool.QueryRow(ctx, query, chunkID, chunkProcessed, chunkSuccess, chunkFailed).Scan(&status, &processedChunks, &totalChunks)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Chunk is already completed — no-op. Return current state by fetching job.
			job, fetchErr := r.GetJobByID(ctx, jobID)
			if fetchErr != nil {
				return "", false, fmt.Errorf("imports.Repository.AtomicChunkCompletion: fetch after no-op: %w", fetchErr)
			}
			isTerminal := job.ProcessedChunks >= job.TotalChunks
			return job.Status, isTerminal, nil
		}
		return "", false, fmt.Errorf("imports.Repository.AtomicChunkCompletion: %w", err)
	}

	isTerminal := processedChunks >= totalChunks
	return ImportJobStatus(status), isTerminal, nil
}

// ─── UpdateJobStatus ───────────────────────────────────────────────────────

func (r *PgRepository) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status ImportJobStatus) error {
	query := `
		UPDATE import_jobs
		SET status = $1,
		    started_at = CASE WHEN $1 = 'processing'::import_job_status AND started_at IS NULL THEN NOW() ELSE started_at END,
		    completed_at = CASE WHEN $1 IN ('completed'::import_job_status, 'completed_with_errors'::import_job_status, 'failed'::import_job_status, 'cancelled'::import_job_status) THEN NOW() ELSE completed_at END
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, string(status), jobID)
	if err != nil {
		return fmt.Errorf("imports.Repository.UpdateJobStatus: %w", err)
	}
	return nil
}

// ─── GetJobStagingRowCount ────────────────────────────────────────────────

func (r *PgRepository) GetJobStagingRowCount(ctx context.Context, jobID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM import_job_staging WHERE job_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, jobID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("imports.Repository.GetJobStagingRowCount: %w", err)
	}
	return count, nil
}

// ─── GetJobByIDempotencyKey ───────────────────────────────────────────────

func (r *PgRepository) GetJobByIDempotencyKey(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (*Job, error) {
	query := `
		SELECT id, tenant_id, school_id, job_type, role, created_by, status,
		       total_records, processed_records, success_count, failed_count,
		       idempotency_key, payload_hash, total_chunks, processed_chunks, metadata,
		       created_at, started_at, completed_at, last_progress_at
		FROM import_jobs
		WHERE tenant_id = $1 AND idempotency_key = $2
	`
	var j Job
	var role, idempotencyKeyOut, payloadHash *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query, tenantID, idempotencyKey).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKeyOut, &payloadHash, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.LastProgressAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("imports.Repository.GetJobByIDempotencyKey: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("imports.Repository.GetJobByIDempotencyKey: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKeyOut
	j.PayloadHash = payloadHash
	j.CreatedBy = createdBy
	return &j, nil
}

// ─── GetActiveJobBySchoolID ─────────────────────────────────────────────────

func (r *PgRepository) GetActiveJobBySchoolID(ctx context.Context, schoolID uuid.UUID) (*Job, error) {
	query := `
		SELECT id, tenant_id, school_id, job_type, role, created_by, status,
		       total_records, processed_records, success_count, failed_count,
		       idempotency_key, payload_hash, total_chunks, processed_chunks, metadata,
		       created_at, started_at, completed_at, last_progress_at
		FROM import_jobs
		WHERE school_id = $1 AND status IN ('processing'::import_job_status, 'cancelling'::import_job_status)
		ORDER BY created_at DESC
		LIMIT 1
	`
	var j Job
	var role, idempotencyKey, payloadHash *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query, schoolID).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKey, &payloadHash, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.LastProgressAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("imports.Repository.GetActiveJobBySchoolID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("imports.Repository.GetActiveJobBySchoolID: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKey
	j.PayloadHash = payloadHash
	j.CreatedBy = createdBy
	return &j, nil
}

// ─── CleanupStagingData ─────────────────────────────────────────────────────

func (r *PgRepository) CleanupStagingData(ctx context.Context, cutoff time.Time, batchSize int) (int, error) {
	query := `
		DELETE FROM import_job_staging
		WHERE job_id IN (
			SELECT id FROM import_jobs
			WHERE completed_at IS NOT NULL
			  AND completed_at < $1
			  AND status IN ('completed'::import_job_status, 'completed_with_errors'::import_job_status,
			                'failed'::import_job_status, 'cancelled'::import_job_status)
		)
		LIMIT $2
	`
	totalDeleted := 0
	for {
		result, err := r.pool.Exec(ctx, query, cutoff, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("imports.Repository.CleanupStagingData: %w", err)
		}
		deleted := int(result.RowsAffected())
		if deleted == 0 {
			break
		}
		totalDeleted += deleted
	}
	return totalDeleted, nil
}

// ─── CleanupFailureData ─────────────────────────────────────────────────────

func (r *PgRepository) CleanupFailureData(ctx context.Context, cutoff time.Time, batchSize int) (int, error) {
	query := `
		DELETE FROM import_job_failures
		WHERE import_job_id IN (
			SELECT id FROM import_jobs
			WHERE completed_at IS NOT NULL
			  AND completed_at < $1
			  AND status IN ('completed'::import_job_status, 'completed_with_errors'::import_job_status,
			                'failed'::import_job_status, 'cancelled'::import_job_status)
		)
		LIMIT $2
	`
	totalDeleted := 0
	for {
		result, err := r.pool.Exec(ctx, query, cutoff, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("imports.Repository.CleanupFailureData: %w", err)
		}
		deleted := int(result.RowsAffected())
		if deleted == 0 {
			break
		}
		totalDeleted += deleted
	}
	return totalDeleted, nil
}

// ─── TouchLastProgressAt ────────────────────────────────────────────────────

func (r *PgRepository) TouchLastProgressAt(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE import_jobs
		SET last_progress_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("imports.Repository.TouchLastProgressAt: %w", err)
	}
	return nil
}

// ─── GetFailures ───────────────────────────────────────────────────────────

func (r *PgRepository) GetFailures(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]RowFailure, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM import_job_failures WHERE import_job_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, jobID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("imports.Repository.GetFailures: count: %w", err)
	}

	query := `
		SELECT row_number, raw_payload, error_message, error_type
		FROM import_job_failures
		WHERE import_job_id = $1
		ORDER BY row_number ASC, id ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, jobID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("imports.Repository.GetFailures: query: %w", err)
	}
	defer rows.Close()

	var failures []RowFailure
	for rows.Next() {
		var f RowFailure
		var errType string
		if err := rows.Scan(&f.RowNumber, &f.RawPayload, &f.ErrorMessage, &errType); err != nil {
			return nil, 0, fmt.Errorf("imports.Repository.GetFailures: scan: %w", err)
		}
		f.ErrorType = ImportFailureType(errType)
		failures = append(failures, f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("imports.Repository.GetFailures: rows: %w", err)
	}

	if failures == nil {
		failures = []RowFailure{}
	}
	return failures, total, nil
}

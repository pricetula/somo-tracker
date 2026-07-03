package imports

import (
	"context"
	"fmt"
	"strings"

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
		                         total_records, idempotency_key, total_chunks, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		job.TenantID,
		job.SchoolID,
		job.JobType,
		job.Role,
		job.CreatedBy,
		job.Status,
		job.TotalRecords,
		job.IDempotencyKey,
		job.TotalChunks,
		job.Metadata,
	).Scan(&id)
	if err != nil {
		// Check for unique violation on idempotency_key
		if isUniqueViolation(err, "uq_import_jobs_tenant_idempotency") {
			return uuid.Nil, fmt.Errorf("imports.Repository.CreateJob: %w", ErrDuplicateJob)
		}
		return uuid.Nil, fmt.Errorf("imports.Repository.CreateJob: %w", err)
	}
	return id, nil
}

// ─── GetJobByID ────────────────────────────────────────────────────────────

func (r *PgRepository) GetJobByID(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		SELECT id, tenant_id, school_id, job_type, role, created_by, status,
		       total_records, processed_records, success_count, failed_count,
		       idempotency_key, total_chunks, processed_chunks, metadata,
		       created_at, started_at, completed_at
		FROM import_jobs
		WHERE id = $1
	`
	var j Job
	var role, idempotencyKey *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKey, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("imports.Repository.GetJobByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("imports.Repository.GetJobByID: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKey
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

// ─── GetStagingRows ────────────────────────────────────────────────────────

func (r *PgRepository) GetStagingRows(ctx context.Context, jobID uuid.UUID, rowStart, rowEnd int) ([]StagingRow, error) {
	query := `
		SELECT id, job_id, tenant_id, school_id, row_number, raw_data, status
		FROM import_job_staging
		WHERE job_id = $1 AND row_number >= $2 AND row_number < $3
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

// ─── AtomicChunkCompletion ─────────────────────────────────────────────────

func (r *PgRepository) AtomicChunkCompletion(ctx context.Context, jobID uuid.UUID, chunkProcessed, chunkSuccess, chunkFailed int) (ImportJobStatus, bool, error) {
	query := `
		UPDATE import_jobs
		SET
		    processed_records = processed_records + $2,
		    processed_chunks  = processed_chunks + 1,
		    success_count     = success_count + $3,
		    failed_count      = failed_count + $4,
		    status = CASE
		        WHEN processed_chunks + 1 = total_chunks AND failed_count + $4 = 0 THEN 'completed'::import_job_status
		        WHEN processed_chunks + 1 = total_chunks THEN 'completed_with_errors'::import_job_status
		        ELSE 'processing'::import_job_status
		    END,
		    completed_at = CASE WHEN processed_chunks + 1 = total_chunks THEN NOW() ELSE completed_at END
		WHERE id = $1
		RETURNING status, processed_chunks, total_chunks
	`
	var status string
	var processedChunks, totalChunks int
	err := r.pool.QueryRow(ctx, query, jobID, chunkProcessed, chunkSuccess, chunkFailed).Scan(&status, &processedChunks, &totalChunks)
	if err != nil {
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
		    completed_at = CASE WHEN $1 IN ('completed'::import_job_status, 'completed_with_errors'::import_job_status, 'failed'::import_job_status) THEN NOW() ELSE completed_at END
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
		       idempotency_key, total_chunks, processed_chunks, metadata,
		       created_at, started_at, completed_at
		FROM import_jobs
		WHERE tenant_id = $1 AND idempotency_key = $2
	`
	var j Job
	var role, idempotencyKeyOut *string
	var createdBy *uuid.UUID
	err := r.pool.QueryRow(ctx, query, tenantID, idempotencyKey).Scan(
		&j.ID, &j.TenantID, &j.SchoolID, &j.JobType, &role, &createdBy, &j.Status,
		&j.TotalRecords, &j.ProcessedRecords, &j.SuccessCount, &j.FailedCount,
		&idempotencyKeyOut, &j.TotalChunks, &j.ProcessedChunks, &j.Metadata,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("imports.Repository.GetJobByIDempotencyKey: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("imports.Repository.GetJobByIDempotencyKey: %w", err)
	}
	j.Role = role
	j.IDempotencyKey = idempotencyKeyOut
	j.CreatedBy = createdBy
	return &j, nil
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

// ============================================================================
// Helpers
// ============================================================================

func isUniqueViolation(err error, constraint string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") &&
		(constraint == "" || strings.Contains(msg, constraint))
}

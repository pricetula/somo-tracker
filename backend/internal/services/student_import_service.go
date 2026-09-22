package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type StudentImportItem struct {
	ID              uuid.UUID      `json:"id"`
	AdmissionNumber string         `json:"admission_number"`
	FullName        string         `json:"full_name"`
	DateOfBirth     string         `json:"date_of_birth"`
	Gender          string         `json:"gender"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type StudentImportService interface {
	CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []StudentImportItem) (uuid.UUID, error)
	GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error)
	GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error
	IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error
	GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error)
	UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error)
	RecomputeGenderCounts(ctx context.Context, schoolID uuid.UUID) error
	InsertStudent(ctx context.Context, schoolID uuid.UUID, admissionNumber, fullName string, dob time.Time, gender string, metadata json.RawMessage) error
	TryFinalizeJob(ctx context.Context, jobID uuid.UUID) (bool, error)
}

type studentImportService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewStudentImportService(pool *pgxpool.Pool, logger *zap.Logger) StudentImportService {
	return &studentImportService{pool: pool, logger: logger.With(zap.String("service", "student_import"))}
}

func (s *studentImportService) CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []StudentImportItem) (uuid.UUID, error) {
	if s.pool == nil {
		return uuid.Nil, fmt.Errorf("student_import_service: pool nil")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin_tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var jobID uuid.UUID
	qJob := `INSERT INTO bulk_jobs (job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id, idempotency_key) DO UPDATE SET updated_at=NOW() WHERE bulk_jobs.tenant_id=$4 AND bulk_jobs.idempotency_key=$2 RETURNING id`
	err = tx.QueryRow(ctx, qJob, "STUDENT_IMPORT", idempotencyKey, schoolID, tenantID, createdBy, "QUEUED", len(items), `{"source":"bulk_student_import"}`).Scan(&jobID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create_bulk_job_tx: %w", err)
	}

	batch := &pgx.Batch{}
	for idx, it := range items {
		payload, err := json.Marshal(it)
		if err != nil {
			return uuid.Nil, fmt.Errorf("marshal payload at %d: %w", idx, err)
		}
		batch.Queue(`INSERT INTO bulk_job_items (id, job_id, row_index, payload, status) VALUES ($1,$2,$3,$4,$5)`, it.ID, jobID, idx, payload, "PENDING")
	}
	br := tx.SendBatch(ctx, batch)
	_, err = br.Exec()
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert_items_tx: %w", err)
	}
	if err = br.Close(); err != nil {
		return uuid.Nil, fmt.Errorf("batch_close: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit_tx: %w", err)
	}
	return jobID, nil
}

func (s *studentImportService) GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE id=$1`, jobID)
	var b BulkJob
	err := row.Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_job: %w", err)
	}
	return &b, nil
}

func (s *studentImportService) GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, COALESCE(last_error,''), created_at, updated_at FROM bulk_job_items WHERE job_id=$1 ORDER BY row_index`, jobID)
	if err != nil {
		return nil, fmt.Errorf("get_items_by_job_id: %w", err)
	}
	defer rows.Close()
	var out []BulkJobItem
	for rows.Next() {
		var i BulkJobItem
		if err := rows.Scan(&i.ID, &i.JobID, &i.RowIndex, &i.Payload, &i.Result, &i.Status, &i.AttemptCount, &i.LastError, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *studentImportService) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_job_items SET status=$1, result=$2, last_error=$3, attempt_count=$4, updated_at=NOW() WHERE id=$5`, status, result, lastError, attempts, itemID)
	return err
}

func (s *studentImportService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_jobs SET status=$1, updated_at=NOW() WHERE id=$2`, status, jobID)
	return err
}

func (s *studentImportService) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_jobs SET succeeded_count = succeeded_count + $1, failed_count = failed_count + $2, deferred_count = deferred_count + $3, updated_at = NOW() WHERE id=$4`, succeeded, failed, deferred, jobID)
	return err
}

func (s *studentImportService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error) {
	var b BulkJob
	err := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, key).Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *studentImportService) UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error) {
	var role string
	err := s.pool.QueryRow(ctx, `SELECT role FROM school_memberships WHERE school_id=$1 AND user_id=$2 AND is_active=true`, schoolID, userID).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("user_has_admin_role: %w", err)
	}
	return role == "ADMIN", nil
}

func (s *studentImportService) GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, COALESCE(last_error,''), created_at, updated_at FROM bulk_job_items WHERE id=$1`, itemID)
	var i BulkJobItem
	err := row.Scan(&i.ID, &i.JobID, &i.RowIndex, &i.Payload, &i.Result, &i.Status, &i.AttemptCount, &i.LastError, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_item_by_id: %w", err)
	}
	return &i, nil
}

func (s *studentImportService) InsertStudent(ctx context.Context, schoolID uuid.UUID, admissionNumber, fullName string, dob time.Time, gender string, metadata json.RawMessage) error {
	if metadata == nil {
		metadata = []byte(`{}`)
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO students (school_id, admission_number, full_name, date_of_birth, gender, metadata) VALUES ($1,$2,$3,$4,$5,$6)`, schoolID, admissionNumber, fullName, dob, gender, metadata)
	if err != nil {
		return fmt.Errorf("insert_student: %w", err)
	}
	return nil
}

func (s *studentImportService) TryFinalizeJob(ctx context.Context, jobID uuid.UUID) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var job BulkJob
	err = tx.QueryRow(ctx, `SELECT id, school_id, status, total_records, succeeded_count, failed_count, deferred_count FROM bulk_jobs WHERE id=$1 FOR UPDATE`, jobID).Scan(&job.ID, &job.SchoolID, &job.Status, &job.TotalRecords, &job.SucceededCount, &job.FailedCount, &job.DeferredCount)
	if err != nil {
		return false, err
	}
	if job.Status == "COMPLETED" || job.Status == "COMPLETED_WITH_ERRORS" {
		return false, nil
	}
	if int(job.SucceededCount)+int(job.FailedCount)+int(job.DeferredCount) < int(job.TotalRecords) {
		return false, nil
	}
	var newStatus string
	if job.FailedCount > 0 {
		newStatus = "COMPLETED_WITH_ERRORS"
	} else {
		newStatus = "COMPLETED"
	}
	_, err = tx.Exec(ctx, `UPDATE bulk_jobs SET status=$1, updated_at=NOW() WHERE id=$2`, newStatus, jobID)
	if err != nil {
		return false, err
	}
	if newStatus == "COMPLETED" || newStatus == "COMPLETED_WITH_ERRORS" {
		_ = s.RecomputeGenderCounts(ctx, job.SchoolID)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// RecomputeGenderCounts recomputes student_gender_counts for a school using Option B.
// Normalises gender to M/F/OTHER.
func (s *studentImportService) RecomputeGenderCounts(ctx context.Context, schoolID uuid.UUID) error {
	// Upsert row with recomputed counts
	q := `
		INSERT INTO student_gender_counts (school_id, male_count, female_count, other_count, total_count, updated_at)
		SELECT $1,
		       COUNT(*) FILTER (WHERE upper(gender) IN ('M','MALE')),
		       COUNT(*) FILTER (WHERE upper(gender) IN ('F','FEMALE')),
		       COUNT(*) FILTER (WHERE upper(gender) NOT IN ('M','MALE','F','FEMALE') AND gender IS NOT NULL),
		       COUNT(*),
		       NOW()
		FROM students
		WHERE school_id = $1
		ON CONFLICT (school_id) DO UPDATE SET
		    male_count = EXCLUDED.male_count,
		    female_count = EXCLUDED.female_count,
		    other_count = EXCLUDED.other_count,
		    total_count = EXCLUDED.total_count,
		    updated_at = NOW()
	`
	_, err := s.pool.Exec(ctx, q, schoolID)
	if err != nil {
		return fmt.Errorf("recompute_gender_counts: %w", err)
	}
	return nil
}

// NormalizeGender maps various inputs to M/F/OTHER
func NormalizeGender(input string) string {
	if input == "" {
		return "OTHER"
	}
	u := strings.ToUpper(strings.TrimSpace(input))
	switch u {
	case "M", "MALE", "BOY", "BOYS":
		return "M"
	case "F", "FEMALE", "GIRL", "GIRLS":
		return "F"
	default:
		return "OTHER"
	}
}

// ParseDate tolerant parser for common formats
func ParseDate(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	layouts := []string{
		"2006-01-02",
		"02/01/2006", // MM/DD/YYYY
		"01-02-2006", // MM-DD-YYYY (must come before DD-MM-YYYY)
		"02-01-2006", // DD-MM-YYYY
		"01/02/2006", // DD/MM/YYYY
		"2006/01/02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, input); err == nil {
			return t, nil
		}
	}
	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", input)
}

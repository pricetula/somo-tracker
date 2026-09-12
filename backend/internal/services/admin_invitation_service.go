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

type InvitationItem struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type BulkJob struct {
	ID             uuid.UUID       `json:"id"`
	JobType        string          `json:"job_type"`
	IdempotencyKey string          `json:"idempotency_key"`
	SchoolID       uuid.UUID       `json:"school_id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	CreatedBy      uuid.UUID       `json:"created_by"`
	Status         string          `json:"status"`
	TotalRecords   int             `json:"total_records"`
	SucceededCount int             `json:"succeeded_count"`
	FailedCount    int             `json:"failed_count"`
	DeferredCount  int             `json:"deferred_count"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type BulkJobItem struct {
	ID           uuid.UUID       `json:"id"`
	JobID        uuid.UUID       `json:"job_id"`
	RowIndex     int             `json:"row_index"`
	Payload      json.RawMessage `json:"payload"`
	Result       json.RawMessage `json:"result"`
	Status       string          `json:"status"`
	AttemptCount int             `json:"attempt_count"`
	LastError    string          `json:"last_error"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type AdminInvitationService interface {
	CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error)
	InsertItems(ctx context.Context, jobID uuid.UUID, items []InvitationItem) error
	GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error)
	GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error)
	GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error
	UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result json.RawMessage, status string, lastError string) error
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error
	IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error
	GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error)
}

type adminInvitationService struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewAdminInvitationService(pool *pgxpool.Pool, logger *zap.Logger) AdminInvitationService {
	return &adminInvitationService{pool: pool, logger: logger.With(zap.String("service", "admin_invitation"))}
}

func (s *adminInvitationService) CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error) {
	if s.pool == nil {
		return uuid.Nil, fmt.Errorf("admin_invitation_service: pool nil")
	}
	var id uuid.UUID
	q := `INSERT INTO bulk_jobs (job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`
	err := s.pool.QueryRow(ctx, q, "ADMIN_INVITATION", idempotencyKey, schoolID, tenantID, createdBy, "QUEUED", total, `{"source":"bulk_invitation"}`).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "violates unique constraint") {
			return uuid.Nil, fmt.Errorf("idempotency_key_exists: %w", err)
		}
		return uuid.Nil, fmt.Errorf("create_bulk_job: %w", err)
	}
	return id, nil
}

func (s *adminInvitationService) InsertItems(ctx context.Context, jobID uuid.UUID, items []InvitationItem) error {
	if s.pool == nil {
		return fmt.Errorf("pool nil")
	}
	batch := &pgx.Batch{}
	for idx, it := range items {
		payload, _ := json.Marshal(it)
		batch.Queue(`INSERT INTO bulk_job_items (job_id, row_index, payload, status) VALUES ($1,$2,$3,$4)`, jobID, idx, payload, "PENDING")
	}
	br := s.pool.SendBatch(ctx, batch)
	_, err := br.Exec()
	if err != nil {
		return fmt.Errorf("batch_insert_items: %w", err)
	}
	return br.Close()
}

func (s *adminInvitationService) GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE id=$1`, jobID)
	var b BulkJob
	err := row.Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_job: %w", err)
	}
	return &b, nil
}

func (s *adminInvitationService) GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, last_error, created_at, updated_at FROM bulk_job_items WHERE job_id=$1 AND status IN ('FAILED','DEFERRED') ORDER BY row_index`, jobID)
	if err != nil {
		return nil, fmt.Errorf("get_retry_items: %w", err)
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

func (s *adminInvitationService) UpdateItem(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_job_items SET status=$1, result=$2, last_error=$3, attempt_count=$4, updated_at=NOW() WHERE id=$5`, status, result, lastError, attempts, itemID)
	return err
}

func (s *adminInvitationService) UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result json.RawMessage, status string, lastError string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_job_items SET result=$1, status=$2, last_error=$3, updated_at=NOW() WHERE id=$4`, result, status, lastError, itemID)
	return err
}

func (s *adminInvitationService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_jobs SET status=$1, updated_at=NOW() WHERE id=$2`, status, jobID)
	return err
}

func (s *adminInvitationService) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_jobs SET succeeded_count = succeeded_count + $1, failed_count = failed_count + $2, deferred_count = deferred_count + $3, updated_at = NOW() WHERE id=$4`, succeeded, failed, deferred, jobID)
	return err
}

func (s *adminInvitationService) GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, last_error, created_at, updated_at FROM bulk_job_items WHERE id=$1`, itemID)
	var i BulkJobItem
	err := row.Scan(&i.ID, &i.JobID, &i.RowIndex, &i.Payload, &i.Result, &i.Status, &i.AttemptCount, &i.LastError, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_item_by_id: %w", err)
	}
	return &i, nil
}

func (s *adminInvitationService) GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, last_error, created_at, updated_at FROM bulk_job_items WHERE job_id=$1 ORDER BY row_index`, jobID)
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

func (s *adminInvitationService) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_job_items SET status=$1, result=$2, last_error=$3, attempt_count=$4, updated_at=NOW() WHERE id=$5`, status, result, lastError, attempts, itemID)
	return err
}

func (s *adminInvitationService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error) {
	var b BulkJob
	err := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, key).Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

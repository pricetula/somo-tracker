package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type InvitationItem struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
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

type OutboxEntry struct {
	ID         uuid.UUID
	JobID      uuid.UUID
	BatchIndex int
	ItemIDs    []uuid.UUID
	Status     string
}

type AdminInvitationService interface {
	CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error)
	CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []InvitationItem) (uuid.UUID, error)
	InsertItems(ctx context.Context, jobID uuid.UUID, items []InvitationItem) error
	GetPendingOutbox(ctx context.Context, limit int) ([]OutboxEntry, error)
	MarkOutboxEnqueued(ctx context.Context, id uuid.UUID) error
	ArchiveFailedItem(ctx context.Context, itemID uuid.UUID) error
	GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error)
	GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error)
	GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error
	UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result json.RawMessage, status string, lastError string) error
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error
	IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error
	GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error)
	GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error)
	UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error)
	TryAcquireItem(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, bool, error)
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
	q := `INSERT INTO bulk_jobs (job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id, idempotency_key) DO UPDATE SET updated_at=NOW() WHERE bulk_jobs.tenant_id=$3 AND bulk_jobs.idempotency_key=$2 RETURNING id`
	err := s.pool.QueryRow(ctx, q, "ADMIN_INVITATION", idempotencyKey, schoolID, tenantID, createdBy, "QUEUED", total, `{"source":"bulk_invitation"}`).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create_bulk_job: %w", err)
	}
	return id, nil
}

func (s *adminInvitationService) CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []InvitationItem) (uuid.UUID, error) {
	if s.pool == nil {
		return uuid.Nil, fmt.Errorf("admin_invitation_service: pool nil")
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
	err = tx.QueryRow(ctx, qJob, "ADMIN_INVITATION", idempotencyKey, schoolID, tenantID, createdBy, "QUEUED", len(items), `{"source":"bulk_invitation"}`).Scan(&jobID)
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

	// Write outbox entries for reliable enqueueing
	const batchSize = 40
	for b := 0; b < (len(items)+batchSize-1)/batchSize; b++ {
		start := b * batchSize
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		ids := make([]uuid.UUID, 0, end-start)
		for j := start; j < end; j++ {
			ids = append(ids, items[j].ID)
		}
		_, err = tx.Exec(ctx, `INSERT INTO bulk_job_outbox (job_id, batch_index, item_ids) VALUES ($1,$2,$3) ON CONFLICT (job_id, batch_index) DO NOTHING`, jobID, b, ids)
		if err != nil {
			return uuid.Nil, fmt.Errorf("insert_outbox_tx batch %d: %w", b, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit_tx: %w", err)
	}
	return jobID, nil
}

func (s *adminInvitationService) InsertItems(ctx context.Context, jobID uuid.UUID, items []InvitationItem) error {
	if s.pool == nil {
		return fmt.Errorf("pool nil")
	}
	batch := &pgx.Batch{}
	for idx, it := range items {
		payload, err := json.Marshal(it)
		if err != nil {
			return fmt.Errorf("batch_insert_items: marshal payload at %d: %w", idx, err)
		}
		batch.Queue(`INSERT INTO bulk_job_items (id, job_id, row_index, payload, status) VALUES ($1,$2,$3,$4,$5)`, it.ID, jobID, idx, payload, "PENDING")
	}
	br := s.pool.SendBatch(ctx, batch)
	_, err := br.Exec()
	if err != nil {
		return fmt.Errorf("batch_insert_items: %w", err)
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("batch_insert_items: close batch: %w", err)
	}
	return nil
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
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, COALESCE(last_error,''), created_at, updated_at FROM bulk_job_items WHERE job_id=$1 AND status IN ('FAILED','DEFERRED') ORDER BY row_index`, jobID)
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
	row := s.pool.QueryRow(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, COALESCE(last_error,''), created_at, updated_at FROM bulk_job_items WHERE id=$1`, itemID)
	var i BulkJobItem
	err := row.Scan(&i.ID, &i.JobID, &i.RowIndex, &i.Payload, &i.Result, &i.Status, &i.AttemptCount, &i.LastError, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_item_by_id: %w", err)
	}
	return &i, nil
}

func (s *adminInvitationService) GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error) {
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

func (s *adminInvitationService) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_job_items SET status=$1, result=$2, last_error=$3, attempt_count=$4, updated_at=NOW() WHERE id=$5`, status, result, lastError, attempts, itemID)
	return err
}

func (s *adminInvitationService) GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if s.pool == nil {
		return "", fmt.Errorf("pool nil")
	}
	var org string
	err := s.pool.QueryRow(ctx, `SELECT stytch_org_id FROM tenants WHERE id=$1`, tenantID).Scan(&org)
	if err != nil {
		return "", fmt.Errorf("get_stytch_org_id: %w", err)
	}
	return org, nil
}

func (s *adminInvitationService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error) {
	var b BulkJob
	err := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, key).Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *adminInvitationService) UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error) {
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

func (s *adminInvitationService) TryAcquireItem(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, bool, error) {
	var item BulkJobItem
	// Try to move item from PENDING to PROCESSING atomically
	err := s.pool.QueryRow(ctx, `UPDATE bulk_job_items SET status='PROCESSING', attempt_count=attempt_count+1, updated_at=NOW() WHERE id=$1 AND status='PENDING' RETURNING id, job_id, row_index, payload, result, status, attempt_count, COALESCE(last_error,''), created_at, updated_at`, itemID).Scan(&item.ID, &item.JobID, &item.RowIndex, &item.Payload, &item.Result, &item.Status, &item.AttemptCount, &item.LastError, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("try_acquire_item: %w", err)
	}
	return &item, true, nil
}

func (s *adminInvitationService) GetPendingOutbox(ctx context.Context, limit int) ([]OutboxEntry, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, job_id, batch_index, item_ids, status FROM bulk_job_outbox WHERE status='PENDING' ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("get_pending_outbox: %w", err)
	}
	defer rows.Close()
	var out []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var itemIDs []uuid.UUID
		if err := rows.Scan(&e.ID, &e.JobID, &e.BatchIndex, &itemIDs, &e.Status); err != nil {
			return nil, err
		}
		e.ItemIDs = itemIDs
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *adminInvitationService) MarkOutboxEnqueued(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE bulk_job_outbox SET status='ENQUEUED', updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (s *adminInvitationService) ArchiveFailedItem(ctx context.Context, itemID uuid.UUID) error {
	var jobID uuid.UUID
	var payload json.RawMessage
	var lastError string
	var attemptCount int
	err := s.pool.QueryRow(ctx, `SELECT job_id, payload, COALESCE(last_error,''), attempt_count FROM bulk_job_items WHERE id=$1`, itemID).Scan(&jobID, &payload, &lastError, &attemptCount)
	if err != nil {
		return fmt.Errorf("archive_failed_item fetch: %w", err)
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO bulk_job_dead_letter (item_id, job_id, payload, last_error, attempt_count) VALUES ($1,$2,$3,$4,$5)`, itemID, jobID, payload, lastError, attemptCount)
	if err != nil {
		return fmt.Errorf("archive_failed_item insert: %w", err)
	}
	return nil
}

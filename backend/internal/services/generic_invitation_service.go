package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database/sqlc"
)

type GenericInvitationService struct {
	pool    *pgxpool.Pool
	logger  *zap.Logger
	queries *sqlc.Queries
	jobType string
	role    string
}

func NewGenericInvitationService(pool *pgxpool.Pool, logger *zap.Logger, jobType, role string) *GenericInvitationService {
	return &GenericInvitationService{
		pool:    pool,
		logger:  logger.With(zap.String("service", "invitation")),
		queries: sqlc.New(pool),
		jobType: jobType,
		role:    role,
	}
}

func (s *GenericInvitationService) DeleteBulkJobsForUser(ctx context.Context, userID uuid.UUID) error {
	if s.pool == nil {
		return fmt.Errorf("invitation_service: pool nil")
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE created_by = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete_bulk_jobs_for_user: %w", err)
	}
	return nil
}

func (s *GenericInvitationService) CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error) {
	if s.pool == nil {
		return uuid.Nil, fmt.Errorf("invitation_service: pool nil")
	}
	var id uuid.UUID
	q := `INSERT INTO bulk_jobs (job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id, idempotency_key) DO UPDATE SET updated_at=NOW() WHERE bulk_jobs.tenant_id=$4 AND bulk_jobs.idempotency_key=$2 RETURNING id`
	err := s.pool.QueryRow(ctx, q, s.jobType, idempotencyKey, schoolID, tenantID, createdBy, "QUEUED", total, `{"source":"bulk_invitation"}`).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create_bulk_job: %w", err)
	}
	return id, nil
}

func (s *GenericInvitationService) CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []InvitationItem) (uuid.UUID, error) {
	return s.CreateBulkJobWithItemsWithAuth(ctx, schoolID, tenantID, createdBy, createdBy, idempotencyKey, items)
}

// CreateBulkJobWithItemsWithAuth creates a bulk job with authorization check.
// actorUserID is the user performing the action; createdBy is recorded as job creator.
func (s *GenericInvitationService) CreateBulkJobWithItemsWithAuth(ctx context.Context, schoolID, tenantID, createdBy, actorUserID uuid.UUID, idempotencyKey string, items []InvitationItem) (uuid.UUID, error) {
	if s.pool == nil {
		return uuid.Nil, fmt.Errorf("invitation_service: pool nil")
	}
	// Authorization: verify actor has admin role for this school
	hasAdmin, err := s.UserHasAdminRole(ctx, schoolID, actorUserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("auth check failed: %w", err)
	}
	if !hasAdmin {
		return uuid.Nil, NewErrForbidden("insufficient_permissions", "insufficient permissions")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin_tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
				s.logger.Error("CreateBulkJobWithItems rollback failed", zap.Error(rbErr))
			}
		}
	}()

	var jobID uuid.UUID
	qJob := `INSERT INTO bulk_jobs (job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id, idempotency_key) DO UPDATE SET updated_at=NOW() WHERE bulk_jobs.tenant_id=$4 AND bulk_jobs.idempotency_key=$2 RETURNING id`
	err = tx.QueryRow(ctx, qJob, s.jobType, idempotencyKey, schoolID, tenantID, createdBy, "QUEUED", len(items), `{"source":"bulk_invitation"}`).Scan(&jobID)
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

func (s *GenericInvitationService) InsertItems(ctx context.Context, jobID uuid.UUID, items []InvitationItem) error {
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

func (s *GenericInvitationService) GetJob(ctx context.Context, jobID uuid.UUID) (*BulkJob, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE id=$1`, jobID)
	var b BulkJob
	err := row.Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_job: %w", err)
	}
	return &b, nil
}

func (s *GenericInvitationService) GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error) {
	return s.GetFailedOrDeferredItemsWithAuth(ctx, jobID, uuid.Nil, uuid.Nil)
}

// GetFailedOrDeferredItemsWithAuth returns failed/deferred items with authorization check.
// schoolID and actorUserID are used to verify the job belongs to the school and actor is admin.
func (s *GenericInvitationService) GetFailedOrDeferredItemsWithAuth(ctx context.Context, jobID, schoolID, actorUserID uuid.UUID) ([]BulkJobItem, error) {
	if schoolID != uuid.Nil && actorUserID != uuid.Nil {
		hasAdmin, err := s.UserHasAdminRole(ctx, schoolID, actorUserID)
		if err != nil {
			return nil, fmt.Errorf("auth check failed: %w", err)
		}
		if !hasAdmin {
			return nil, NewErrForbidden("insufficient_permissions", "insufficient permissions")
		}
	}
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

func (s *GenericInvitationService) GetItemByID(ctx context.Context, itemID uuid.UUID) (*BulkJobItem, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, job_id, row_index, payload, result, status, attempt_count, COALESCE(last_error,''), created_at, updated_at FROM bulk_job_items WHERE id=$1`, itemID)
	var i BulkJobItem
	err := row.Scan(&i.ID, &i.JobID, &i.RowIndex, &i.Payload, &i.Result, &i.Status, &i.AttemptCount, &i.LastError, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get_item_by_id: %w", err)
	}
	return &i, nil
}

func (s *GenericInvitationService) GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]BulkJobItem, error) {
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

func (s *GenericInvitationService) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result []byte, lastError string, attempts int) error {
	return s.queries.UpdateBulkJobItem(ctx, sqlc.UpdateBulkJobItemParams{
		ID:           pgtype.UUID{Bytes: itemID, Valid: true},
		Status:       status,
		Result:       result,
		LastError:    pgtype.Text{String: lastError, Valid: lastError != ""},
		AttemptCount: int32(attempts),
	})
}

func (s *GenericInvitationService) UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result []byte, status string, lastError string) error {
	return s.queries.UpdateBulkJobItemResultStatus(ctx, sqlc.UpdateBulkJobItemResultStatusParams{
		ID:        pgtype.UUID{Bytes: itemID, Valid: true},
		Result:    result,
		Status:    status,
		LastError: pgtype.Text{String: lastError, Valid: lastError != ""},
	})
}

func (s *GenericInvitationService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	return s.queries.UpdateBulkJobStatus(ctx, sqlc.UpdateBulkJobStatusParams{
		ID:     pgtype.UUID{Bytes: jobID, Valid: true},
		Status: status,
	})
}

func (s *GenericInvitationService) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error {
	return s.queries.IncrementBulkJobCounts(ctx, sqlc.IncrementBulkJobCountsParams{
		ID:             pgtype.UUID{Bytes: jobID, Valid: true},
		SucceededCount: int32(succeeded),
		FailedCount:    int32(failed),
		DeferredCount:  int32(deferred),
	})
}

func (s *GenericInvitationService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*BulkJob, error) {
	var b BulkJob
	err := s.pool.QueryRow(ctx, `SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, key).Scan(&b.ID, &b.JobType, &b.IdempotencyKey, &b.SchoolID, &b.TenantID, &b.CreatedBy, &b.Status, &b.TotalRecords, &b.SucceededCount, &b.FailedCount, &b.DeferredCount, &b.Metadata, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *GenericInvitationService) GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var org string
	err := s.pool.QueryRow(ctx, `SELECT stytch_org_id FROM tenants WHERE id=$1`, tenantID).Scan(&org)
	if err != nil {
		return "", fmt.Errorf("get_stytch_org_id: %w", err)
	}
	return org, nil
}

func (s *GenericInvitationService) UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error) {
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

func (s *GenericInvitationService) ProvisionInvitee(ctx context.Context, tenantID, schoolID, invitedBy uuid.UUID, email, fullName, role, stytchMemberID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("provision_invitee begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			s.logger.Error("provision_invitee rollback failed", zap.Error(rbErr))
		}
	}()
	var userID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO users (email, full_name, external_auth_id, tenant_id) VALUES ($1,$2,$3,$4) ON CONFLICT (tenant_id, email) DO UPDATE SET external_auth_id = COALESCE(users.external_auth_id,$3), full_name = COALESCE(users.full_name,$2) RETURNING id`, email, fullName, stytchMemberID, tenantID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("provision_invitee upsert user: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO members (stytch_member_id, user_id, tenant_id, stytch_member_raw) VALUES ($1,$2,$3,'{}') ON CONFLICT (tenant_id, stytch_member_id) DO NOTHING`, stytchMemberID, userID, tenantID)
	if err != nil {
		return fmt.Errorf("provision_invitee upsert member: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active, invited_at, invited_by) VALUES ($1,$2,$3,true,NOW(),$4) ON CONFLICT (school_id, user_id) DO UPDATE SET role = EXCLUDED.role, is_active = true`, schoolID, userID, role, invitedBy)
	if err != nil {
		return fmt.Errorf("provision_invitee upsert membership: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("provision_invitee commit: %w", err)
	}
	return nil
}

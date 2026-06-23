package imports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles import job database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ─── Import Jobs ─────────────────────────────────────────────────────────

// CreateImportJob inserts a new import job and returns its ID.
func (r *PgRepository) CreateImportJob(ctx context.Context, job *ImportJob) error {
	const query = `
		INSERT INTO import_jobs (id, tenant_id, school_id, role, created_by, status, total_records,
		                         processed_records, success_count, failed_count, parent_import_job_id)
		VALUES ($1, $2, $3, $4::user_role, $5, $6, $7, $8, $9, $10, $11)
	`

	createdByArg := interface{}(nil)
	if job.CreatedBy != nil {
		createdByArg = *job.CreatedBy
	}

	parentArg := interface{}(nil)
	if job.ParentImportJobID != nil {
		parentArg = *job.ParentImportJobID
	}

	_, err := r.pool.Exec(ctx, query,
		job.ID, job.TenantID, job.SchoolID, job.Role,
		createdByArg, job.Status, job.TotalRecords,
		job.ProcessedRecords, job.SuccessCount, job.FailedCount,
		parentArg,
	)
	if err != nil {
		return fmt.Errorf("create import job: %w", err)
	}
	return nil
}

// GetImportJob retrieves an import job by ID.
func (r *PgRepository) GetImportJob(ctx context.Context, jobID string) (*ImportJob, error) {
	const query = `
		SELECT id, tenant_id, school_id, role::text, created_by, status,
		       total_records, processed_records, success_count, failed_count,
		       parent_import_job_id, created_at, started_at, completed_at
		FROM import_jobs
		WHERE id = $1
	`

	var job ImportJob
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&job.ID, &job.TenantID, &job.SchoolID, &job.Role, &job.CreatedBy,
		&job.Status, &job.TotalRecords, &job.ProcessedRecords, &job.SuccessCount,
		&job.FailedCount, &job.ParentImportJobID, &job.CreatedAt,
		&job.StartedAt, &job.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("import job not found")
		}
		return nil, fmt.Errorf("get import job: %w", err)
	}
	return &job, nil
}

// UpdateImportJobStatus updates the status and counters of an import job.
func (r *PgRepository) UpdateImportJobStatus(ctx context.Context, id, status string, processed, successCount, failedCount int) error {
	const query = `
		UPDATE import_jobs
		SET status = $2, processed_records = $3, success_count = $4, failed_count = $5
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, status, processed, successCount, failedCount)
	if err != nil {
		return fmt.Errorf("update import job status: %w", err)
	}
	return nil
}

// SetImportJobStarted marks a job as started.
func (r *PgRepository) SetImportJobStarted(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs SET status = 'processing', started_at = NOW() WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("set import job started: %w", err)
	}
	return nil
}

// SetImportJobCompleted marks a job as completed (or completed_with_errors).
func (r *PgRepository) SetImportJobCompleted(ctx context.Context, id string, hasErrors bool) error {
	status := "completed"
	if hasErrors {
		status = "completed_with_errors"
	}
	const query = `
		UPDATE import_jobs SET status = $2, completed_at = NOW() WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("set import job completed: %w", err)
	}
	return nil
}

// SetImportJobFailed marks a job as failed after all retries are exhausted.
func (r *PgRepository) SetImportJobFailed(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs SET status = 'failed', completed_at = NOW() WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("set import job failed: %w", err)
	}
	return nil
}

// ─── Invitations (bulk insert with temp_id reconciliation) ────────────────

// BulkInsertInvitations inserts a batch of invitations using a CTE that pairs
// each row's client-generated temp_id with the inserted invitation id.
// Returns a map[temp_id]invitation_id and a list of rows that were duplicates
// (not inserted due to ON CONFLICT DO NOTHING).
func (r *PgRepository) BulkInsertInvitations(
	ctx context.Context, records []ImportStaffRecord,
	tenantID, schoolID, role, jobID string,
	now time.Time, tokenPrefix string,
) (map[string]string, []FailedInsertion, error) {
	if len(records) == 0 {
		return map[string]string{}, nil, nil
	}

	// Build CTE VALUES clause: 12 params per row (temp_id through token)
	valueStrings := make([]string, 0, len(records))
	args := make([]interface{}, 0, len(records)*12)
	argIdx := 1

	for _, rec := range records {
		// Each row: (temp_id, tenant_id, school_id, LOWER(email), role, expires_at, first_name, last_name, phone, registration_number, import_job_id, token)
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::text, $%d::uuid, $%d::uuid, LOWER($%d), $%d::user_role, $%d::timestamptz, $%d, $%d, $%d, $%d, $%d::uuid, $%d)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4,
				argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10, argIdx+11),
		)
		args = append(args,
			rec.TempID, // temp_id — used for reconciliation
			tenantID,
			schoolID,
			rec.Email,
			role,
			now.Add(InvitationTTL), // expires_at
			rec.FirstName,
			rec.LastName,
			rec.Phone,
			rec.RegistrationNumber,
			jobID,
			tokenPrefix+rec.TempID, // token
		)
		argIdx += 12
	}

	query := `
		WITH input_rows (temp_id, tenant_id, school_id, email, role, expires_at, first_name, last_name, phone, registration_number, import_job_id, token) AS (
			VALUES ` + strings.Join(valueStrings, ",\n			       ") + `
		),
		inserted AS (
			INSERT INTO invitations
				(tenant_id, school_id, email, role, status, expires_at,
				 first_name, last_name, phone, registration_number, import_job_id, token)
			SELECT tenant_id, school_id, email, role, 'pending'::invitation_status, expires_at,
			       first_name, last_name, phone, registration_number, import_job_id, token
			FROM input_rows
			ON CONFLICT (tenant_id, school_id, email) WHERE status NOT IN ('expired', 'revoked')
			DO NOTHING
			RETURNING id, email
		)
		SELECT ir.temp_id, ins.id, ins.email
		FROM input_rows ir
		LEFT JOIN inserted ins ON LOWER(ir.email) = LOWER(ins.email)
		ORDER BY ir.temp_id
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("bulk insert invitations: %w", err)
	}
	defer rows.Close()

	inserted := make(map[string]string) // temp_id -> invitation_id
	var failures []FailedInsertion

	for rows.Next() {
		var tempID, email string
		var invIDPtr *string
		if err := rows.Scan(&tempID, &invIDPtr, &email); err != nil {
			return nil, nil, fmt.Errorf("scan inserted invitation: %w", err)
		}
		if invIDPtr != nil {
			inserted[tempID] = *invIDPtr
		} else {
			// temp_id was not inserted — duplicate
			failures = append(failures, FailedInsertion{
				TempID: tempID,
				Email:  email,
				Reason: "duplicate",
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration: %w", err)
	}

	return inserted, failures, nil
}

// FailedInsertion represents a record that could not be inserted.
type FailedInsertion struct {
	TempID string `json:"temp_id"`
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

// ─── Invitation updates (Stage 2 + correction resubmit) ──────────────────

// SetInvitationStytchMemberID updates the stytch_member_id on an invitation.
func (r *PgRepository) SetInvitationStytchMemberID(ctx context.Context, id, stytchMemberID string) error {
	const query = `UPDATE invitations SET stytch_member_id = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, stytchMemberID, id)
	if err != nil {
		return fmt.Errorf("imports.Repository.SetInvitationStytchMemberID: %w", err)
	}
	return nil
}

// SetInvitationFailed marks an invitation as permanently failed.
func (r *PgRepository) SetInvitationFailed(ctx context.Context, id, errorMessage string, attemptCount int) error {
	const query = `
		UPDATE invitations
		SET status = 'invite_failed', error_message = $2, attempt_count = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, errorMessage, attemptCount)
	if err != nil {
		return fmt.Errorf("imports.Repository.SetInvitationFailed: %w", err)
	}
	return nil
}

// GetInvitationStytchMemberID returns the stytch_member_id for a row (or empty string).
func (r *PgRepository) GetInvitationStytchMemberID(ctx context.Context, id string) (string, error) {
	const query = `SELECT COALESCE(stytch_member_id, '') FROM invitations WHERE id = $1`
	var memberID string
	err := r.pool.QueryRow(ctx, query, id).Scan(&memberID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get stytch member id: %w", err)
	}
	return memberID, nil
}

// GetPendingStage2Records returns invitations for this job that haven't yet been
// sent to Stytch. Used on task retry to resume Stage 2 from where it left off
// instead of re-processing already-invited records.
func (r *PgRepository) GetPendingStage2Records(ctx context.Context, jobID string) ([]Stage2Record, error) {
	const query = `
		SELECT id, email, first_name, last_name
		FROM invitations
		WHERE import_job_id = $1
		  AND (stytch_member_id IS NULL OR stytch_member_id = '')
		  AND status != 'invite_failed'
		ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("get pending stage 2 records: %w", err)
	}
	defer rows.Close()

	var records []Stage2Record
	for rows.Next() {
		var rec Stage2Record
		if err := rows.Scan(&rec.InvitationID, &rec.Email, &rec.FirstName, &rec.LastName); err != nil {
			return nil, fmt.Errorf("scan stage 2 record: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return records, nil
}

// BulkUpdateInvitations updates existing invitation rows by ID (correction resubmit).
// Used when re-running failed invitations: re-validates the partial unique index
// constraint (active email uniqueness) before retrying Stage 2.
// Returns the count of successfully updated rows.
func (r *PgRepository) BulkUpdateInvitations(
	ctx context.Context, records []ImportStaffRecord,
	role, jobID string, now time.Time,
) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	// Build CTE VALUES clause
	valueStrings := make([]string, 0, len(records))
	args := make([]interface{}, 0, len(records)*8)
	argIdx := 1

	for _, rec := range records {
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::uuid, LOWER($%d), $%d, $%d, $%d, $%d, $%d::user_role, $%d::uuid)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7),
		)
		args = append(args,
			rec.TempID, // id of the invitation row (passed as temp_id in correction flow)
			rec.Email,
			rec.FirstName,
			rec.LastName,
			rec.Phone,
			rec.RegistrationNumber,
			role,
			jobID,
		)
		argIdx += 8
	}

	query := `
		WITH corrections (id, email, first_name, last_name, phone, registration_number, role, import_job_id) AS (
			VALUES ` + strings.Join(valueStrings, ",\n			         ") + `
		)
		UPDATE invitations inv
		SET
			email               = LOWER(c.email),
			first_name          = c.first_name,
			last_name           = c.last_name,
			phone               = c.phone,
			registration_number = c.registration_number,
			role                = c.role::user_role,
			status              = 'pending',
			error_message       = NULL,
			attempt_count       = 0,
			import_job_id       = c.import_job_id
		FROM corrections c
		WHERE inv.id = c.id
		RETURNING inv.id
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("bulk update invitations: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	return count, rows.Err()
}
func (r *PgRepository) RecordImportFailure(ctx context.Context, jobID, rawPayloadJSON, errMsg string) error {
	const query = `
		INSERT INTO import_job_failures (import_job_id, raw_payload, error_message)
		VALUES ($1, $2::jsonb, $3)
	`
	_, err := r.pool.Exec(ctx, query, jobID, rawPayloadJSON, errMsg)
	if err != nil {
		return fmt.Errorf("imports.Repository.RecordImportFailure: %w", err)
	}
	return nil
}

// ─── Failed Invitations (post-import recovery) ───────────────────────────

// GetFailedInvitationsByJob returns invitations that failed during Stytch invite.
func (r *PgRepository) GetFailedInvitationsByJob(ctx context.Context, jobID string) ([]FailedInvitation, error) {
	const query = `
		SELECT id, email, first_name, last_name, phone, error_message
		FROM invitations
		WHERE import_job_id = $1 AND status = 'invite_failed'
		ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("get failed invitations: %w", err)
	}
	defer rows.Close()

	var results []FailedInvitation
	for rows.Next() {
		var fi FailedInvitation
		if err := rows.Scan(&fi.ID, &fi.Email, &fi.FirstName, &fi.LastName, &fi.Phone, &fi.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan failed invitation: %w", err)
		}
		results = append(results, fi)
	}
	return results, rows.Err()
}

// ─── School / Tenant helpers ─────────────────────────────────────────────

// GetActiveSchoolID returns the active school ID for a user in a tenant.
func (r *PgRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	const query = `
		SELECT school_id FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		ORDER BY
			CASE role
				WHEN 'SCHOOL_ADMIN'::user_role THEN 1
				WHEN 'TEACHER'::user_role THEN 2
				WHEN 'NURSE'::user_role THEN 3
				WHEN 'FINANCE'::user_role THEN 4
			END
		LIMIT 1
	`

	var schoolID string
	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(&schoolID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("no active membership found")
		}
		return "", fmt.Errorf("get active school: %w", err)
	}
	return schoolID, nil
}

// GetTenantStytchOrgID returns the Stytch org ID for a tenant.
func (r *PgRepository) GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	const query = `SELECT stytch_org_id FROM tenants WHERE id = $1`

	var orgID string
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&orgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("tenant not found")
		}
		return "", fmt.Errorf("get tenant stytch org: %w", err)
	}
	return orgID, nil
}

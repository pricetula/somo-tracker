package imports

import (
	"context"
	"encoding/json"
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

	// Build CTE VALUES clause: 11 params per row (temp_id through token)
	valueStrings := make([]string, 0, len(records))
	args := make([]interface{}, 0, len(records)*11)
	argIdx := 1

	for _, rec := range records {
		// Each row: (temp_id, tenant_id, school_id, LOWER(email), role, expires_at, full_name, phone, registration_number, import_job_id, token)
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::text, $%d::uuid, $%d::uuid, LOWER($%d), $%d::user_role, $%d::timestamptz, $%d, $%d, $%d, $%d::uuid, $%d)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4,
				argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10),
		)
		args = append(args,
			rec.TempID, // temp_id — used for reconciliation
			tenantID,
			schoolID,
			rec.Email,
			role,
			now.Add(InvitationTTL), // expires_at
			rec.FullName,
			rec.Phone,
			rec.RegistrationNumber,
			jobID,
			tokenPrefix+rec.TempID, // token
		)
		argIdx += 11
	}

	query := `
		WITH input_rows (temp_id, tenant_id, school_id, email, role, expires_at, full_name, phone, registration_number, import_job_id, token) AS (
			VALUES ` + strings.Join(valueStrings, ",\n			       ") + `
		),
		inserted AS (
			INSERT INTO invitations
				(tenant_id, school_id, email, role, status, expires_at,
				 full_name, phone, registration_number, import_job_id, token)
			SELECT tenant_id, school_id, email, role, 'pending'::invitation_status, expires_at,
			       full_name, phone, registration_number, import_job_id, token
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
		SELECT id, email, full_name
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
		if err := rows.Scan(&rec.InvitationID, &rec.Email, &rec.FullName); err != nil {
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
	args := make([]interface{}, 0, len(records)*7)
	argIdx := 1

	for _, rec := range records {
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::uuid, LOWER($%d), $%d, $%d, $%d, $%d::user_role, $%d::uuid)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6),
		)
		args = append(args,
			rec.TempID, // id of the invitation row (passed as temp_id in correction flow)
			rec.Email,
			rec.FullName,
			rec.Phone,
			rec.RegistrationNumber,
			role,
			jobID,
		)
		argIdx += 7
	}

	query := `
		WITH corrections (id, email, full_name, phone, registration_number, role, import_job_id) AS (
			VALUES ` + strings.Join(valueStrings, ",\n			         ") + `
		)
		UPDATE invitations inv
		SET
			email               = LOWER(c.email),
			full_name           = c.full_name,
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

// BulkRecordImportFailure inserts multiple failure records in a single query.
// This replaces the previous per-record loop, reducing N round-trips to 1.
func (r *PgRepository) BulkRecordImportFailure(
	ctx context.Context, jobID string,
	records []ImportStaffRecord, errMsg string,
) error {
	if len(records) == 0 {
		return nil
	}

	// Build CTE VALUES clause: 2 params per row (raw_payload JSONB, error_message)
	valueStrings := make([]string, 0, len(records))
	args := make([]interface{}, 0, len(records)*2+1)
	argIdx := 1

	// First param is the shared import_job_id
	args = append(args, jobID)
	argIdx++

	for _, rec := range records {
		raw, err := json.Marshal(rec)
		if err != nil {
			// Marshal should never fail for our struct, but if it does,
			// use a fallback so we don't lose the failure tracking.
			raw = []byte(`{"marshal_error": "` + err.Error() + `"}`)
		}
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::jsonb, $%d)", argIdx, argIdx+1),
		)
		args = append(args, string(raw), errMsg)
		argIdx += 2
	}

	query := `
		WITH input_rows (raw_payload, error_message) AS (
			VALUES ` + strings.Join(valueStrings, ",\n			       ") + `
		)
		INSERT INTO import_job_failures (import_job_id, raw_payload, error_message)
		SELECT $1, raw_payload, error_message
		FROM input_rows
	`

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.BulkRecordImportFailure: %w", err)
	}
	return nil
}

// ─── Failed Invitations (post-import recovery) ───────────────────────────

// GetFailedInvitationsByJob returns invitations that failed during Stytch invite.
func (r *PgRepository) GetFailedInvitationsByJob(ctx context.Context, jobID string) ([]FailedInvitation, error) {
	const query = `
		SELECT id, email, full_name, phone, error_message
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
		if err := rows.Scan(&fi.ID, &fi.Email, &fi.FullName, &fi.Phone, &fi.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan failed invitation: %w", err)
		}
		results = append(results, fi)
	}
	return results, rows.Err()
}

// ─── School / Tenant helpers ─────────────────────────────────────────────

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

// ============================================================================
// Student Import Repository Methods
// ============================================================================

// CheckConcurrentImport returns true if an import is already in progress
// (pending or processing) for this tenant+school.
func (r *PgRepository) CheckConcurrentImport(ctx context.Context, tenantID, schoolID string) (bool, error) {
	const query = `
		SELECT id FROM import_jobs
		WHERE tenant_id = $1
		  AND school_id = $2
		  AND status IN ('pending', 'processing')
		LIMIT 1
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("imports.Repository.CheckConcurrentImport: %w", err)
	}
	return true, nil
}

// BulkInsertStaging inserts student records into import_job_staging in a single query.
func (r *PgRepository) BulkInsertStaging(ctx context.Context, jobID, tenantID, schoolID string, records []StudentRecord, academicYear, term string) error {
	if len(records) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(records))
	args := make([]interface{}, 0, len(records)*6)
	argIdx := 1

	for i, rec := range records {
		// Build raw_data JSONB: student fields + academic_year + term
		raw := map[string]interface{}{
			"full_name":              rec.FullName,
			"gender":                 rec.Gender,
			"date_of_birth":          rec.DateOfBirth,
			"upi_number":             rec.UPINumber,
			"knec_assessment_number": rec.KNECAssessmentNumber,
			"cbc_student_parents_id": rec.CBCStudentParentsID,
			"class_id":               rec.ClassID,
			"academic_year":          academicYear,
			"term":                   term,
		}
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			return fmt.Errorf("imports.Repository.BulkInsertStaging: marshal raw_data: %w", err)
		}

		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::uuid, $%d::uuid, $%d::uuid, $%d, $%d::jsonb)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4),
		)
		args = append(args, jobID, tenantID, schoolID, i+1, string(rawJSON))
		argIdx += 5
	}

	query := `
		INSERT INTO import_job_staging (job_id, tenant_id, school_id, row_number, raw_data)
		VALUES ` + strings.Join(valueStrings, ",\n		       ") + `
	`

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.BulkInsertStaging: %w", err)
	}
	return nil
}

// GetStagingRows loads all staging rows for a job, ordered by row_number.
func (r *PgRepository) GetStagingRows(ctx context.Context, jobID string) ([]StagingRow, error) {
	const query = `
		SELECT row_number, raw_data
		FROM import_job_staging
		WHERE job_id = $1
		ORDER BY row_number ASC
	`

	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.GetStagingRows: %w", err)
	}
	defer rows.Close()

	var results []StagingRow
	for rows.Next() {
		var sr StagingRow
		var rawJSON []byte
		if err := rows.Scan(&sr.RowNumber, &rawJSON); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetStagingRows: scan: %w", err)
		}
		if err := json.Unmarshal(rawJSON, &sr.RawData); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetStagingRows: unmarshal: %w", err)
		}
		results = append(results, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.GetStagingRows: rows: %w", err)
	}
	return results, nil
}

// GetValidClasses returns a set of valid (active) class IDs for this tenant+school.
func (r *PgRepository) GetValidClasses(ctx context.Context, tenantID, schoolID string, classIDs []string) (map[string]bool, error) {
	if len(classIDs) == 0 {
		return map[string]bool{}, nil
	}

	query := `
		SELECT id::text FROM cbc_classes
		WHERE tenant_id = $1 AND school_id = $2 AND id = ANY($3) AND is_active = true
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, classIDs)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.GetValidClasses: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool, len(classIDs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetValidClasses: scan: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

// GetValidParentIDs returns a set of valid active parent IDs for this tenant.
func (r *PgRepository) GetValidParentIDs(ctx context.Context, tenantID string, parentIDs []string) (map[string]bool, error) {
	if len(parentIDs) == 0 {
		return map[string]bool{}, nil
	}

	query := `
		SELECT id::text FROM cbc_parents
		WHERE tenant_id = $1 AND id = ANY($2) AND is_active = true
	`

	rows, err := r.pool.Query(ctx, query, tenantID, parentIDs)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.GetValidParentIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool, len(parentIDs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetValidParentIDs: scan: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

// BulkInsertStudents inserts validated students and returns (id, class_id) pairs.
func (r *PgRepository) BulkInsertStudents(ctx context.Context, tenantID string, students []ValidStudent) ([]StudentResult, error) {
	if len(students) == 0 {
		return nil, nil
	}

	// Use CTE with unnest arrays for bulk insert + RETURNING
	studentCount := len(students)
	fullNames := make([]string, studentCount)
	genders := make([]string, studentCount)
	dateOfBirths := make([]*string, studentCount)
	upiNumbers := make([]*string, studentCount)
	knecNumbers := make([]*string, studentCount)
	classIDs := make([]*string, studentCount)

	for i, s := range students {
		fullNames[i] = s.FullName
		genders[i] = s.Gender
		dateOfBirths[i] = s.DateOfBirth
		upiNumbers[i] = s.UPINumber
		knecNumbers[i] = s.KNECAssessmentNumber
		classIDs[i] = s.ClassID
	}

	// Build a row_number -> class_id mapping using the same row_number ordering
	// Actually, we can't join back easily. Let's use a different approach:
	// Insert all students, then query back using full_name (not ideal).
	// Better: use RETURNING with a CTE that pairs input rows.

	// Rewrite: use a VALUES-based CTE approach
	valueStrings := make([]string, 0, studentCount)
	args := make([]interface{}, 0, studentCount*7+1)
	args = append(args, tenantID)
	argIdx := 2

	for i, s := range students {
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::int, $%d::text, $%d::gender_type, $%d::date, $%d::text, $%d::text, $%d::text)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6),
		)
		dob := interface{}(nil)
		if s.DateOfBirth != nil {
			dob = *s.DateOfBirth
		}
		upi := interface{}(nil)
		if s.UPINumber != nil {
			upi = *s.UPINumber
		}
		knec := interface{}(nil)
		if s.KNECAssessmentNumber != nil {
			knec = *s.KNECAssessmentNumber
		}
		classID := interface{}(nil)
		if s.ClassID != nil {
			classID = *s.ClassID
		}

		args = append(args, i, s.FullName, s.Gender, dob, upi, knec, classID)
		argIdx += 7
	}

	query := `
		WITH input_rows (rn, full_name, gender, date_of_birth, upi_number, knec_assessment_number, class_id) AS (
			VALUES ` + strings.Join(valueStrings, ",\n			       ") + `
		),
		inserted AS (
			INSERT INTO cbc_students (tenant_id, full_name, gender, date_of_birth, upi_number, knec_assessment_number)
			SELECT $1, ir.full_name, ir.gender, ir.date_of_birth, ir.upi_number, ir.knec_assessment_number
			FROM input_rows ir
			RETURNING id, full_name
		)
		SELECT i.id::text, ir.class_id
		FROM inserted i
		JOIN input_rows ir ON i.full_name = ir.full_name
		   AND (i.date_of_birth IS NOT DISTINCT FROM ir.date_of_birth)
		ORDER BY ir.rn
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.BulkInsertStudents: %w", err)
	}
	defer rows.Close()

	results := make([]StudentResult, 0, studentCount)
	for rows.Next() {
		var sr StudentResult
		var classID *string
		if err := rows.Scan(&sr.StudentID, &classID); err != nil {
			return nil, fmt.Errorf("imports.Repository.BulkInsertStudents: scan: %w", err)
		}
		if classID != nil && *classID != "" {
			sr.ClassID = classID
		}
		results = append(results, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.BulkInsertStudents: rows: %w", err)
	}
	return results, nil
}

// BulkInsertEnrollments inserts enrollment rows in a single bulk query.
func (r *PgRepository) BulkInsertEnrollments(ctx context.Context, tenantID, schoolID, academicTermID string, enrollments []StudentResult) error {
	if len(enrollments) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(enrollments))
	args := make([]interface{}, 0, len(enrollments)*4+1)
	args = append(args, academicTermID, tenantID, schoolID)
	argIdx := 4

	for _, e := range enrollments {
		if e.ClassID == nil {
			continue
		}
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::uuid, $%d::uuid)", argIdx, argIdx+1),
		)
		args = append(args, e.StudentID, *e.ClassID)
		argIdx += 2
	}

	if len(valueStrings) == 0 {
		return nil
	}

	query := `
		INSERT INTO cbc_student_enrollments (tenant_id, school_id, student_id, academic_term_id, class_id)
		SELECT $2, $3, v.student_id, $1, v.class_id
		FROM (VALUES ` + strings.Join(valueStrings, ",\n		       ") + `
		) AS v(student_id, class_id)
		ON CONFLICT (student_id, academic_term_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.BulkInsertEnrollments: %w", err)
	}
	return nil
}

// ResolveAcademicTerm resolves academic_year name + term name to an academic_term_id UUID.
func (r *PgRepository) ResolveAcademicTerm(ctx context.Context, tenantID, schoolID, academicYear, term string) (string, error) {
	const query = `
		SELECT t.id::text FROM academic_terms t
		JOIN academic_years y ON t.academic_year_id = y.id
		WHERE y.tenant_id = $1 AND y.school_id = $2 AND y.name = $3 AND t.name = $4
		LIMIT 1
	`

	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID, academicYear, term).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("imports.Repository.ResolveAcademicTerm: academic term not found for year=%q term=%q", academicYear, term)
		}
		return "", fmt.Errorf("imports.Repository.ResolveAcademicTerm: %w", err)
	}
	return id, nil
}

// BulkInsertFailures inserts failure rows in a single query.
func (r *PgRepository) BulkInsertFailures(ctx context.Context, jobID string, failures []FailedRow) error {
	if len(failures) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(failures))
	args := make([]interface{}, 0, len(failures)*2+1)
	args = append(args, jobID)
	argIdx := 2

	for _, f := range failures {
		rawJSON, err := json.Marshal(f.RawData)
		if err != nil {
			rawJSON = []byte(`{"marshal_error": "` + err.Error() + `"}`)
		}
		valueStrings = append(valueStrings,
			fmt.Sprintf("($1, $%d::jsonb, $%d)", argIdx, argIdx+1),
		)
		args = append(args, string(rawJSON), f.ErrorMessage)
		argIdx += 2
	}

	query := `
		INSERT INTO import_job_failures (import_job_id, raw_payload, error_message)
		VALUES ` + strings.Join(valueStrings, ",\n		       ") + `
	`

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("imports.Repository.BulkInsertFailures: %w", err)
	}
	return nil
}

// PurgeStaging deletes all staging rows for a completed job.
func (r *PgRepository) PurgeStaging(ctx context.Context, jobID string) error {
	const query = `DELETE FROM import_job_staging WHERE job_id = $1`
	_, err := r.pool.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("imports.Repository.PurgeStaging: %w", err)
	}
	return nil
}

// GetAcademicYears returns all academic years for a tenant+school.
func (r *PgRepository) GetAcademicYears(ctx context.Context, tenantID, schoolID string) ([]AcademicYearRecord, error) {
	const query = `
		SELECT id::text, name, start_date::text, end_date::text, is_current
		FROM academic_years
		WHERE tenant_id = $1 AND school_id = $2
		ORDER BY start_date DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.GetAcademicYears: %w", err)
	}
	defer rows.Close()

	var results []AcademicYearRecord
	for rows.Next() {
		var rec AcademicYearRecord
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.StartDate, &rec.EndDate, &rec.IsCurrent); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetAcademicYears: scan: %w", err)
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.GetAcademicYears: rows: %w", err)
	}
	return results, nil
}

// GetAcademicPeriods returns all academic periods (terms) for a given academic year.
func (r *PgRepository) GetAcademicPeriods(ctx context.Context, tenantID, schoolID, academicYearID string) ([]AcademicPeriodRecord, error) {
	const query = `
		SELECT id::text, name, term_number, start_date::text, end_date::text, is_current
		FROM academic_terms
		WHERE tenant_id = $1 AND school_id = $2 AND academic_year_id = $3
		ORDER BY term_number ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.GetAcademicPeriods: %w", err)
	}
	defer rows.Close()

	var results []AcademicPeriodRecord
	for rows.Next() {
		var rec AcademicPeriodRecord
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.TermNumber, &rec.StartDate, &rec.EndDate, &rec.IsCurrent); err != nil {
			return nil, fmt.Errorf("imports.Repository.GetAcademicPeriods: scan: %w", err)
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.GetAcademicPeriods: rows: %w", err)
	}
	return results, nil
}

// ============================================================================
// Lookup Methods for Student Import
// ============================================================================

// ListParents returns all active parents for a tenant+school.
func (r *PgRepository) ListParents(ctx context.Context, tenantID, schoolID string) ([]ParentRecord, error) {
	const query = `
		SELECT p.id::text, u.full_name, p.phone_number, u.email
		FROM cbc_parents p
		JOIN users u ON u.id = p.user_id AND u.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1
		  AND p.is_active = true
		ORDER BY u.full_name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.ListParents: %w", err)
	}
	defer rows.Close()

	var results []ParentRecord
	for rows.Next() {
		var rec ParentRecord
		if err := rows.Scan(&rec.ID, &rec.FullName, &rec.Phone, &rec.Email); err != nil {
			return nil, fmt.Errorf("imports.Repository.ListParents: scan: %w", err)
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.ListParents: rows: %w", err)
	}
	if results == nil {
		results = []ParentRecord{}
	}
	return results, nil
}

// ListClasses returns all active classes for a tenant+school.
func (r *PgRepository) ListClasses(ctx context.Context, tenantID, schoolID string) ([]ClassRecord, error) {
	const query = `
		SELECT id::text, name
		FROM cbc_classes
		WHERE tenant_id = $1 AND school_id = $2 AND is_active = true
		ORDER BY grade_level ASC, stream ASC, name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.ListClasses: %w", err)
	}
	defer rows.Close()

	var results []ClassRecord
	for rows.Next() {
		var rec ClassRecord
		if err := rows.Scan(&rec.ID, &rec.Name); err != nil {
			return nil, fmt.Errorf("imports.Repository.ListClasses: scan: %w", err)
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.ListClasses: rows: %w", err)
	}
	if results == nil {
		results = []ClassRecord{}
	}
	return results, nil
}

// ListExistingStudents returns all existing students for a tenant+school
// for duplicate detection during import.
func (r *PgRepository) ListExistingStudents(ctx context.Context, tenantID, schoolID string) ([]ExistingStudentRecord, error) {
	const query = `
		SELECT s.full_name, s.date_of_birth::text, s.upi_number
		FROM cbc_students s
		JOIN cbc_student_enrollments e ON e.student_id = s.id AND e.tenant_id = s.tenant_id
		WHERE s.tenant_id = $1
		  AND e.school_id = $2
		GROUP BY s.id, s.full_name, s.date_of_birth, s.upi_number
		ORDER BY s.full_name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("imports.Repository.ListExistingStudents: %w", err)
	}
	defer rows.Close()

	var results []ExistingStudentRecord
	for rows.Next() {
		var rec ExistingStudentRecord
		if err := rows.Scan(&rec.FullName, &rec.DateOfBirth, &rec.UPINumber); err != nil {
			return nil, fmt.Errorf("imports.Repository.ListExistingStudents: scan: %w", err)
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imports.Repository.ListExistingStudents: rows: %w", err)
	}
	if results == nil {
		results = []ExistingStudentRecord{}
	}
	return results, nil
}

// GetImportJobStatus returns the current status, total_records, and school_id of a job.
func (r *PgRepository) GetImportJobStatus(ctx context.Context, jobID string) (string, int, string, error) {
	const query = `
		SELECT status, total_records, school_id FROM import_jobs WHERE id = $1
	`

	var status string
	var totalRecords int
	var schoolID string
	err := r.pool.QueryRow(ctx, query, jobID).Scan(&status, &totalRecords, &schoolID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", 0, "", fmt.Errorf("imports.Repository.GetImportJobStatus: %w", ErrNotFound)
		}
		return "", 0, "", fmt.Errorf("imports.Repository.GetImportJobStatus: %w", err)
	}
	return status, totalRecords, schoolID, nil
}

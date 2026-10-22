-- name: CreateBulkJob :one
INSERT INTO bulk_jobs (job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;

-- name: GetBulkJob :one
SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE id=$1;

-- name: UpdateBulkJobStatus :exec
UPDATE bulk_jobs SET status=$1, updated_at=NOW() WHERE id=$2;

-- name: UpdateBulkJobItem :exec
UPDATE bulk_job_items SET status=$1, result=$2, last_error=$3, attempt_count=$4, updated_at=NOW() WHERE id=$5;

-- name: UpdateBulkJobItemResultStatus :exec
UPDATE bulk_job_items SET result=$1, status=$2, last_error=$3, updated_at=NOW() WHERE id=$4;

-- name: IncrementBulkJobCounts :exec
UPDATE bulk_jobs SET succeeded_count = succeeded_count + $1, failed_count = failed_count + $2, deferred_count = deferred_count + $3, updated_at = NOW() WHERE id=$4;

-- name: GetBulkJobByTenantAndIdempotency :one
SELECT id, job_type, idempotency_key, school_id, tenant_id, created_by, status, total_records, succeeded_count, failed_count, deferred_count, metadata, created_at, updated_at FROM bulk_jobs WHERE tenant_id=$1 AND idempotency_key=$2;

-- name: GetTenantStytchOrgID :one
SELECT stytch_org_id FROM tenants WHERE id=$1;

-- name: GetSchoolMembershipRole :one
SELECT role FROM school_memberships WHERE school_id=$1 AND user_id=$2 AND is_active=true;

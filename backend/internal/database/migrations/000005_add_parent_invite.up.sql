-- Migration: 000005_add_parent_invite
-- Adds PARENT_INVITE to the import_job_type enum and updates the CHECK constraint
-- on import_jobs to require role for both STAFF_INVITE and PARENT_INVITE job types.
--
-- ALTER TYPE ... ADD VALUE cannot be done inside a transaction block once the
-- enum type is referenced by a table. It must run as its own DDL statement
-- outside any explicit transaction (following the pattern set by 000003).

ALTER TYPE import_job_type ADD VALUE IF NOT EXISTS 'PARENT_INVITE';

BEGIN;

-- Update the CHECK constraint to require role for both STAFF_INVITE and PARENT_INVITE
ALTER TABLE import_jobs DROP CONSTRAINT IF EXISTS chk_import_jobs_role_required_for_staff;
ALTER TABLE import_jobs ADD CONSTRAINT chk_import_jobs_role_required_for_staff
    CHECK (
        (job_type IN ('STAFF_INVITE', 'PARENT_INVITE') AND role IS NOT NULL)
        OR (job_type NOT IN ('STAFF_INVITE', 'PARENT_INVITE'))
    );

COMMIT;

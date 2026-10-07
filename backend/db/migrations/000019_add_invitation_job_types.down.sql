-- Migration: 000019_add_invitation_job_types (down)
-- Purpose: Revert bulk_jobs job_type check to previous state

ALTER TABLE bulk_jobs DROP CONSTRAINT IF EXISTS bulk_jobs_job_type_check;
ALTER TABLE bulk_jobs ADD CONSTRAINT bulk_jobs_job_type_check
    CHECK (job_type IN ('ADMIN_INVITATION','STUDENT_IMPORT'));

COMMENT ON COLUMN bulk_jobs.job_type IS 'Bulk operation category. Extendable via CHECK constraint or lookup table migration. Supported: ADMIN_INVITATION, STUDENT_IMPORT.';
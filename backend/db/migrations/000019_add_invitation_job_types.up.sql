-- Migration: 000019_add_invitation_job_types
-- Purpose: Extend bulk_jobs job_type check to include TEACHER_INVITATION, GUARDIAN_INVITATION, FINANCE_INVITATION
-- Dependencies:
--   - 000017_add_student_bulk_import_support (bulk_jobs job_type check)

-- ============================================================================
-- Section 1: Extend bulk_jobs job_type check
-- ============================================================================

ALTER TABLE bulk_jobs DROP CONSTRAINT IF EXISTS bulk_jobs_job_type_check;
ALTER TABLE bulk_jobs ADD CONSTRAINT bulk_jobs_job_type_check
    CHECK (job_type IN ('ADMIN_INVITATION','STUDENT_IMPORT','TEACHER_INVITATION','GUARDIAN_INVITATION','FINANCE_INVITATION'));

COMMENT ON COLUMN bulk_jobs.job_type IS 'Bulk operation category. Extendable via CHECK constraint or lookup table migration. Supported: ADMIN_INVITATION, STUDENT_IMPORT, TEACHER_INVITATION, GUARDIAN_INVITATION, FINANCE_INVITATION.';
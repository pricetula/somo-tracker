-- Migration: 000005_add_parent_invite (down)
-- Reverts the CHECK constraint. PARENT_INVITE remains in the enum type since
-- PostgreSQL does not allow removing individual values from an enum.

BEGIN;

-- Restore original CHECK constraint
ALTER TABLE import_jobs DROP CONSTRAINT IF EXISTS chk_import_jobs_role_required_for_staff;
ALTER TABLE import_jobs ADD CONSTRAINT chk_import_jobs_role_required_for_staff
    CHECK (job_type <> 'STAFF_INVITE' OR role IS NOT NULL);

COMMIT;

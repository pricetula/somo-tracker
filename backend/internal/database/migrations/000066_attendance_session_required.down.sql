-- Migration: 000066_attendance_session_required (revert)
-- SomoTracker
-- Purpose: revert attendance_session_id back to nullable FK.

BEGIN;

-- Drop the mandatory FK and re-add the nullable version with ON DELETE SET NULL
ALTER TABLE attendance_records
    DROP CONSTRAINT IF EXISTS fk_attendance_tenant_session;

ALTER TABLE attendance_records
    ALTER COLUMN attendance_session_id DROP NOT NULL;

ALTER TABLE attendance_records
    ADD CONSTRAINT fk_attendance_tenant_session
        FOREIGN KEY (tenant_id, attendance_session_id)
        REFERENCES cbc_attendance_sessions(tenant_id, id)
        ON DELETE SET NULL;

COMMENT ON COLUMN attendance_records.attendance_session_id IS
    'Optional reference to the cbc_attendance_sessions row. Populated when
     session is marked as SKIPPED to link existing records. NULL for normal
     (non-skipped) attendance marks.';

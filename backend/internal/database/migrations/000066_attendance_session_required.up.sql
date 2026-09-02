-- Migration: 000066_attendance_session_required
-- SomoTracker
-- Purpose: make attendance_session_id in attendance_records a required (NOT NULL) FK.
--
-- Rationale: every attendance record must belong to a session.
-- BatchMark always creates the session before the records; the only reason this
-- column was nullable was a migration-order artifact from when it was added to
-- support SKIPPED sessions linking their (pre-existing) records.
--
-- This migration:
--   1. Back-fills NULL attendance_session_id values by looking up the session
--      for each (student_id, timetable_allocation_id, date) combination.
--   2. Drops the existing FK (fk_attendance_tenant_session) which had
--      ON DELETE SET NULL.
--   3. Alters the column to NOT NULL.
--   4. Adds a new FK that disallows NULL and uses RESTRICT instead of SET NULL.

BEGIN;

-- Step 1: backfill NULL attendance_session_id using the existing sub-query pattern
-- that BatchMark uses (session is created by upserting cbc_attendance_sessions
-- before records are written).
UPDATE attendance_records ar
SET attendance_session_id = s.id
FROM cbc_attendance_sessions s
WHERE
    ar.attendance_session_id IS NULL
    AND s.tenant_id = ar.tenant_id
    AND s.timetable_allocation_id = ar.timetable_allocation_id
    AND s.date = ar.date
    AND s.status = 'SUBMITTED';

-- Verify no NULLs remain after backfill. Fail loudly if any do.
DO $$
DECLARE
    null_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO null_count
    FROM attendance_records
    WHERE attendance_session_id IS NULL;

    IF null_count > 0 THEN
        RAISE EXCEPTION 'Cannot enforce NOT NULL on attendance_session_id: % rows still have NULL after backfill', null_count
            USING ERRCODE = 'P0001';
    END IF;
END;
$$;

-- Step 2: drop the nullable FK (cascade to remove any FK index if present)
ALTER TABLE attendance_records
    DROP CONSTRAINT IF EXISTS fk_attendance_tenant_session;

-- Step 3: make the column NOT NULL
ALTER TABLE attendance_records
    ALTER COLUMN attendance_session_id SET NOT NULL;

-- Step 4: re-add the FK with RESTRICT (no SET NULL — deletes must be explicit)
ALTER TABLE attendance_records
    ADD CONSTRAINT fk_attendance_tenant_session
        FOREIGN KEY (tenant_id, attendance_session_id)
        REFERENCES cbc_attendance_sessions(tenant_id, id)
        ON DELETE RESTRICT;

-- Update the column comment to reflect the new mandatory semantics
COMMENT ON COLUMN attendance_records.attendance_session_id IS
    'Reference to the cbc_attendance_sessions row for this slot+date.
     Always populated when a record is created or updated (NOT NULL).
     Set to the SUBMITTED session created by BatchMark.';

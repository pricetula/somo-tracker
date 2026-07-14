-- Migration: 000003_attendance_updated_at_nonbreak
-- SomoTracker — Kenya CBC/CBE academic platform
-- Adds updated_at tracking and non-break-period enforcement to attendance_records.
-- ============================================================================

-- ============================================================================
-- 1. ADD updated_at COLUMN
-- ============================================================================

ALTER TABLE attendance_records
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DROP TRIGGER IF EXISTS trg_attendance_records_updated_at ON attendance_records;
CREATE TRIGGER trg_attendance_records_updated_at
    BEFORE UPDATE ON attendance_records
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN attendance_records.updated_at IS
    'Tracks when the record was last modified (status or note update).
     Populated automatically by the trg_attendance_records_updated_at trigger.';

-- ============================================================================
-- 2. NON-BREAK SLOT CONSTRAINT
-- Enforces that attendance records can only reference timetable slots
-- whose corresponding structure period is NOT a break period.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_check_non_break_slot()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM cbc_timetable_slots ts
        JOIN timetable_structures tstr ON tstr.id = ts.structure_id
        WHERE ts.id = NEW.timetable_slot_id
          AND tstr.is_break = true
    ) THEN
        RAISE EXCEPTION 'Cannot create attendance record for a break period (timetable_slot_id: %)', NEW.timetable_slot_id
            USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_attendance_check_non_break_slot ON attendance_records;
CREATE TRIGGER trg_attendance_check_non_break_slot
    BEFORE INSERT OR UPDATE ON attendance_records
    FOR EACH ROW
    EXECUTE FUNCTION fn_check_non_break_slot();

COMMENT ON TRIGGER trg_attendance_check_non_break_slot ON attendance_records IS
    'Enforces that attendance records can only reference timetable slots
     whose corresponding timetable_structures row has is_break = false.
     Prevents system or application bugs from creating attendance marks
     for break/assembly/recess periods.';

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

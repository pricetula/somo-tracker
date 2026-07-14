-- Migration: 000003_attendance_updated_at_nonbreak (rollback)
-- Reverses every change made by the up migration.

DROP TRIGGER IF EXISTS trg_attendance_records_updated_at ON attendance_records;
DROP TRIGGER IF EXISTS trg_attendance_check_non_break_slot ON attendance_records;
DROP FUNCTION IF EXISTS fn_check_non_break_slot CASCADE;
ALTER TABLE attendance_records DROP COLUMN IF EXISTS updated_at;

-- ============================================================================
-- END OF ROLLBACK
-- ============================================================================

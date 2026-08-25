-- Migration: 000039_attendance_records (rollback)
-- SomoTracker — rollback for 000039_attendance_records.

DROP FUNCTION IF EXISTS fn_check_non_break_slot CASCADE;

DROP TABLE IF EXISTS attendance_records CASCADE;

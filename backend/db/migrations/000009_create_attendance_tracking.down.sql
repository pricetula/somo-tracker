-- Migration: 000009_create_attendance_tracking (down)
-- Purpose: Rollback the attendance tracking schema:
--   timetable_attendance, event_attendance,
--   timetable_attendance_status, event_attendance_status enums.
--
-- Order of removal is the strict inverse of creation:
--   1. RLS policies
--   2. updated_at triggers
--   3. Indexes (explicit removal for clarity)
--   4. Unique constraints
--   5. Tables
--   6. Enum types

-- ============================================================================
-- Section 1: Drop RLS policies
-- ============================================================================
DROP POLICY IF EXISTS event_attendance_tenant_isolation ON event_attendance;
DROP POLICY IF EXISTS timetable_attendance_tenant_isolation ON timetable_attendance;

-- ============================================================================
-- Section 2: Drop updated_at triggers
-- ============================================================================
DROP TRIGGER IF EXISTS event_attendance_updated_at_trg ON event_attendance;
DROP TRIGGER IF EXISTS timetable_attendance_updated_at_trg ON timetable_attendance;

-- ============================================================================
-- Section 3: Drop indexes on timetable_attendance
-- ============================================================================
DROP INDEX IF EXISTS timetable_attendance_student_date_idx;
DROP INDEX IF EXISTS timetable_attendance_slot_date_idx;
DROP INDEX IF EXISTS timetable_attendance_recorded_by_idx;
DROP INDEX IF EXISTS timetable_attendance_date_idx;
DROP INDEX IF EXISTS timetable_attendance_slot_id_idx;
DROP INDEX IF EXISTS timetable_attendance_student_id_idx;
DROP INDEX IF EXISTS timetable_attendance_school_id_idx;

-- ============================================================================
-- Section 4: Drop indexes on event_attendance
-- ============================================================================
DROP INDEX IF EXISTS event_attendance_student_event_idx;
DROP INDEX IF EXISTS event_attendance_date_idx;
DROP INDEX IF EXISTS event_attendance_event_id_idx;
DROP INDEX IF EXISTS event_attendance_student_id_idx;
DROP INDEX IF EXISTS event_attendance_school_id_idx;

-- ============================================================================
-- Section 5: Drop unique constraints
-- ============================================================================
ALTER TABLE timetable_attendance DROP CONSTRAINT IF EXISTS timetable_attendance_uniq_student_slot_date;
ALTER TABLE event_attendance DROP CONSTRAINT IF EXISTS event_attendance_uniq_student_event_date;

-- ============================================================================
-- Section 6: Drop tables
-- ============================================================================
DROP TABLE IF EXISTS event_attendance;
DROP TABLE IF EXISTS timetable_attendance;

-- ============================================================================
-- Section 7: Drop enum types
-- ============================================================================
DROP TYPE IF EXISTS event_attendance_status;
DROP TYPE IF EXISTS timetable_attendance_status;

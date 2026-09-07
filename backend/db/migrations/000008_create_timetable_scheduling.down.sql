-- Migration: 000008_create_timetable_scheduling (down)
-- Purpose: Rollback the timetable & scheduling schema:
--   timetable_templates, time_slots, rooms, class_timetable_slots,
--   timetable_substitutions, substitution_status enum, room_type enum.
--
-- Order of removal is the strict inverse of creation:
--   1. RLS policies (must exist before table drop, but policy lives on table)
--   2. updated_at triggers
--   3. Indexes (explicit removal for clarity)
--   4. Tables
--   5. Enum types
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- ============================================================================
-- Section 1: Drop RLS policies
-- ============================================================================
DROP POLICY IF EXISTS timetable_substitutions_tenant_isolation ON timetable_substitutions;
DROP POLICY IF EXISTS class_timetable_slots_tenant_isolation ON class_timetable_slots;
DROP POLICY IF EXISTS rooms_tenant_isolation ON rooms;
DROP POLICY IF EXISTS time_slots_tenant_isolation ON time_slots;
DROP POLICY IF EXISTS timetable_templates_tenant_isolation ON timetable_templates;

-- ============================================================================
-- Section 2: Drop updated_at triggers
-- ============================================================================
DROP TRIGGER IF EXISTS timetable_substitutions_updated_at_trg ON timetable_substitutions;
DROP TRIGGER IF EXISTS class_timetable_slots_updated_at_trg ON class_timetable_slots;
DROP TRIGGER IF EXISTS rooms_updated_at_trg ON rooms;
DROP TRIGGER IF EXISTS time_slots_updated_at_trg ON time_slots;
DROP TRIGGER IF EXISTS timetable_templates_updated_at_trg ON timetable_templates;

-- ============================================================================
-- Section 3: Drop indexes on timetable_substitutions
-- ============================================================================
DROP INDEX IF EXISTS timetable_substitutions_date_status_idx;
DROP INDEX IF EXISTS timetable_substitutions_status_idx;
DROP INDEX IF EXISTS timetable_substitutions_substitute_teacher_idx;
DROP INDEX IF EXISTS timetable_substitutions_original_teacher_idx;
DROP INDEX IF EXISTS timetable_substitutions_substitution_date_idx;
DROP INDEX IF EXISTS timetable_substitutions_class_timetable_slot_id_idx;
DROP INDEX IF EXISTS timetable_substitutions_school_id_idx;

-- ============================================================================
-- Section 4: Drop unique constraint on timetable_substitutions
-- ============================================================================
ALTER TABLE timetable_substitutions DROP CONSTRAINT IF EXISTS timetable_substitutions_slot_date_uniq;

-- ============================================================================
-- Section 5: Drop indexes on class_timetable_slots
-- ============================================================================
DROP INDEX IF EXISTS class_timetable_slots_day_time_idx;
DROP INDEX IF EXISTS class_timetable_slots_room_id_idx;
DROP INDEX IF EXISTS class_timetable_slots_teacher_membership_id_idx;
DROP INDEX IF EXISTS class_timetable_slots_academic_term_id_idx;
DROP INDEX IF EXISTS class_timetable_slots_class_room_id_idx;
DROP INDEX IF EXISTS class_timetable_slots_school_id_idx;

-- ============================================================================
-- Section 6: Drop unique constraints on class_timetable_slots
-- ============================================================================
ALTER TABLE class_timetable_slots DROP CONSTRAINT IF EXISTS class_timetable_slots_class_day_time_uniq;
ALTER TABLE class_timetable_slots DROP CONSTRAINT IF EXISTS class_timetable_slots_teacher_no_clash;

-- ============================================================================
-- Section 7: Drop indexes on rooms
-- ============================================================================
DROP INDEX IF EXISTS rooms_capacity_idx;
DROP INDEX IF EXISTS rooms_room_type_idx;
DROP INDEX IF EXISTS rooms_school_id_idx;

-- ============================================================================
-- Section 8: Drop unique constraint on rooms
-- ============================================================================
ALTER TABLE rooms DROP CONSTRAINT IF EXISTS rooms_school_name_uniq;

-- ============================================================================
-- Section 9: Drop indexes on time_slots
-- ============================================================================
DROP INDEX IF EXISTS time_slots_sequence_idx;
DROP INDEX IF EXISTS time_slots_timetable_template_id_idx;
DROP INDEX IF EXISTS time_slots_school_id_idx;

-- ============================================================================
-- Section 10: Drop unique constraints on time_slots
-- ============================================================================
ALTER TABLE time_slots DROP CONSTRAINT IF EXISTS time_slots_time_order;
ALTER TABLE time_slots DROP CONSTRAINT IF EXISTS time_slots_template_seq_uniq;

-- ============================================================================
-- Section 11: Drop indexes on timetable_templates
-- ============================================================================
DROP INDEX IF EXISTS timetable_templates_name_idx;
DROP INDEX IF EXISTS timetable_templates_school_id_idx;

-- ============================================================================
-- Section 12: Drop unique constraint on timetable_templates
-- ============================================================================
ALTER TABLE timetable_templates DROP CONSTRAINT IF EXISTS timetable_templates_school_name_uniq;

-- ============================================================================
-- Section 13: Drop tables
-- ============================================================================
DROP TABLE IF EXISTS timetable_substitutions;
DROP TABLE IF EXISTS class_timetable_slots;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS time_slots;
DROP TABLE IF EXISTS timetable_templates;

-- ============================================================================
-- Section 14: Drop enum types
-- ============================================================================
DROP TYPE IF EXISTS substitution_status;
DROP TYPE IF EXISTS room_type;
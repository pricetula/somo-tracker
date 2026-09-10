-- Migration: 000007_create_class_rooms_and_enrollments (down)
-- Purpose: Reverse the creation of class_rooms, student_class_enrollments,
--           enrollment_status enum, RLS policies, and indexes.
--
-- Order of removal is the strict inverse of creation:
--   1. RLS policies (must exist before table drop, but policy lives on table)
--   2. updated_at triggers
--   3. Indexes (explicit removal for clarity)
--   4. Tables
--   5. Enum type
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- ============================================================================
-- Section 1: Drop RLS policy on student_class_enrollments
-- ============================================================================
DROP POLICY IF EXISTS student_class_enrollments_tenant_isolation ON student_class_enrollments;

-- ============================================================================
-- Section 2: Drop updated_at triggers
-- ============================================================================
DROP TRIGGER IF EXISTS class_rooms_updated_at_trg ON class_rooms;
DROP TRIGGER IF EXISTS student_class_enrollments_updated_at_trg ON student_class_enrollments;

-- ============================================================================
-- Section 3: Drop indexes on class_rooms
-- ============================================================================
DROP INDEX IF EXISTS class_rooms_school_id_idx;
DROP INDEX IF EXISTS class_rooms_academic_year_id_idx;
DROP INDEX IF EXISTS class_rooms_grade_level_id_idx;
DROP INDEX IF EXISTS class_rooms_name_idx;

-- ============================================================================
-- Section 4: Drop unique index on class_rooms
-- ============================================================================
DROP INDEX IF EXISTS class_rooms_school_year_grade_stream_uniq;

-- ============================================================================
-- Section 5: Drop indexes on student_class_enrollments
-- ============================================================================
DROP INDEX IF EXISTS student_class_enrollments_school_id_idx;
DROP INDEX IF EXISTS student_class_enrollments_student_id_idx;
DROP INDEX IF EXISTS student_class_enrollments_class_room_id_idx;
DROP INDEX IF EXISTS student_class_enrollments_academic_year_id_idx;
DROP INDEX IF EXISTS student_class_enrollments_academic_term_id_idx;
DROP INDEX IF EXISTS student_class_enrollments_status_idx;
DROP INDEX IF EXISTS student_class_enrollments_student_term_uniq;

-- ============================================================================
-- Section 5b: Drop unique constraints on student_class_enrollments
-- ============================================================================
ALTER TABLE student_class_enrollments DROP CONSTRAINT IF EXISTS student_class_enrollments_student_year_uniq;

-- ============================================================================
-- Section 6: Drop tables
-- ============================================================================
DROP TABLE IF EXISTS student_class_enrollments;
DROP TABLE IF EXISTS class_rooms;

-- ============================================================================
-- Section 7: Drop enum type
-- ============================================================================
DROP TYPE IF EXISTS enrollment_status;

-- Migration: 000022_add_kenya_identifiers
-- Purpose: Add Kenya-specific national identifiers for schools, teachers and students.
--   - schools.nemis_school_code
--   - school_memberships.tsc_number for teachers
--   - students.nemis_upn
-- Keeps existing metadata JSONB for extensibility.
--
-- Dependencies:
--   - 000004_create_sis_schema (schools, school_memberships, students)
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: schools - NEMIS school code
-- ============================================================================

ALTER TABLE schools
    ADD COLUMN nemis_school_code VARCHAR(32);

-- NEMIS school code is unique within Kenya; enforce globally unique for now
-- with a nullable unique index. Nulls are allowed for non-Kenya schools.
CREATE UNIQUE INDEX schools_nemis_school_code_uniq_idx
    ON schools (nemis_school_code)
    WHERE nemis_school_code IS NOT NULL;

CREATE INDEX schools_nemis_school_code_idx ON schools (nemis_school_code);

COMMENT ON COLUMN schools.nemis_school_code IS 'Kenya NEMIS school code. Nullable for non-Kenya schools. Unique when present.';

-- ============================================================================
-- Section 2: school_memberships - TSC number
-- ============================================================================

ALTER TABLE school_memberships
    ADD COLUMN tsc_number VARCHAR(32);

-- TSC number is national unique for teachers. Enforce uniqueness only for TEACHER role
-- and when tsc_number is present.
CREATE UNIQUE INDEX school_memberships_tsc_number_uniq_idx
    ON school_memberships (tsc_number)
    WHERE tsc_number IS NOT NULL;

CREATE INDEX school_memberships_tsc_number_idx ON school_memberships (tsc_number);

COMMENT ON COLUMN school_memberships.tsc_number IS 'Kenya TSC registration number for TEACHER memberships. Nullable for other roles and non-Kenya schools.';

-- Optional check: tsc_number should only be set for TEACHER role
ALTER TABLE school_memberships
    ADD CONSTRAINT school_memberships_tsc_number_role_check
    CHECK (
        tsc_number IS NULL OR role = 'TEACHER'
    );

-- ============================================================================
-- Section 3: students - NEMIS UPN
-- ============================================================================

ALTER TABLE students
    ADD COLUMN nemis_upn VARCHAR(32);

-- UPN is nationally unique. Enforce unique when present.
CREATE UNIQUE INDEX students_nemis_upn_uniq_idx
    ON students (nemis_upn)
    WHERE nemis_upn IS NOT NULL;

CREATE INDEX students_nemis_upn_idx ON students (nemis_upn);

COMMENT ON COLUMN students.nemis_upn IS 'Kenya NEMIS Unique Pupil Number / Learner ID. Nullable for non-Kenya students. Unique when present.';

-- ============================================================================
-- Section 4: updated_at triggers already exist for these tables
-- No new triggers needed.
-- ============================================================================

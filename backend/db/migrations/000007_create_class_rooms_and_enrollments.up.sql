-- Migration: 000007_create_class_rooms_and_enrollments
-- Purpose: Create class_rooms (operational container per year/stream) and
--   student_class_enrollments (historical student tracking across academic years
--   and terms). Enrollments map a student to a specific class_room for a given
--   time frame, ensuring that past data (attendance, assessments, report cards)
--   remains permanently locked and unaltered even when students change streams
--   or get promoted.
--
-- Dependencies:
--   - 000001_init_extensions (pgcrypto for gen_random_uuid()).
--   - 000004_create_sis_schema (provides schools, students, grade_levels tables).
--   - 000006_create_academic_calendar (provides academic_years, academic_terms tables).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: enrollment_status enum
-- ============================================================================

-- Tracks a student's lifecycle within a class for a given year/term.
CREATE TYPE enrollment_status AS ENUM ('ACTIVE', 'PROMOTED', 'REPEATING', 'GRADUATED');

-- ============================================================================
-- Section 2: class_rooms
-- ============================================================================

-- Operational classroom container for a specific academic year and stream.
-- A class_room represents a grade-level + stream combination (e.g., "Class 1
-- Blue", "Class 3 Yellow") within a given academic year. Each year creates new
-- class_room entries, preserving the historical context for past enrollments.
CREATE TABLE class_rooms (
    id               UUID        NOT NULL    DEFAULT gen_random_uuid()
                                        PRIMARY KEY,
    school_id        UUID        NOT NULL    REFERENCES schools(id)
                                        ON DELETE CASCADE,
    academic_year_id UUID        NOT NULL    REFERENCES academic_years(id)
                                        ON DELETE CASCADE,
    grade_level_id   UUID        NOT NULL    REFERENCES grade_levels(id)
                                        ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    stream           VARCHAR(64)  DEFAULT NULL,
    created_at       TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: class_rooms_school_id_idx covers school-scoped class_room listings.
CREATE INDEX class_rooms_school_id_idx ON class_rooms (school_id);

-- Index: class_rooms_academic_year_id_idx covers year-scoped class_room queries.
CREATE INDEX class_rooms_academic_year_id_idx ON class_rooms (academic_year_id);

-- Index: class_rooms_grade_level_id_idx covers grade-level aggregations.
CREATE INDEX class_rooms_grade_level_id_idx ON class_rooms (grade_level_id);

-- Index: class_rooms_name_idx supports name-based class_room lookups.
CREATE INDEX class_rooms_name_idx ON class_rooms (name);

-- Constraint: class_rooms_school_year_grade_stream_uniq ensures a school
-- cannot have duplicate class_rooms for the same year/grade/stream combination.
-- Stream may be NULL (e.g., "Class 1" without a stream), so we use
-- NULLIF(stream, '') in a unique index to treat NULL streams as equal.
CREATE UNIQUE INDEX class_rooms_school_year_grade_stream_uniq
    ON class_rooms (school_id, academic_year_id, grade_level_id, NULLIF(stream, ''));

-- ============================================================================
-- Section 3: student_class_enrollments
-- ============================================================================

-- Historical mapping of a student to a class_room for a specific academic
-- term or year. Each enrollment is immutable once status reaches a terminal
-- state (GRADUATED), preserving attendance registers, assessments, and
-- report cards permanently.
CREATE TABLE student_class_enrollments (
    id               UUID        NOT NULL    DEFAULT gen_random_uuid()
                                        PRIMARY KEY,
    school_id        UUID        NOT NULL    REFERENCES schools(id)
                                        ON DELETE CASCADE,
    student_id       UUID        NOT NULL    REFERENCES students(student_id)
                                        ON DELETE CASCADE,
    class_room_id    UUID        NOT NULL    REFERENCES class_rooms(id)
                                        ON DELETE CASCADE,
    academic_year_id UUID        NOT NULL    REFERENCES academic_years(id)
                                        ON DELETE CASCADE,
    academic_term_id UUID        REFERENCES academic_terms(id)
                                        ON DELETE CASCADE,
    status           enrollment_status NOT NULL DEFAULT 'ACTIVE',
    enrolled_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    completed_at     TIMESTAMPTZ  DEFAULT NULL,
    metadata         JSONB       NOT NULL    DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: student_class_enrollments_school_id_idx supports school-scoped queries.
CREATE INDEX student_class_enrollments_school_id_idx ON student_class_enrollments (school_id);

-- Index: student_class_enrollments_student_id_idx supports student enrollment history lookups.
CREATE INDEX student_class_enrollments_student_id_idx ON student_class_enrollments (student_id);

-- Index: student_class_enrollments_class_room_id_idx supports class_room enrollment queries.
CREATE INDEX student_class_enrollments_class_room_id_idx ON student_class_enrollments (class_room_id);

-- Index: student_class_enrollments_academic_year_id_idx supports year-based enrollment queries.
CREATE INDEX student_class_enrollments_academic_year_id_idx ON student_class_enrollments (academic_year_id);

-- Index: student_class_enrollments_academic_term_id_idx supports term-based enrollment queries.
CREATE INDEX student_class_enrollments_academic_term_id_idx ON student_class_enrollments (academic_term_id);

-- Index: student_class_enrollments_status_idx supports status filtering.
CREATE INDEX student_class_enrollments_status_idx ON student_class_enrollments (status);

-- Constraint: student_class_enrollments_student_year_uniq ensures a student
-- cannot be enrolled in multiple class_rooms within the same academic year.
ALTER TABLE student_class_enrollments ADD CONSTRAINT student_class_enrollments_student_year_uniq
    UNIQUE (student_id, academic_year_id);

-- Constraint: student_class_enrollments_student_term_uniq ensures a student
-- cannot be enrolled in multiple class_rooms within the same academic term.
-- Only applies when academic_term_id is not NULL.
CREATE UNIQUE INDEX student_class_enrollments_student_term_uniq
    ON student_class_enrollments (student_id, academic_term_id)
    WHERE academic_term_id IS NOT NULL;



-- ============================================================================
-- Section 4: updated_at triggers
-- ============================================================================

CREATE TRIGGER class_rooms_updated_at_trg
    BEFORE UPDATE ON class_rooms
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER student_class_enrollments_updated_at_trg
    BEFORE UPDATE ON student_class_enrollments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- Section 5: Row-Level Security (RLS) on student_class_enrollments
-- ============================================================================

-- RLS is enabled but NOT forced, meaning the table owner (and any BYPASSRLS
-- roles used by migration tooling) bypasses the policy. Production application
-- queries run under a role that has RLS enforced.
ALTER TABLE student_class_enrollments ENABLE ROW LEVEL SECURITY;
ALTER TABLE student_class_enrollments FORCE ROW LEVEL SECURITY;

-- Policy: student_class_enrollments_tenant_isolation
--
-- Every row-level operation (SELECT, INSERT, UPDATE, DELETE) is gated by a
-- comparison ensuring the row's school_id belongs to a school operated by the
-- current tenant. The policy uses a subquery against schools to validate that
-- the school's tenant_id matches the session's app.current_tenant_id.
--
-- The pattern NULLIF(current_setting(...), '')::UUID ensures fail-closed
-- behavior: if the session variable is unset, empty, or invalid, the cast
-- raises an error and no rows are returned/inserted.
--
-- This mirrors the contract established by database.WithTenantTx, which sets
-- 'app.current_tenant_id' via SET LOCAL at the start of every application
-- transaction.

CREATE POLICY student_class_enrollments_tenant_isolation
    ON student_class_enrollments
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = student_class_enrollments.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = student_class_enrollments.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

-- ============================================================================
-- Section 6: Comments (documentation)
-- ============================================================================

COMMENT ON TYPE enrollment_status IS 'Student enrollment lifecycle status: ACTIVE (currently enrolled), PROMOTED (moved to next grade), REPEATING (retained in same grade), GRADUATED (completed final grade / alumni).';

COMMENT ON TABLE class_rooms IS 'Operational classroom container per academic year and stream (e.g., Class 1 Blue, Class 3 Yellow). Each academic year creates new class_room entries.';
COMMENT ON COLUMN class_rooms.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN class_rooms.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN class_rooms.academic_year_id IS 'FK to academic_years(id). Links the class_room to a specific academic year.';
COMMENT ON COLUMN class_rooms.grade_level_id IS 'FK to grade_levels(id). The grade level for this class_room (e.g., Grade 1, Grade 3).';
COMMENT ON COLUMN class_rooms.name IS 'Human-readable class_room name (e.g., "Class 1 Blue", "Grade 3 Yellow").';
COMMENT ON COLUMN class_rooms.stream IS 'Optional stream identifier within the grade (e.g., "Blue", "Yellow", "A", "B"). NULL if no streams.';
COMMENT ON COLUMN class_rooms.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN class_rooms.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE student_class_enrollments IS 'Historical mapping of a student to a class_room for a specific academic term or year. Preserves attendance, assessments, and report cards permanently.';
COMMENT ON COLUMN student_class_enrollments.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN student_class_enrollments.school_id IS 'FK to schools(id). Enforces multi-tenant isolation via RLS. Cascades on school delete.';
COMMENT ON COLUMN student_class_enrollments.student_id IS 'FK to students(student_id). The enrolled student.';
COMMENT ON COLUMN student_class_enrollments.class_room_id IS 'FK to class_rooms(id). The operational class_room for this enrollment.';
COMMENT ON COLUMN student_class_enrollments.academic_year_id IS 'FK to academic_years(id). The academic year of enrollment.';
COMMENT ON COLUMN student_class_enrollments.academic_term_id IS 'FK to academic_terms(id). Optional; NULL for year-level enrollments (e.g., final year without term splits).';
COMMENT ON COLUMN student_class_enrollments.status IS 'Enrollment status (enrollment_status enum). Controls promotion, repetition, and graduation workflows.';
COMMENT ON COLUMN student_class_enrollments.enrolled_at IS 'UTC timestamp when the student was enrolled in this class_room.';
COMMENT ON COLUMN student_class_enrollments.completed_at IS 'UTC timestamp when enrollment reached a terminal status (PROMOTED, REPEATING, GRADUATED). NULL for ACTIVE.';
COMMENT ON COLUMN student_class_enrollments.metadata IS 'JSONB for flexible enrollment metadata (e.g., previous school, transfer notes).';
COMMENT ON COLUMN student_class_enrollments.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN student_class_enrollments.updated_at IS 'UTC timestamp of last modification.';

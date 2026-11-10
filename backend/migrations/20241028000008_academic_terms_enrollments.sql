-- +goose Up
-- Academic years scoped to a School; Terms bounded with dates; Enrollments with status.

CREATE TYPE enrollment_status AS ENUM ('ACTIVE', 'TRANSFERRED', 'GRADUATED', 'WITHDRAWN');

-- academic_years: school-scoped cycle (3 terms)
CREATE TABLE academic_years (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    school_id       UUID        NOT NULL    REFERENCES schools(id) ON DELETE RESTRICT,
    year_label      VARCHAR(16) NOT NULL,     -- e.g., "2025/2026"
    is_current      BOOLEAN     NOT NULL    DEFAULT false,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (school_id, year_label)
);

CREATE INDEX academic_years_school_id_idx ON academic_years (school_id);
CREATE INDEX academic_years_is_current_idx ON academic_years (school_id, is_current) WHERE is_current = true;
CREATE INDEX academic_years_deleted_at_idx ON academic_years (deleted_at) WHERE deleted_at IS NULL;

-- terms: 1, 2, 3 per academic year
CREATE TABLE terms (
    id              UUID           NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    academic_year_id UUID          NOT NULL    REFERENCES academic_years(id) ON DELETE RESTRICT,
    sequence_index  INTEGER        NOT NULL,             -- 1, 2, 3
    label           VARCHAR(32)    NOT NULL,             -- "Term 1"
    start_date      DATE           NOT NULL,
    end_date        DATE           NOT NULL,
    created_at      TIMESTAMPTZ    NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (academic_year_id, sequence_index)
);

CREATE INDEX terms_academic_year_id_idx ON terms (academic_year_id);
CREATE UNIQUE INDEX terms_year_seq_uniq ON terms (academic_year_id, sequence_index);
CREATE INDEX terms_deleted_at_idx ON terms (deleted_at) WHERE deleted_at IS NULL;

-- enrollments: one active per student per term (DB enforced)
CREATE TABLE enrollments (
    id                UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    student_id        UUID        NOT NULL    REFERENCES students(student_id) ON DELETE RESTRICT,
    classroom_id      UUID        NOT NULL    REFERENCES classrooms(id) ON DELETE RESTRICT,
    term_id           UUID        NOT NULL    REFERENCES terms(id) ON DELETE RESTRICT,
    status            enrollment_status NOT NULL DEFAULT 'ACTIVE',
    created_at        TIMESTAMPTZ NOT NULL    DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL    DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX enrollments_student_id_idx ON enrollments (student_id);
CREATE INDEX enrollments_classroom_id_idx ON enrollments (classroom_id);
CREATE INDEX enrollments_term_id_idx ON enrollments (term_id);
CREATE INDEX enrollments_active_student_term_idx ON enrollments (student_id, term_id) WHERE status = 'ACTIVE';

-- DB enforcement: exactly one ACTIVE enrollment per (student, term)
CREATE UNIQUE INDEX enrollments_active_student_term_uniq
    ON enrollments (student_id, term_id) WHERE status = 'ACTIVE';
CREATE INDEX enrollments_deleted_at_idx ON enrollments (deleted_at) WHERE deleted_at IS NULL;

-- updated_at triggers (applied per-table)
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON enrollments FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON terms FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_set_timestamp ON enrollments;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON terms;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON academic_years;
DROP INDEX IF EXISTS enrollments_active_student_term_uniq;
DROP INDEX IF EXISTS enrollments_active_student_term_idx;
DROP INDEX IF EXISTS enrollments_term_id_idx;
DROP INDEX IF EXISTS enrollments_classroom_id_idx;
DROP INDEX IF EXISTS enrollments_student_id_idx;
DROP TABLE IF EXISTS enrollments;
DROP INDEX IF EXISTS terms_year_seq_uniq;
DROP INDEX IF EXISTS terms_academic_year_id_idx;
DROP TABLE IF EXISTS terms;
DROP INDEX IF EXISTS academic_years_is_current_idx;
DROP INDEX IF EXISTS academic_years_school_id_idx;
DROP TABLE IF EXISTS academic_years;
DROP TYPE IF EXISTS enrollment_status;
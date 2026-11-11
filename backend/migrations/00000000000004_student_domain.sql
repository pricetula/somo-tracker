-- +goose Up
-- Student records scoped to a school. admission_number is unique within a school.
CREATE TABLE students (
    student_id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    admission_number VARCHAR(255) NOT NULL,
    national_number VARCHAR(255),
    full_name VARCHAR(255) NOT NULL,
    date_of_birth DATE NOT NULL,
    gender VARCHAR(32) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT students_school_admission_uniq UNIQUE (school_id, admission_number)
);

CREATE INDEX students_school_id_idx ON students (school_id);
CREATE INDEX students_admission_number_idx ON students (admission_number);
CREATE INDEX students_full_name_idx ON students (full_name);
CREATE INDEX idx_students_deleted_at ON students (deleted_at) WHERE deleted_at IS NULL;

-- Links a guardian (via their membership) to a student.
CREATE TABLE guardian_student_links (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    guardian_id UUID NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    relationship_type VARCHAR(64) NOT NULL,
    is_emergency_contact BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT guardian_student_links_membership_student_uniq UNIQUE (guardian_id, student_id)
);

CREATE INDEX guardian_student_links_membership_id_idx ON guardian_student_links (guardian_id);
CREATE INDEX guardian_student_links_student_id_idx ON guardian_student_links (student_id);
CREATE UNIQUE INDEX guardian_student_links_emergency_uniq_idx ON guardian_student_links (student_id) WHERE is_emergency_contact = true;
CREATE INDEX idx_guardian_student_links_deleted_at ON guardian_student_links (deleted_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_guardian_student_links_deleted_at;
DROP INDEX IF EXISTS guardian_student_links_emergency_uniq_idx;
DROP INDEX IF EXISTS guardian_student_links_student_id_idx;
DROP INDEX IF EXISTS guardian_student_links_membership_id_idx;
DROP TABLE IF EXISTS guardian_student_links;
DROP INDEX IF EXISTS idx_students_deleted_at;
DROP INDEX IF EXISTS students_full_name_idx;
DROP INDEX IF EXISTS students_admission_number_idx;
DROP INDEX IF EXISTS students_school_id_idx;
DROP TABLE IF EXISTS students;

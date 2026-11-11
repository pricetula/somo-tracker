-- +goose Up
-- Subject / Topic / Subtopic hierarchy (replaces LearningArea/Strand/SubStrand).
CREATE TABLE subjects (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE RESTRICT,
    class_level_id UUID NOT NULL REFERENCES class_levels(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(64),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT subjects_education_grade_uniq UNIQUE (education_system_id, class_level_id, name)
);

CREATE INDEX subjects_education_id_idx ON subjects (education_system_id);
CREATE INDEX subjects_class_level_id_idx ON subjects (class_level_id);
CREATE INDEX subjects_name_idx ON subjects (name);
CREATE INDEX subjects_deleted_at_idx ON subjects (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE topics (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    sequence_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT topics_subject_sequence_uniq UNIQUE (subject_id, sequence_index)
);

CREATE INDEX topics_subject_id_idx ON topics (subject_id);
CREATE INDEX topics_name_idx ON topics (name);
CREATE INDEX topics_deleted_at_idx ON topics (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE subtopics (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    sequence_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT subtopics_topic_sequence_uniq UNIQUE (topic_id, sequence_index)
);

CREATE INDEX subtopics_topic_id_idx ON subtopics (topic_id);
CREATE INDEX subtopics_name_idx ON subtopics (name);
CREATE INDEX subtopics_deleted_at_idx ON subtopics (deleted_at) WHERE deleted_at IS NULL;

-- Stream: color/group name scoped to a School.
CREATE TABLE streams (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    color VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT streams_school_name_uniq UNIQUE (school_id, name)
);

CREATE INDEX streams_school_id_idx ON streams (school_id);
CREATE INDEX streams_school_name_idx ON streams (school_id, name);
CREATE INDEX streams_name_idx ON streams (name);
CREATE INDEX streams_deleted_at_idx ON streams (deleted_at) WHERE deleted_at IS NULL;

-- Classroom: logical grouping = School + ClassLevel + Stream.
CREATE TABLE classrooms (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    class_level_id UUID NOT NULL REFERENCES class_levels(id) ON DELETE RESTRICT,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE RESTRICT,
    name VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT classrooms_school_level_stream_uniq UNIQUE (school_id, class_level_id, stream_id)
);

CREATE INDEX classrooms_school_id_idx ON classrooms (school_id);
CREATE INDEX classrooms_stream_id_idx ON classrooms (stream_id);
CREATE INDEX idx_classrooms_deleted_at ON classrooms (deleted_at) WHERE deleted_at IS NULL;

-- Academic years scoped to a School.
CREATE TYPE enrollment_status AS ENUM ('ACTIVE', 'TRANSFERRED', 'GRADUATED', 'WITHDRAWN');

CREATE TABLE academic_years (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    year_label VARCHAR(16) NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (school_id, year_label)
);

CREATE INDEX academic_years_school_id_idx ON academic_years (school_id);
CREATE INDEX academic_years_is_current_idx ON academic_years (school_id, is_current) WHERE is_current = true;
CREATE INDEX academic_years_deleted_at_idx ON academic_years (deleted_at) WHERE deleted_at IS NULL;

-- Terms: 1, 2, 3 per academic year.
CREATE TABLE terms (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    sequence_index INTEGER NOT NULL,
    label VARCHAR(32) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (academic_year_id, sequence_index)
);

CREATE INDEX terms_academic_year_id_idx ON terms (academic_year_id);
CREATE UNIQUE INDEX terms_year_seq_uniq ON terms (academic_year_id, sequence_index);
CREATE INDEX terms_deleted_at_idx ON terms (deleted_at) WHERE deleted_at IS NULL;

-- Enrollments: one active per student per term (DB enforced).
CREATE TABLE enrollments (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    student_id UUID NOT NULL REFERENCES students(student_id) ON DELETE RESTRICT,
    classroom_id UUID NOT NULL REFERENCES classrooms(id) ON DELETE RESTRICT,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE RESTRICT,
    status enrollment_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX enrollments_student_id_idx ON enrollments (student_id);
CREATE INDEX enrollments_classroom_id_idx ON enrollments (classroom_id);
CREATE INDEX enrollments_term_id_idx ON enrollments (term_id);
CREATE INDEX enrollments_active_student_term_idx ON enrollments (student_id, term_id) WHERE status = 'ACTIVE';
CREATE UNIQUE INDEX enrollments_active_student_term_uniq ON enrollments (student_id, term_id) WHERE status = 'ACTIVE';
CREATE INDEX enrollments_deleted_at_idx ON enrollments (deleted_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS enrollments_deleted_at_idx;
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
DROP INDEX IF EXISTS idx_classrooms_deleted_at;
DROP INDEX IF EXISTS classrooms_stream_id_idx;
DROP INDEX IF EXISTS classrooms_school_id_idx;
DROP TABLE IF EXISTS classrooms;
DROP INDEX IF EXISTS streams_deleted_at_idx;
DROP INDEX IF EXISTS streams_name_idx;
DROP INDEX IF EXISTS streams_school_name_idx;
DROP INDEX IF EXISTS streams_school_id_idx;
DROP TABLE IF EXISTS streams;
DROP INDEX IF EXISTS subtopics_deleted_at_idx;
DROP INDEX IF EXISTS subtopics_topic_id_idx;
DROP INDEX IF EXISTS subtopics_name_idx;
DROP TABLE IF EXISTS subtopics;
DROP INDEX IF EXISTS topics_deleted_at_idx;
DROP INDEX IF EXISTS topics_subject_id_idx;
DROP INDEX IF EXISTS topics_name_idx;
DROP TABLE IF EXISTS topics;
DROP INDEX IF EXISTS subjects_deleted_at_idx;
DROP INDEX IF EXISTS subjects_name_idx;
DROP INDEX IF EXISTS subjects_class_level_id_idx;
DROP INDEX IF EXISTS subjects_education_id_idx;
DROP TABLE IF EXISTS subjects;

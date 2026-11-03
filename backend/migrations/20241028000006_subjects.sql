-- +goose Up
-- Replace LearningArea/Strand/SubStrand with Subject/Topic/Subtopic.
-- Subject is global reference tied to education_system and GradeLevel (not country).

CREATE TABLE subjects (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    education_system_id UUID    NOT NULL    REFERENCES education_systems(id) ON DELETE RESTRICT,
    grade_level_id  UUID        NOT NULL    REFERENCES grade_levels(id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    code            VARCHAR(64),
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT subjects_education_grade_uniq UNIQUE (education_system_id, grade_level_id, name)
);

CREATE INDEX subjects_education_id_idx ON subjects (education_system_id);
CREATE INDEX subjects_grade_level_id_idx ON subjects (grade_level_id);
CREATE INDEX subjects_name_idx ON subjects (name);
CREATE INDEX subjects_deleted_at_idx ON subjects (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE topics (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    subject_id      UUID        NOT NULL    REFERENCES subjects(id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    sequence_index  INTEGER     NOT NULL    DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT topics_subject_sequence_uniq UNIQUE (subject_id, sequence_index)
);

CREATE INDEX topics_subject_id_idx ON topics (subject_id);
CREATE INDEX topics_name_idx ON topics (name);
CREATE INDEX topics_deleted_at_idx ON topics (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE subtopics (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    topic_id        UUID        NOT NULL    REFERENCES topics(id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    sequence_index  INTEGER     NOT NULL    DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT subtopics_topic_sequence_uniq UNIQUE (topic_id, sequence_index)
);

CREATE INDEX subtopics_topic_id_idx ON subtopics (topic_id);
CREATE INDEX subtopics_name_idx ON subtopics (name);
CREATE INDEX subtopics_deleted_at_idx ON subtopics (deleted_at) WHERE deleted_at IS NULL;

-- Updated assessment references: Subject + Topic + Subtopic (replaces LearningArea/Strand/SubStrand)
-- Assessment table itself not yet created; this document documents the intended target.

-- +goose Down
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
DROP INDEX IF EXISTS subjects_grade_level_id_idx;
DROP INDEX IF EXISTS subjects_education_id_idx;
DROP TABLE IF EXISTS subjects;

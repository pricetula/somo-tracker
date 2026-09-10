-- Migration: 000005_create_subject_hierarchy
-- Purpose: Create the subject–topic–sub-topic curriculum hierarchy:
--   subjects (scoped to education_system), topics, sub_topics.
--
-- Dependencies:
--   - 000001_init_extensions (pgcrypto for gen_random_uuid()).
--   - 000004_create_sis_schema (provides education_systems table).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: subjects
-- ============================================================================

-- Subjects offered within a specific education system.
-- Example: (CBE, Kenya) -> Mathematics, English, Kiswahili, Science, Social Studies.
CREATE TABLE subjects (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    education_system_id UUID        NOT NULL    REFERENCES education_systems(id)
                                       ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    code                VARCHAR(16)  NOT NULL,
    type                VARCHAR(32)  NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    -- A subject code must be unique within an education system.
    CONSTRAINT subjects_system_code_uniq UNIQUE (education_system_id, code)
);

-- Index: subjects_education_system_id_idx supports system-scoped subject listings.
CREATE INDEX subjects_education_system_id_idx ON subjects (education_system_id);
-- Index: subjects_code_idx supports code-based lookups.
CREATE INDEX subjects_code_idx ON subjects (code);

-- ============================================================================
-- Section 2: topics
-- ============================================================================

-- Topics within a subject, ordered chronologically by sequence_index.
-- Example: (Mathematics) -> Fractions and Decimals, Algebra, Geometry.
CREATE TABLE topics (
    id              UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    subject_id      UUID        NOT NULL    REFERENCES subjects(id)
                                       ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    sequence_index  INTEGER      NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    -- A topic's sequence position must be unique within a subject
    -- to guarantee deterministic chronological ordering.
    CONSTRAINT topics_subject_seq_uniq UNIQUE (subject_id, sequence_index)
);

-- Index: topics_subject_id_idx supports subject-scoped topic listings.
CREATE INDEX topics_subject_id_idx ON topics (subject_id);
-- Composite index: topics_subject_seq_idx covers the primary
-- ordered listing query (subject ordered by sequence_index).
CREATE INDEX topics_subject_seq_idx ON topics (subject_id, sequence_index);

-- ============================================================================
-- Section 3: sub_topics
-- ============================================================================

-- Sub-topics within a topic, ordered by sequence_index.
-- Example: (Fractions and Decimals) -> Addition of Fractions, Subtraction of Fractions, Multiplication of Fractions.
CREATE TABLE sub_topics (
    id              UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    topic_id        UUID        NOT NULL    REFERENCES topics(id)
                                       ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    sequence_index  INTEGER      NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    -- A sub-topic's sequence position must be unique within a topic
    -- to guarantee deterministic chronological ordering.
    CONSTRAINT sub_topics_topic_seq_uniq UNIQUE (topic_id, sequence_index)
);

-- Index: sub_topics_topic_id_idx supports topic-scoped sub-topic listings.
CREATE INDEX sub_topics_topic_id_idx ON sub_topics (topic_id);
-- Composite index: sub_topics_topic_seq_idx covers the primary
-- ordered listing query (topic ordered by sequence_index).
CREATE INDEX sub_topics_topic_seq_idx ON sub_topics (topic_id, sequence_index);

-- ============================================================================
-- Section 4: updated_at triggers
-- ============================================================================

-- Reuse the set_updated_at() helper from migration 000004.
CREATE TRIGGER subjects_updated_at_trg
    BEFORE UPDATE ON subjects
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER topics_updated_at_trg
    BEFORE UPDATE ON topics
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER sub_topics_updated_at_trg
    BEFORE UPDATE ON sub_topics
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- Section 5: Comments (documentation)
-- ============================================================================

COMMENT ON TABLE subjects IS 'Subjects offered within an education system (e.g. Mathematics, English). Scoped to education_system_id.';
COMMENT ON COLUMN subjects.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN subjects.education_system_id IS 'FK to education_systems(id). Cascades on education system delete.';
COMMENT ON COLUMN subjects.name IS 'Human-readable subject name (e.g. Mathematics).';
COMMENT ON COLUMN subjects.code IS 'Short subject code (e.g. MAT, ENG). Unique within an education system.';
COMMENT ON COLUMN subjects.type IS 'Subject type: Core, Optional, Elective, etc.';
COMMENT ON COLUMN subjects.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN subjects.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE topics IS 'Topics within a subject, ordered by sequence_index for curriculum sequencing.';
COMMENT ON COLUMN topics.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN topics.subject_id IS 'FK to subjects(id). Cascades on subject delete.';
COMMENT ON COLUMN topics.name IS 'Topic name (e.g. Fractions and Decimals).';
COMMENT ON COLUMN topics.sequence_index IS 'Integer for chronological sorting; unique within a subject.';
COMMENT ON COLUMN topics.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN topics.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE sub_topics IS 'Sub-topics within a topic, ordered by sequence_index for granular curriculum sequencing.';
COMMENT ON COLUMN sub_topics.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN sub_topics.topic_id IS 'FK to topics(id). Cascades on topic delete.';
COMMENT ON COLUMN sub_topics.name IS 'Sub-topic name (e.g. Addition of Fractions).';
COMMENT ON COLUMN sub_topics.sequence_index IS 'Integer for chronological sorting; unique within a topic.';
COMMENT ON COLUMN sub_topics.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN sub_topics.updated_at IS 'UTC timestamp of last modification.';
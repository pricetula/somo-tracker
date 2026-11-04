-- +goose Up
-- Stream: pure color/group name scoped to a School. e.g., "Grade 1 Blue".
-- Tied to School only (no GradeLevel).

CREATE TABLE streams (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    school_id       UUID        NOT NULL    REFERENCES schools(id) ON DELETE RESTRICT,
    color           VARCHAR(64) NOT NULL,
    name            VARCHAR(64) NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT streams_school_name_uniq UNIQUE (school_id, name)
);

CREATE INDEX streams_school_id_idx ON streams (school_id);
CREATE INDEX streams_school_name_idx ON streams (school_id, name);
CREATE INDEX streams_name_idx ON streams (name);
CREATE INDEX streams_deleted_at_idx ON streams (deleted_at) WHERE deleted_at IS NULL;

-- 3) updated_at trigger (applied per-table since migration 5 runs before streams exists)
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON streams FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- 5) Row Level Security (tenant isolation, model FR-FND-001).
ALTER TABLE streams ENABLE ROW LEVEL SECURITY;

CREATE POLICY streams_isolate ON streams
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = streams.school_id));

-- 14) Classrooms: logical classroom grouping = School + ClassLevel + Stream.
CREATE TABLE classrooms (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    school_id       UUID        NOT NULL    REFERENCES schools(id) ON DELETE RESTRICT,
    class_level_id  UUID        NOT NULL    REFERENCES class_levels(id) ON DELETE RESTRICT,
    stream_id       UUID        NOT NULL    REFERENCES streams(id) ON DELETE RESTRICT,
    name            VARCHAR(64),
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT classrooms_school_level_stream_uniq UNIQUE (school_id, class_level_id, stream_id)
);

CREATE INDEX classrooms_school_id_idx ON classrooms (school_id);
CREATE INDEX classrooms_stream_id_idx ON classrooms (stream_id);

ALTER TABLE classrooms ENABLE ROW LEVEL SECURITY;

CREATE POLICY classrooms_isolate ON classrooms
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = classrooms.school_id));

CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON classrooms FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- 6) updated_at trigger (reused; applied per-table since migration 5 runs before streams)

-- 15) No JSON schema validation; defaults only.

-- +goose Down
DROP POLICY IF EXISTS classrooms_isolate ON classrooms;
ALTER TABLE classrooms DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS classrooms_stream_id_idx;
DROP INDEX IF EXISTS classrooms_school_id_idx;
DROP TABLE IF EXISTS classrooms;
DROP POLICY IF EXISTS streams_isolate ON streams;
ALTER TABLE streams DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS streams_deleted_at_idx;
DROP INDEX IF EXISTS streams_name_idx;
DROP INDEX IF EXISTS streams_school_name_idx;
DROP INDEX IF EXISTS streams_school_id_idx;
DROP TABLE IF EXISTS streams;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON streams;

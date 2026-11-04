-- +goose Up
-- Stream: subdivision of GradeLevel within a School, used to group students of the same grade.
-- Tied to School + GradeLevel (not country).

CREATE TABLE streams (
    id              UUID        NOT NULL    DEFAULT uuidv7() PRIMARY KEY,
    school_id       UUID        NOT NULL    REFERENCES schools(id) ON DELETE RESTRICT,
    grade_level_id  UUID        NOT NULL    REFERENCES grade_levels(id) ON DELETE RESTRICT,
    name            VARCHAR(64) NOT NULL,
    code            VARCHAR(32),
    description     TEXT,
    sequence_index  INTEGER     NOT NULL    DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT streams_school_grade_name_uniq UNIQUE (school_id, grade_level_id, name)
);

CREATE INDEX streams_school_id_idx ON streams (school_id);
CREATE INDEX streams_grade_level_id_idx ON streams (grade_level_id);
CREATE INDEX streams_school_grade_idx ON streams (school_id, grade_level_id);
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

-- 13) No JSON schema validation; defaults only.

-- +goose Down
DROP POLICY IF EXISTS streams_isolate ON streams;
ALTER TABLE streams DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS streams_deleted_at_idx;
DROP INDEX IF EXISTS streams_name_idx;
DROP INDEX IF EXISTS streams_school_grade_idx;
DROP INDEX IF EXISTS streams_grade_level_id_idx;
DROP INDEX IF EXISTS streams_school_id_idx;
DROP TABLE IF EXISTS streams;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON streams;

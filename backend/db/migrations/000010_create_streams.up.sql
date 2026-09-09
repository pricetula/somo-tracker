-- Migration: 000010_create_streams
CREATE TABLE streams (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    school_id   UUID        NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name        VARCHAR(64) NOT NULL,
    color       VARCHAR(32) DEFAULT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX streams_school_id_idx ON streams (school_id);
CREATE UNIQUE INDEX streams_school_name_uniq ON streams (school_id, name);

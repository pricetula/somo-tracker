-- Migration: 000003_cbc_streams_and_classes
-- SomoTracker — CBC stream management + class schema refactor
--
-- 1. Creates cbc_streams table (domesticated streams with no cascade delete)
-- 2. Refactors cbc_classes: drops name/stream, adds stream_id FK with RESTRICT.
--    Existing unique constraint on (tenant_id, id) is preserved; the new
--    uniqueness contract moves to uq_cbc_classes_tier_stream on
--    (school_id, academic_year_id, grade_level, stream_id).

BEGIN;

-- ============================================================================
-- 1. CBC STREAMS
-- ============================================================================

CREATE TABLE IF NOT EXISTS cbc_streams (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL,
    school_id  UUID         NOT NULL,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cbc_streams_school
        FOREIGN KEY (school_id) REFERENCES cbc_schools(id) ON DELETE NO ACTION,

    CONSTRAINT uq_cbc_streams_tenant_school_name
        UNIQUE (tenant_id, school_id, name)
);

CREATE INDEX IF NOT EXISTS idx_cbc_streams_school_id
    ON cbc_streams (school_id);

CREATE INDEX IF NOT EXISTS idx_cbc_streams_tenant_id
    ON cbc_streams (tenant_id);

COMMENT ON TABLE cbc_streams IS
    'Named streams within a school (e.g. "Blue", "Red", "Green"). A stream is
     the second axis of class identity alongside grade_level. Streams themselves
     cannot be deleted while any cbc_classes row references them (ON DELETE
     RESTRICT via fk_cbc_classes_stream). Streams with no class references may
     be deleted via the API.';

COMMENT ON CONSTRAINT fk_cbc_streams_school ON cbc_streams IS
    'school_id alone carries the tenancy relationship via cbc_schools. tenant_id
     is stored denormalised for fast multi-tenant filtering without joins.
     ON DELETE NO ACTION — streams are never cascade-deleted. Deletion of a
     school must be handled explicitly upstream before streams can be removed.';

-- ============================================================================
-- 2. REFACTOR CBC_CLASSES
-- ============================================================================

-- Drop old columns that are being replaced by the stream_id relation.
ALTER TABLE cbc_classes
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS stream;

-- Add stream_id FK with ON DELETE RESTRICT.
-- The stream_id column must be NOT NULL so every class has exactly one stream.
ALTER TABLE cbc_classes
    ADD COLUMN stream_id UUID NOT NULL
        REFERENCES cbc_streams(id) ON DELETE RESTRICT;

-- Drop the old uniqueness constraint on (tenant_id, id) since the new schema
-- replaces it with a business-level uniqueness constraint.
ALTER TABLE cbc_classes
    DROP CONSTRAINT IF EXISTS uq_cbc_classes_tenant;

-- New uniqueness constraint: one class per (school, academic year, grade, stream).
ALTER TABLE cbc_classes
    ADD CONSTRAINT uq_cbc_classes_tier_stream
        UNIQUE (school_id, academic_year_id, grade_level, stream_id);

-- New covering index for the most common listing query pattern:
-- listing classes filtered by school, year, grade, and/or stream.
CREATE INDEX IF NOT EXISTS idx_cbc_classes_school_year_grade_stream
    ON cbc_classes (school_id, academic_year_id, grade_level, stream_id);

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

COMMIT;

-- Migration: 000003_cbc_streams_and_classes (rollback)
-- SomoTracker — Reverts stream and class schema changes.
--
-- 1. Restores dropped columns (name, stream) on cbc_classes
-- 2. Drops stream_id column and its FK constraint
-- 3. Restores original uniqueness constraint on (tenant_id, id)
-- 4. Drops cbc_streams table

BEGIN;

-- ============================================================================
-- 1. RESTORE CBC_CLASSES
-- ============================================================================

-- Drop the new constraint and index
ALTER TABLE cbc_classes
    DROP CONSTRAINT IF EXISTS uq_cbc_classes_tier_stream;

DROP INDEX IF EXISTS idx_cbc_classes_school_year_grade_stream;

-- Drop stream_id column (cascades to drop the FK constraint)
ALTER TABLE cbc_classes
    DROP COLUMN IF EXISTS stream_id;

-- Restore the old unique constraint
ALTER TABLE cbc_classes
    ADD CONSTRAINT uq_cbc_classes_tenant
        UNIQUE (tenant_id, id);

-- Restore old columns. These are nullable during the down migration because
-- there are no values to restore — the data model is being reverted and
-- data loss is expected in the rollback path.
ALTER TABLE cbc_classes
    ADD COLUMN name   VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN stream VARCHAR(100) NOT NULL DEFAULT '';

-- ============================================================================
-- 2. DROP CBC_STREAMS
-- ============================================================================

DROP INDEX IF EXISTS idx_cbc_streams_school_id;
DROP INDEX IF EXISTS idx_cbc_streams_tenant_id;

DROP TABLE IF EXISTS cbc_streams CASCADE;

-- ============================================================================
-- END OF ROLLBACK
-- ============================================================================

COMMIT;

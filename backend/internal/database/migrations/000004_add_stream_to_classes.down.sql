-- Migration: 000004_add_stream_to_classes (rollback)
-- ============================================================================

DROP INDEX IF EXISTS idx_classes_stream;

ALTER TABLE classes
    DROP COLUMN IF EXISTS stream;

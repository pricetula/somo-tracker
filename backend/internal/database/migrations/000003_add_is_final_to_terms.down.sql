-- Migration: 000003_add_is_final_to_terms (rollback)
-- ============================================================================

DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'academic_terms' AND column_name = 'is_final'
    ) THEN
        ALTER TABLE academic_terms DROP COLUMN is_final;
    END IF;
END $$;

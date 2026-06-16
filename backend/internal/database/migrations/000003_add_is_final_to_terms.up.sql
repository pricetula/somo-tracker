-- Migration: 000003_add_is_final_to_terms
-- ============================================================================
-- Adds is_final column to academic_terms so schools can mark the final
-- period (e.g. Term 3 / Exam term) of their academic year.
-- ============================================================================

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'academic_terms' AND column_name = 'is_final'
    ) THEN
        ALTER TABLE academic_terms
            ADD COLUMN is_final BOOLEAN NOT NULL DEFAULT false;
    END IF;
END $$;

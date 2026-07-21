-- Migration: 000009_create_student_subject_strand_summaries — DOWN
-- Reverts: table, function, trigger extension, RLS

-- Restore the original after-publish trigger (without strand summary call)
CREATE OR REPLACE FUNCTION fn_assessment_sessions_after_publish()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'PUBLISHED' AND (OLD.status IS DISTINCT FROM 'PUBLISHED') THEN
        PERFORM fn_refresh_term_subject_summary_for_session(NEW.id);
    END IF;
    RETURN NEW;
END;
$$;

-- Drop the strand summary function
DROP FUNCTION IF EXISTS fn_refresh_subject_strand_summary_for_session(UUID);

-- Drop RLS policy
DROP POLICY IF EXISTS tenant_isolation_policy ON student_subject_strand_summaries;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_student_subject_strand_summaries_updated_at
    ON student_subject_strand_summaries;

-- Drop indexes
DROP INDEX IF EXISTS idx_strand_summaries_term_sub_strand;
DROP INDEX IF EXISTS idx_strand_summaries_student_term;
DROP INDEX IF EXISTS idx_strand_summaries_school;
DROP INDEX IF EXISTS idx_strand_summaries_tenant;

-- Drop table
DROP TABLE IF EXISTS student_subject_strand_summaries;

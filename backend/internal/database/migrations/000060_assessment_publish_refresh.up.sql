-- Migration: 000060_assessment_publish_refresh
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: AFTER-PUBLISH refresh trigger for assessment_sessions

-- ============================================================================
-- AFTER-PUBLISH REFRESH TRIGGER (assessment_sessions)
-- Fires AFTER UPDATE on assessment_sessions when status changes to PUBLISHED.
-- Refreshes both the term-level subject summaries (000006) and the
-- sub-strand-level rubric summaries (000009).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_assessment_sessions_after_publish()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'PUBLISHED' AND (OLD.status IS DISTINCT FROM 'PUBLISHED') THEN
        -- Refresh term-level subject summaries (existing)
        PERFORM fn_refresh_term_subject_summary_for_session(NEW.id);

        -- Refresh sub-strand-level summaries (new, rubric-only)
        PERFORM fn_refresh_subject_strand_summary_for_session(NEW.id);
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_assessment_sessions_refresh_summary
    ON assessment_sessions;
CREATE TRIGGER trg_assessment_sessions_refresh_summary
    AFTER UPDATE OF status ON assessment_sessions
    FOR EACH ROW
    EXECUTE FUNCTION fn_assessment_sessions_after_publish();

COMMENT ON TRIGGER trg_assessment_sessions_refresh_summary ON assessment_sessions IS
    'After an assessment session is published, refresh the term-level subject
     summaries and the rubric sub-strand summaries for all students in that
     session.';

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

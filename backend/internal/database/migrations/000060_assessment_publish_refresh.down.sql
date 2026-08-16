-- Migration: 000060_assessment_publish_refresh (rollback)
-- SomoTracker — rollback for 000060_assessment_publish_refresh.

DROP TRIGGER IF EXISTS trg_assessment_sessions_refresh_summary ON assessment_sessions;
DROP FUNCTION IF EXISTS fn_assessment_sessions_after_publish CASCADE;

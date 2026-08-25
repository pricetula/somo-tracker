-- Migration: 000045_assessment_sessions (rollback)
-- SomoTracker — rollback for 000045_assessment_sessions.

DROP FUNCTION IF EXISTS fn_block_assessment_max_points_update CASCADE;

DROP TABLE IF EXISTS assessment_sessions CASCADE;

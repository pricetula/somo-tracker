-- Migration: 000054_student_behavior_term_summaries (rollback)
-- SomoTracker — rollback for 000054_student_behavior_term_summaries.

DROP FUNCTION IF EXISTS fn_refresh_student_behavior_term_summary CASCADE;
DROP FUNCTION IF EXISTS fn_refresh_student_behavior_term_summary_for_note CASCADE;

DROP TABLE IF EXISTS student_behavior_term_summaries CASCADE;

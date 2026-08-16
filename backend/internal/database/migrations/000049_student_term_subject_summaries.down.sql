-- Migration: 000049_student_term_subject_summaries (rollback)
-- SomoTracker — rollback for 000049_student_term_subject_summaries.

DROP FUNCTION IF EXISTS fn_refresh_term_subject_summary_for_session CASCADE;

DROP TABLE IF EXISTS student_term_subject_summaries CASCADE;

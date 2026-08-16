-- Migration: 000052_student_subject_strand_summaries (rollback)
-- SomoTracker — rollback for 000052_student_subject_strand_summaries.

DROP FUNCTION IF EXISTS fn_refresh_subject_strand_summary_for_session CASCADE;

DROP TABLE IF EXISTS student_subject_strand_summaries CASCADE;

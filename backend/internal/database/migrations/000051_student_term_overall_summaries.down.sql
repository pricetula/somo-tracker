-- Migration: 000050_student_term_overall_summaries (rollback)
-- SomoTracker — rollback for 000050_student_term_overall_summaries.

DROP FUNCTION IF EXISTS fn_compute_term_overall_summaries_for_term CASCADE;
DROP FUNCTION IF EXISTS fn_compute_single_student_term_overall_summary CASCADE;

DROP TABLE IF EXISTS student_term_overall_summaries CASCADE;

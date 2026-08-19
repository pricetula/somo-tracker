-- Migration: 000051_student_cohort_position_summaries (rollback)
-- SomoTracker — rollback for 000051_student_cohort_position_summaries.

DROP FUNCTION IF EXISTS fn_compute_cohort_positions_for_term CASCADE;

DROP TABLE IF EXISTS student_cohort_position_summaries CASCADE;

-- Migration: 000055_teacher_subject_performance_summaries (rollback)
-- SomoTracker — rollback for 000055_teacher_subject_performance_summaries.

DROP FUNCTION IF EXISTS fn_compute_teacher_subject_performance_summaries CASCADE;

DROP TABLE IF EXISTS teacher_subject_performance_summaries CASCADE;

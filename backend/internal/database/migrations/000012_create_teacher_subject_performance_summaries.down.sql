-- Migration: 000012_create_teacher_subject_performance_summaries (down)

DROP FUNCTION IF EXISTS fn_compute_teacher_subject_performance_summaries(UUID);
DROP TABLE IF EXISTS teacher_subject_performance_summaries;

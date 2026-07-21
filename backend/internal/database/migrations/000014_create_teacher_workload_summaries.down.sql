-- Migration: 000014_create_teacher_workload_summaries (down)

DROP FUNCTION IF EXISTS fn_compute_teacher_workload_summaries(UUID);
DROP TABLE IF EXISTS teacher_workload_summaries;

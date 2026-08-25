-- Migration: 000057_teacher_workload_summaries (rollback)
-- SomoTracker — rollback for 000057_teacher_workload_summaries.

DROP FUNCTION IF EXISTS fn_compute_teacher_workload_summaries CASCADE;

DROP TABLE IF EXISTS teacher_workload_summaries CASCADE;

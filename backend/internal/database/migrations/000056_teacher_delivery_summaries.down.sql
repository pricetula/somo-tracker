-- Migration: 000056_teacher_delivery_summaries (rollback)
-- SomoTracker — rollback for 000056_teacher_delivery_summaries.

DROP FUNCTION IF EXISTS fn_compute_teacher_delivery_summaries CASCADE;

DROP TABLE IF EXISTS teacher_delivery_summaries CASCADE;

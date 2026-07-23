-- Migration: 000013_create_teacher_delivery_summaries (down)

DROP FUNCTION IF EXISTS fn_compute_teacher_delivery_summaries(UUID);
DROP TABLE IF EXISTS teacher_delivery_summaries;

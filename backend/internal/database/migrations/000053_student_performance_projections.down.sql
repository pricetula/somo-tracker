-- Migration: 000053_student_performance_projections (rollback)
-- SomoTracker — rollback for 000053_student_performance_projections.

DROP FUNCTION IF EXISTS fn_compute_performance_projections_for_term CASCADE;

DROP TABLE IF EXISTS student_performance_projections CASCADE;

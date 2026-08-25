-- Migration: 000046_student_assessment_scores (rollback)
-- SomoTracker — rollback for 000046_student_assessment_scores.

DROP FUNCTION IF EXISTS max_points_check CASCADE;

DROP TABLE IF EXISTS student_assessment_scores CASCADE;

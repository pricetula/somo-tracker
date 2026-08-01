-- Migration: 000016_create_class_attendance_rollups (down)
-- Reverses the changes from the up migration.

DROP TRIGGER IF EXISTS trg_class_term_attendance_summaries_updated_at
    ON class_term_attendance_summaries;
DROP TABLE IF EXISTS class_term_attendance_summaries;

DROP TRIGGER IF EXISTS trg_class_learning_area_term_summaries_updated_at
    ON class_learning_area_term_summaries;
DROP TABLE IF EXISTS class_learning_area_term_summaries;

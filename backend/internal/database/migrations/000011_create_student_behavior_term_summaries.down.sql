-- Migration: 000011_create_student_behavior_term_summaries (down)

DROP TRIGGER IF EXISTS trg_behavior_notes_refresh_term_summary ON behavior_notes;
DROP FUNCTION IF EXISTS fn_refresh_student_behavior_term_summary_for_note();
DROP FUNCTION IF EXISTS fn_refresh_student_behavior_term_summary(UUID, UUID);
DROP TABLE IF EXISTS student_behavior_term_summaries;

ALTER TABLE IF EXISTS behavior_categories
    DROP COLUMN IF EXISTS category_type;

DROP TYPE IF EXISTS behavior_category_type;

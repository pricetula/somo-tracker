-- Migration: 000005_extend_summaries_and_daily (down)
-- Reverses the changes from the up migration.

DROP TRIGGER IF EXISTS trg_class_daily_attendance_summaries_updated_at
    ON class_daily_attendance_summaries;

DROP TABLE IF EXISTS class_daily_attendance_summaries;

ALTER TABLE attendance_term_summaries
    DROP CONSTRAINT IF EXISTS fk_summaries_tenant_academic_year;

ALTER TABLE attendance_term_summaries
    DROP COLUMN IF EXISTS academic_year_id;

ALTER TABLE attendance_term_summaries
    DROP COLUMN IF EXISTS created_at;

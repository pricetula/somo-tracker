-- Migration: 000013_add_grade_id_to_subjects (down)
-- Purpose: Rollback grade_level_id addition to subjects.

-- Drop index first.
DROP INDEX IF EXISTS subjects_grade_level_id_idx;

-- Drop column.
ALTER TABLE subjects DROP COLUMN IF EXISTS grade_level_id;

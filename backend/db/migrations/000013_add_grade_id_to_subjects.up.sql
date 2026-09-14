-- Migration: 000013_add_grade_id_to_subjects
-- Purpose: Add grade_level_id to subjects to link subjects to specific grade levels.
-- Dependencies:
--   - 000004_create_sis_schema (provides grade_levels table)
--   - 000005_create_subject_hierarchy (provides subjects table)

-- ============================================================================
-- Section 1: Add grade_level_id column
-- ============================================================================

ALTER TABLE subjects
    ADD COLUMN grade_level_id UUID REFERENCES grade_levels(id) ON DELETE CASCADE;

-- Index: subjects_grade_level_id_idx supports grade-scoped subject listings.
CREATE INDEX subjects_grade_level_id_idx ON subjects (grade_level_id);

-- ============================================================================
-- Section 2: Comments
-- ============================================================================

COMMENT ON COLUMN subjects.grade_level_id IS 'FK to grade_levels(id). The grade level this subject belongs to. Cascades on grade level delete. Nullable for legacy subjects.';

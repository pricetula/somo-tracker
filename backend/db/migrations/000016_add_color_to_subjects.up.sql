-- Migration: 000016_add_color_to_subjects
-- Purpose: Add a color column to subjects for visual identification.
-- Dependencies: 000005_create_subject_hierarchy

ALTER TABLE subjects ADD COLUMN IF NOT EXISTS color VARCHAR(32);

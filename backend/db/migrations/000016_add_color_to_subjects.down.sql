-- Migration: 000016_add_color_to_subjects
-- Purpose: Revert color column addition.
-- Dependencies: 000005_create_subject_hierarchy

ALTER TABLE subjects DROP COLUMN IF EXISTS color;

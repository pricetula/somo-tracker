-- Migration: 000014_expand_subject_code
-- Purpose: Increase subjects.code length to accommodate longer CBE subject codes.
-- Dependencies: 000005_create_subject_hierarchy

ALTER TABLE subjects ALTER COLUMN code TYPE VARCHAR(64);

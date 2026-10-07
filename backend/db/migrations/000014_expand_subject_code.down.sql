-- Migration: 000014_expand_subject_code (down)
ALTER TABLE subjects ALTER COLUMN code TYPE VARCHAR(16);

-- Migration: 000023_add_student_structured_fields down
-- Reverts structured student fields

DROP INDEX IF EXISTS students_email_idx;
DROP INDEX IF EXISTS students_phone_idx;
DROP INDEX IF EXISTS students_nationality_idx;

ALTER TABLE students
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS nationality,
    DROP COLUMN IF EXISTS emergency_contact_name,
    DROP COLUMN IF EXISTS emergency_contact_phone,
    DROP COLUMN IF EXISTS special_needs;

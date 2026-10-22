-- Migration: 000023_add_student_structured_fields
-- Purpose: Add structured profile fields to students table to replace unstructured metadata usage.
-- Adds: address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs
--
-- Dependencies:
--   - 000004_create_sis_schema (students)
--   - 000022_add_kenya_identifiers (students.nemis_upn)
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

ALTER TABLE students
    ADD COLUMN address TEXT,
    ADD COLUMN phone VARCHAR(32),
    ADD COLUMN email VARCHAR(255),
    ADD COLUMN nationality VARCHAR(128),
    ADD COLUMN emergency_contact_name VARCHAR(255),
    ADD COLUMN emergency_contact_phone VARCHAR(32),
    ADD COLUMN special_needs TEXT;

-- Indexes for searchability
CREATE INDEX students_email_idx ON students (email) WHERE email IS NOT NULL;
CREATE INDEX students_phone_idx ON students (phone) WHERE phone IS NOT NULL;
CREATE INDEX students_nationality_idx ON students (nationality) WHERE nationality IS NOT NULL;

COMMENT ON COLUMN students.address IS 'Student residential address.';
COMMENT ON COLUMN students.phone IS 'Student phone number.';
COMMENT ON COLUMN students.email IS 'Student email address.';
COMMENT ON COLUMN students.nationality IS 'Student nationality.';
COMMENT ON COLUMN students.emergency_contact_name IS 'Name of emergency contact.';
COMMENT ON COLUMN students.emergency_contact_phone IS 'Phone number of emergency contact.';
COMMENT ON COLUMN students.special_needs IS 'Notes on special needs or accommodations.';

-- Migration rollback: 000022_add_kenya_identifiers

DROP INDEX IF EXISTS schools_nemis_school_code_idx;
DROP INDEX IF EXISTS schools_nemis_school_code_uniq_idx;

DROP INDEX IF EXISTS school_memberships_tsc_number_idx;
DROP INDEX IF EXISTS school_memberships_tsc_number_uniq_idx;

DROP INDEX IF EXISTS students_nemis_upn_idx;
DROP INDEX IF EXISTS students_nemis_upn_uniq_idx;

ALTER TABLE school_memberships DROP CONSTRAINT IF EXISTS school_memberships_tsc_number_role_check;

ALTER TABLE schools DROP COLUMN IF EXISTS nemis_school_code;
ALTER TABLE school_memberships DROP COLUMN IF EXISTS tsc_number;
ALTER TABLE students DROP COLUMN IF EXISTS nemis_upn;

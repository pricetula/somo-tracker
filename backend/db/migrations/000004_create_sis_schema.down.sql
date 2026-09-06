-- Migration: 000004_create_sis_schema (down)
-- Purpose: Rollback the SIS schema (tables, enum, triggers, function).
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- Drop triggers first (they depend on the tables and function).
DROP TRIGGER IF EXISTS guardian_student_links_updated_at_trg ON guardian_student_links;
DROP TRIGGER IF EXISTS students_updated_at_trg ON students;
DROP TRIGGER IF EXISTS school_memberships_updated_at_trg ON school_memberships;
DROP TRIGGER IF EXISTS schools_updated_at_trg ON schools;
DROP TRIGGER IF EXISTS grade_levels_updated_at_trg ON grade_levels;
DROP TRIGGER IF EXISTS education_systems_updated_at_trg ON education_systems;
DROP TRIGGER IF EXISTS countries_updated_at_trg ON countries;

-- Drop the trigger helper function.
DROP FUNCTION IF EXISTS set_updated_at();

-- Drop tables in reverse dependency order.
DROP TABLE IF EXISTS guardian_student_links;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS school_memberships;
DROP TABLE IF EXISTS schools;
DROP TABLE IF EXISTS grade_levels;
DROP TABLE IF EXISTS education_systems;
DROP TABLE IF EXISTS countries;

-- Drop the user_role enum last (after all tables that reference it are gone).
DROP TYPE IF EXISTS user_role;
-- Migration: 000004_create_sis_schema
-- Purpose: Create the multi-country School Information System (SIS) schema:
--   user_role enum, reference tables (countries, education_systems, grade_levels),
--   tenant-scoped schools, school_memberships, students, guardian_student_links.
--
-- Dependencies:
--   - 000001_init_extensions (pgcrypto for gen_random_uuid()).
--   - 000002_create_tenants_and_users (provides tenants + users tables).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 0: user_role enum
-- ============================================================================

CREATE TYPE user_role AS ENUM ('ADMIN', 'TEACHER', 'GUARDIAN', 'FINANCE');

-- ============================================================================
-- Section 1: countries
-- ============================================================================

-- Reference table of countries supported by the SIS. Country codes follow
-- ISO 3166-1 alpha-2 (e.g., KE, TZ, UG, RW).
CREATE TABLE countries (
    id            UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    country_name  VARCHAR(255) NOT NULL,
    country_code  VARCHAR(2)   NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    CONSTRAINT countries_country_code_uniq UNIQUE (country_code)
);

-- Index: countries_country_code_idx covers lookups by ISO code (e.g., routing).
CREATE INDEX countries_country_code_idx ON countries (country_code);
-- Index: countries_created_at_idx supports time-bucketed admin listing.
CREATE INDEX countries_created_at_idx ON countries (created_at DESC);

-- ============================================================================
-- Section 2: education_systems
-- ============================================================================

-- Reference table of education systems (e.g., Competency-Based Education / CBE).
CREATE TABLE education_systems (
    id            UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    system_name   VARCHAR(255) NOT NULL,
    description   TEXT,
    created_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: education_systems_system_name_idx covers lookups by system name.
CREATE INDEX education_systems_system_name_idx ON education_systems (system_name);

-- ============================================================================
-- Section 3: grade_levels
-- ============================================================================

-- Grade levels scoped to a specific education system and country.
-- Example: (CBE, Kenya) -> PP1, Grade 1..6, Grade 7..9 (Junior Sec), Grade 10..12 (Senior Sec).
CREATE TABLE grade_levels (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                         PRIMARY KEY,
    education_system_id UUID        NOT NULL    REFERENCES education_systems(id)
                                         ON DELETE CASCADE,
    country_id          UUID        NOT NULL    REFERENCES countries(id)
                                         ON DELETE CASCADE,
    tier_stage          VARCHAR(64)  NOT NULL,
    local_label         VARCHAR(64)  NOT NULL,
    sequence_index      INTEGER      NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    -- A grade level's sequence position must be unique within a system+country
    -- to guarantee deterministic chronological ordering.
    CONSTRAINT grade_levels_sys_country_seq_uniq UNIQUE (education_system_id, country_id, sequence_index)
);

-- Index: grade_levels_education_system_id_idx supports system-scoped lookups.
CREATE INDEX grade_levels_education_system_id_idx ON grade_levels (education_system_id);
-- Index: grade_levels_country_id_idx supports country-scoped lookups.
CREATE INDEX grade_levels_country_id_idx ON grade_levels (country_id);
-- Composite index: grade_levels_sys_country_seq_idx covers the primary
-- ordered listing query (system + country ordered by sequence_index).
CREATE INDEX grade_levels_sys_country_seq_idx ON grade_levels (education_system_id, country_id, sequence_index);

-- ============================================================================
-- Section 4: schools
-- ============================================================================

-- Schools operated by a tenant. Each school belongs to exactly one tenant,
-- one country, and one education system.
CREATE TABLE schools (
    id                UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    tenant_id         UUID        NOT NULL    REFERENCES tenants(id)
                                       ON DELETE CASCADE,
    school_name       VARCHAR(255) NOT NULL,
    country_id        UUID        NOT NULL    REFERENCES countries(id)
                                       ON DELETE CASCADE,
    education_system_id UUID      NOT NULL    REFERENCES education_systems(id)
                                       ON DELETE CASCADE,
    created_at        TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: schools_tenant_id_idx is the primary access path (tenant-scoped queries).
CREATE INDEX schools_tenant_id_idx ON schools (tenant_id);
-- Index: schools_country_id_idx supports country-level aggregations.
CREATE INDEX schools_country_id_idx ON schools (country_id);
-- Index: schools_education_system_id_idx supports system-level aggregations.
CREATE INDEX schools_education_system_id_idx ON schools (education_system_id);

-- ============================================================================
-- Section 5: school_memberships
-- ============================================================================

-- Links a user to a school with a role. A user may have at most one
-- membership record per school (enforced by school_id + user_id unique).
CREATE TABLE school_memberships (
    id          UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    school_id   UUID        NOT NULL    REFERENCES schools(id)
                                       ON DELETE CASCADE,
    user_id     UUID        NOT NULL    REFERENCES users(id)
                                       ON DELETE CASCADE,
    role        user_role   NOT NULL,
    is_active   BOOLEAN     NOT NULL    DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    CONSTRAINT school_memberships_school_user_uniq UNIQUE (school_id, user_id)
);

-- Index: school_memberships_school_id_idx supports school-scoped member listings.
CREATE INDEX school_memberships_school_id_idx ON school_memberships (school_id);
-- Index: school_memberships_user_id_idx supports user→school reverse lookups.
-- Added/verified for high-performance GET /me query joins (unique constraint leads with school_id).
CREATE INDEX school_memberships_user_id_idx ON school_memberships (user_id);
-- Index: school_memberships_role_idx supports role-based filtering.
CREATE INDEX school_memberships_role_idx ON school_memberships (role);
-- Partial unique index: at most one active membership per user.
CREATE UNIQUE INDEX school_memberships_active_user_uniq_idx ON school_memberships (user_id) WHERE is_active = true;

-- ============================================================================
-- Section 6: students
-- ============================================================================

-- Student records scoped to a school. admission_number is unique within a school.
CREATE TABLE students (
    student_id       UUID        NOT NULL    DEFAULT gen_random_uuid()
                                         PRIMARY KEY,
    school_id        UUID        NOT NULL    REFERENCES schools(id)
                                         ON DELETE CASCADE,
    admission_number VARCHAR(255) NOT NULL,
    full_name        VARCHAR(255) NOT NULL,
    date_of_birth    DATE        NOT NULL,
    gender           VARCHAR(32) NOT NULL,
    metadata         JSONB       NOT NULL    DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    CONSTRAINT students_school_admission_uniq UNIQUE (school_id, admission_number)
);

-- Index: students_school_id_idx is the primary access path (school-scoped queries).
CREATE INDEX students_school_id_idx ON students (school_id);
-- Index: students_admission_number_idx supports admission-number lookups.
CREATE INDEX students_admission_number_idx ON students (admission_number);
-- Index: students_full_name_idx supports name-based search.
CREATE INDEX students_full_name_idx ON students (full_name);

-- ============================================================================
-- Section 7: guardian_student_links
-- ============================================================================

-- Links a guardian (via their school_membership) to a student.
-- Relationship type captures the guardian's role (Parent, Legal Guardian, Sponsor, etc.).
CREATE TABLE guardian_student_links (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                         PRIMARY KEY,
    school_membership_id UUID      NOT NULL    REFERENCES school_memberships(id)
                                         ON DELETE CASCADE,
    student_id          UUID        NOT NULL    REFERENCES students(student_id)
                                         ON DELETE CASCADE,
    relationship_type   VARCHAR(64) NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    CONSTRAINT guardian_student_links_membership_student_uniq UNIQUE (school_membership_id, student_id)
);

-- Index: guardian_student_links_school_membership_id_idx for guardian→students lookup.
CREATE INDEX guardian_student_links_school_membership_id_idx ON guardian_student_links (school_membership_id);
-- Index: guardian_student_links_student_id_idx for student→guardians lookup.
CREATE INDEX guardian_student_links_student_id_idx ON guardian_student_links (student_id);

-- ============================================================================
-- Section 8: updated_at trigger helper
-- ============================================================================

-- Generic trigger function to maintain updated_at on BEFORE UPDATE.
-- Mirrors the update_last_seen() pattern from migration 000003 but targets
-- the conventional updated_at column name used across all SIS tables.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER countries_updated_at_trg
    BEFORE UPDATE ON countries
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER education_systems_updated_at_trg
    BEFORE UPDATE ON education_systems
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER grade_levels_updated_at_trg
    BEFORE UPDATE ON grade_levels
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER schools_updated_at_trg
    BEFORE UPDATE ON schools
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER school_memberships_updated_at_trg
    BEFORE UPDATE ON school_memberships
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER students_updated_at_trg
    BEFORE UPDATE ON students
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER guardian_student_links_updated_at_trg
    BEFORE UPDATE ON guardian_student_links
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- Section 9: Comments (documentation)
-- ============================================================================

COMMENT ON TYPE user_role IS 'Role enum for school_memberships: ADMIN, TEACHER, GUARDIAN, FINANCE.';

COMMENT ON TABLE countries IS 'Reference table of countries supported by the SIS (ISO 3166-1 alpha-2 codes).';
COMMENT ON COLUMN countries.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN countries.country_name IS 'Human-readable country name (e.g. Kenya).';
COMMENT ON COLUMN countries.country_code IS 'ISO 3166-1 alpha-2 code (e.g. KE). Unique.';
COMMENT ON COLUMN countries.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN countries.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE education_systems IS 'Reference table of education systems (e.g. Competency-Based Education / CBE).';
COMMENT ON COLUMN education_systems.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN education_systems.system_name IS 'Human-readable system name (e.g. Competency-Based Education / CBE).';
COMMENT ON COLUMN education_systems.description IS 'Optional free-text description of the education system.';
COMMENT ON COLUMN education_systems.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN education_systems.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE grade_levels IS 'Grade levels scoped to an education system and country (e.g. PP1, Grade 7, Senior 1).';
COMMENT ON COLUMN grade_levels.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN grade_levels.education_system_id IS 'FK to education_systems(id). Cascades on education system delete.';
COMMENT ON COLUMN grade_levels.country_id IS 'FK to countries(id). Cascades on country delete.';
COMMENT ON COLUMN grade_levels.tier_stage IS 'Broad stage bucket: pre_primary, primary, lower_secondary, upper_secondary.';
COMMENT ON COLUMN grade_levels.local_label IS 'Local label used in that country/system (e.g. PP1, Grade 7, Senior 1).';
COMMENT ON COLUMN grade_levels.sequence_index IS 'Integer for chronological sorting; unique within system+country.';
COMMENT ON COLUMN grade_levels.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN grade_levels.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE schools IS 'Schools operated by a tenant. Each school belongs to one tenant, country, and education system.';
COMMENT ON COLUMN schools.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN schools.tenant_id IS 'FK to tenants(id). The B2B organization that operates the school. Cascades on tenant delete.';
COMMENT ON COLUMN schools.school_name IS 'Human-readable school name.';
COMMENT ON COLUMN schools.country_id IS 'FK to countries(id). Cascades on country delete.';
COMMENT ON COLUMN schools.education_system_id IS 'FK to education_systems(id). Cascades on education system delete.';
COMMENT ON COLUMN schools.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN schools.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE school_memberships IS 'Links a user to a school with a role. A user has at most one membership per school, and at most one active membership across all schools.';
COMMENT ON COLUMN school_memberships.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN school_memberships.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN school_memberships.user_id IS 'FK to users(id). Cascades on user delete.';
COMMENT ON COLUMN school_memberships.role IS 'Role within the school (user_role enum).';
COMMENT ON COLUMN school_memberships.is_active IS 'Whether this membership is the user''s currently active school. Enforced by partial unique index: only one active per user.';
COMMENT ON COLUMN school_memberships.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN school_memberships.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE students IS 'Student records scoped to a school. admission_number is unique per school. metadata stores flexible external identifiers (NEMIS, KICD, etc.).';
COMMENT ON COLUMN students.student_id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN students.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN students.admission_number IS 'School-scoped admission number (unique per school).';
COMMENT ON COLUMN students.full_name IS 'Student full name (first + last).';
COMMENT ON COLUMN students.date_of_birth IS 'Date of birth.';
COMMENT ON COLUMN students.gender IS 'Gender (free-text for international flexibility).';
COMMENT ON COLUMN students.metadata IS 'JSONB for flexible external identifiers (e.g. NEMIS, KICD tracking codes).';
COMMENT ON COLUMN students.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN students.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE guardian_student_links IS 'Links a guardian (via school_membership) to a student with a relationship type.';
COMMENT ON COLUMN guardian_student_links.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN guardian_student_links.school_membership_id IS 'FK to school_memberships(id). Cascades on membership delete.';
COMMENT ON COLUMN guardian_student_links.student_id IS 'FK to students(student_id). Cascades on student delete.';
COMMENT ON COLUMN guardian_student_links.relationship_type IS 'Relationship type (e.g. Parent, Legal Guardian, Sponsor).';
COMMENT ON COLUMN guardian_student_links.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN guardian_student_links.updated_at IS 'UTC timestamp of last modification.';
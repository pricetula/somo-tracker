-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- user_role: user role for each member in a schools
CREATE TYPE user_role AS ENUM ('OWNER', 'ADMIN', 'HEADTEACHER', 'TEACHER', 'GUARDIAN', 'FINANCE', 'PLATFORMOPERATOR');
-- tenant_status: status for a tenant
CREATE TYPE tenant_status AS ENUM ('ACTIVE', 'SUSPENDED');

-- Country: table of countries supported by the SIS. Country codes follow
-- ISO 3166-1 alpha-2 (e.g., KE, TZ, UG, RW).
CREATE TABLE countries (
    id            UUID        NOT NULL    DEFAULT gen_random_uuid() PRIMARY KEY,
    country_name  VARCHAR(255) NOT NULL,
    country_code  VARCHAR(3)   NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    CONSTRAINT countries_country_code_uniq UNIQUE (country_code)
);

-- Index: countries_country_code_idx covers lookups by ISO code (e.g., routing).
CREATE INDEX countries_country_code_idx ON countries (country_code);
-- Index: countries_created_at_idx supports time-bucketed admin listing.
CREATE INDEX countries_created_at_idx ON countries (created_at DESC);

-- Reference table of education systems (e.g., Competency-Based Education / CBE).
-- Each education system belongs to exactly one country.
CREATE TABLE education_systems (
    id            UUID        NOT NULL    DEFAULT gen_random_uuid() PRIMARY KEY,
    country_id    UUID        NOT NULL    REFERENCES countries(id) ON DELETE CASCADE,
    system_name   VARCHAR(255) NOT NULL,
    description   TEXT,
    created_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: education_systems_country_id_idx supports country-scoped lookups.
CREATE INDEX education_systems_country_id_idx ON education_systems (country_id);
-- Index: education_systems_system_name_idx covers lookups by system name.
CREATE INDEX education_systems_system_name_idx ON education_systems (system_name);
-- Composite unique: a country cannot have duplicate system names.
ALTER TABLE education_systems ADD CONSTRAINT education_systems_country_name_uniq UNIQUE (country_id, system_name);

-- Grade levels scoped to a specific education system and country.
-- Example: (CBE, Kenya) -> PP1, Grade 1..6, Grade 7..9 (Junior Sec), Grade 10..12 (Senior Sec).
CREATE TABLE class_levels (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid() PRIMARY KEY,
    education_system_id UUID        NOT NULL    REFERENCES education_systems(id) ON DELETE CASCADE,
    tier_stage          VARCHAR(64)  NOT NULL,
    local_label         VARCHAR(64)  NOT NULL,
    sequence_index      INTEGER      NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    -- A grade level's sequence position must be unique within a system+country
    -- to guarantee deterministic chronological ordering.
    CONSTRAINT class_levels_sys_country_seq_uniq UNIQUE (education_system_id, sequence_index)
);

-- Index: class_levels_education_system_id_idx supports system-scoped lookups.
CREATE INDEX class_levels_education_system_id_idx ON class_levels (education_system_id);
-- Composite index: class_levels_sys_seq_idx covers the primary
-- ordered listing query (system + sequence_index).
CREATE INDEX class_levels_sys_seq_idx ON class_levels (education_system_id, sequence_index);

-- Tenant: organization boundary for data isolation
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL UNIQUE,
    stytch_organization_id UUID UNIQUE,
     tenant_status        tenant_status   NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_stytch_org ON tenants (stytch_organization_id) WHERE stytch_organization_id IS NOT NULL;

-- User: identity per tenants
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email_normalized TEXT NOT NULL,
    stytch_member_id UUID UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, email_normalized)
);

CREATE INDEX IF NOT EXISTS idx_user_tenant ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_stytch_member ON users (stytch_member_id) WHERE stytch_member_id IS NOT NULL;

-- School: institution within tenants
CREATE TABLE IF NOT EXISTS schools (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    country_id        UUID        NOT NULL    REFERENCES countries(id) ON DELETE CASCADE,
    education_system_id UUID      NOT NULL    REFERENCES education_systems(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_school_tenant ON schools (tenant_id);
-- Index: schools_country_id_idx supports country-level aggregations.
CREATE INDEX schools_country_id_idx ON schools (country_id);
-- Index: schools_education_system_id_idx supports system-level aggregations.
CREATE INDEX schools_education_system_id_idx ON schools (education_system_id);

-- Membership: User ↔ School role binding
CREATE TABLE IF NOT EXISTS memberships (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
     role        user_role   NOT NULL,
      is_active   BOOLEAN     NOT NULL    DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, school_id)
);

CREATE INDEX IF NOT EXISTS idx_membership_user ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_membership_school ON memberships (school_id);
-- Partial unique index: at most one active membership per user.
CREATE UNIQUE INDEX memberships_active_user_uniq_idx ON memberships (user_id) WHERE is_active = true;

-- PKCECode: per-auth PKCE verifier
CREATE TABLE IF NOT EXISTS pkce_codes (
    pending_onboarding_id UUID NOT NULL REFERENCES pending_onboarding(id) ON DELETE CASCADE,
    code_verifier_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (pending_onboarding_id)
);

CREATE INDEX IF NOT EXISTS idx_pkce_expires ON pkce_codes (expires_at);

-- +goose Down
DROP TABLE IF EXISTS pkce_codes;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS schools;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS tenant_status;
DROP INDEX IF EXISTS countries_country_code_idx;
DROP INDEX IF EXISTS countries_created_at_idx;
DROP INDEX IF EXISTS education_systems_country_id_idx;
DROP INDEX IF EXISTS education_systems_system_name_idx;
DROP INDEX IF EXISTS class_levels_education_system_id_idx;
DROP INDEX IF EXISTS class_levels_sys_seq_idx;
DROP TABLE IF EXISTS class_levels;
DROP TABLE IF EXISTS education_systems;
DROP TABLE IF EXISTS countries;
DROP EXTENSION IF EXISTS "pgcrypto";

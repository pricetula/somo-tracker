-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('OWNER', 'ADMIN', 'HEADTEACHER', 'TEACHER', 'GUARDIAN', 'FINANCE', 'PLATFORMOPERATOR');
CREATE TYPE tenant_status AS ENUM ('ACTIVE', 'SUSPENDED');

-- Country: reference table of supported countries (ISO 3166-1 alpha-2).
CREATE TABLE countries (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    country_name VARCHAR(255) NOT NULL,
    country_code VARCHAR(2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT countries_country_code_uniq UNIQUE (country_code)
);

CREATE INDEX countries_country_code_idx ON countries (country_code);
CREATE INDEX countries_created_at_idx ON countries (created_at DESC);
CREATE INDEX idx_countries_deleted_at ON countries (deleted_at) WHERE deleted_at IS NULL;

-- Education system: belongs to exactly one country.
CREATE TABLE education_systems (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    system_name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT education_systems_country_name_uniq UNIQUE (country_id, system_name)
);

CREATE INDEX education_systems_country_id_idx ON education_systems (country_id);
CREATE INDEX education_systems_system_name_idx ON education_systems (system_name);
CREATE INDEX idx_education_systems_deleted_at ON education_systems (deleted_at) WHERE deleted_at IS NULL;

-- Grade levels scoped to an education system.
CREATE TABLE class_levels (
    id UUID NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE RESTRICT,
    tier_stage VARCHAR(64) NOT NULL,
    local_label VARCHAR(64) NOT NULL,
    sequence_index INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT class_levels_sys_country_seq_uniq UNIQUE (education_system_id, sequence_index)
);

CREATE INDEX class_levels_education_system_id_idx ON class_levels (education_system_id);
CREATE INDEX class_levels_sys_seq_idx ON class_levels (education_system_id, sequence_index);
CREATE INDEX idx_class_levels_deleted_at ON class_levels (deleted_at) WHERE deleted_at IS NULL;

-- Tenant: organization boundary for data isolation.
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL UNIQUE,
    stytch_organization_id UUID UNIQUE,
    tenant_status tenant_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tenant_stytch_org ON tenants (stytch_organization_id) WHERE stytch_organization_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants (deleted_at) WHERE deleted_at IS NULL;

-- User: identity per tenant.
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email_normalized TEXT NOT NULL,
    stytch_member_id UUID UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, email_normalized)
);

CREATE INDEX IF NOT EXISTS idx_user_tenant ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_stytch_member ON users (stytch_member_id) WHERE stytch_member_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users (tenant_id, email_normalized);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;

-- School: institution within a tenant.
CREATE TABLE IF NOT EXISTS schools (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_school_tenant ON schools (tenant_id);
CREATE INDEX IF NOT EXISTS idx_schools_country_id ON schools (country_id);
CREATE INDEX IF NOT EXISTS idx_schools_education_system_id ON schools (education_system_id);
CREATE INDEX IF NOT EXISTS idx_schools_deleted_at ON schools (deleted_at) WHERE deleted_at IS NULL;

-- Membership: User <-> School role binding.
CREATE TABLE IF NOT EXISTS memberships (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    role user_role NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, school_id)
);

CREATE INDEX IF NOT EXISTS idx_membership_user ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_membership_school ON memberships (school_id);
CREATE INDEX IF NOT EXISTS idx_memberships_role ON memberships (role);
CREATE INDEX IF NOT EXISTS idx_memberships_school_role ON memberships (school_id, role);
CREATE INDEX IF NOT EXISTS idx_memberships_is_active ON memberships (is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_memberships_school_tenant ON memberships (school_id, user_id);
CREATE UNIQUE INDEX memberships_active_user_uniq_idx ON memberships (user_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_memberships_deleted_at ON memberships (deleted_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS memberships_active_user_uniq_idx;
DROP INDEX IF EXISTS idx_memberships_deleted_at;
DROP INDEX IF EXISTS idx_memberships_school_tenant;
DROP INDEX IF EXISTS idx_memberships_is_active;
DROP INDEX IF EXISTS idx_memberships_school_role;
DROP INDEX IF EXISTS idx_memberships_role;
DROP INDEX IF EXISTS idx_membership_school;
DROP INDEX IF EXISTS idx_membership_user;
DROP TABLE IF EXISTS memberships;
DROP INDEX IF EXISTS idx_schools_deleted_at;
DROP INDEX IF EXISTS idx_schools_education_system_id;
DROP INDEX IF EXISTS idx_schools_country_id;
DROP INDEX IF EXISTS idx_school_tenant;
DROP TABLE IF EXISTS schools;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_tenant_email;
DROP INDEX IF EXISTS idx_user_stytch_member;
DROP INDEX IF EXISTS idx_user_tenant;
DROP TABLE IF EXISTS users;
DROP INDEX IF EXISTS idx_tenants_deleted_at;
DROP INDEX IF EXISTS idx_tenant_stytch_org;
DROP TABLE IF EXISTS tenants;
DROP INDEX IF EXISTS idx_class_levels_deleted_at;
DROP INDEX IF EXISTS class_levels_sys_seq_idx;
DROP INDEX IF EXISTS class_levels_education_system_id_idx;
DROP TABLE IF EXISTS class_levels;
DROP INDEX IF EXISTS idx_education_systems_deleted_at;
DROP INDEX IF EXISTS education_systems_system_name_idx;
DROP INDEX IF EXISTS education_systems_country_id_idx;
DROP TABLE IF EXISTS education_systems;
DROP INDEX IF EXISTS idx_countries_deleted_at;
DROP INDEX IF EXISTS countries_created_at_idx;
DROP INDEX IF EXISTS countries_country_code_idx;
DROP TABLE IF EXISTS countries;
DROP TYPE IF EXISTS tenant_status;
DROP TYPE IF EXISTS user_role;

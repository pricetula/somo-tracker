-- +goose Up

CREATE TYPE organization_status AS ENUM ('ACTIVE', 'SUSPENDED', 'REVOKED');
CREATE TYPE user_status AS ENUM ('ACTIVE', 'SUSPENDED', 'REVOKED');
CREATE TYPE school_status AS ENUM ('ACTIVE', 'SUSPENDED', 'REVOKED');

-- Organization: top-level customer account and data-isolation boundary.
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL UNIQUE,
    status organization_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_organizations_deleted_at ON organizations (deleted_at) WHERE deleted_at IS NULL;

-- User: one global identity per normalized email.
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    normalized_email TEXT NOT NULL,
    status user_status NOT NULL DEFAULT 'ACTIVE',
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (normalized_email),
    UNIQUE (organization_id, id)
);

CREATE INDEX idx_users_organization ON users (organization_id);
CREATE INDEX idx_users_normalized_email ON users (normalized_email);
CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;

-- School: institution within an organization.
CREATE TABLE schools (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status school_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (organization_id, name),
    UNIQUE (organization_id, id)
);

CREATE INDEX idx_schools_organization ON schools (organization_id);
CREATE INDEX idx_schools_deleted_at ON schools (deleted_at) WHERE deleted_at IS NULL;

-- UserSchool: links a user to a school within their organization.
CREATE TABLE user_schools (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    UNIQUE (user_id, school_id)
);

CREATE INDEX idx_user_schools_organization ON user_schools (organization_id);
CREATE INDEX idx_user_schools_user ON user_schools (user_id);
CREATE INDEX idx_user_schools_school ON user_schools (school_id);
CREATE INDEX idx_user_schools_status ON user_schools (status) WHERE status = 'ACTIVE';

-- Composite foreign keys to prevent cross-organization assignment.
ALTER TABLE user_schools
    ADD CONSTRAINT fk_user_schools_user_org
    FOREIGN KEY (organization_id, user_id)
    REFERENCES users (organization_id, id);

ALTER TABLE user_schools
    ADD CONSTRAINT fk_user_schools_school_org
    FOREIGN KEY (organization_id, school_id)
    REFERENCES schools (organization_id, id);

-- +goose Down
DROP INDEX IF EXISTS idx_user_schools_status;
DROP INDEX IF EXISTS idx_user_schools_school;
DROP INDEX IF EXISTS idx_user_schools_user;
DROP INDEX IF EXISTS idx_user_schools_organization;
DROP INDEX IF EXISTS idx_schools_deleted_at;
DROP INDEX IF EXISTS idx_schools_organization;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_normalized_email;
DROP INDEX IF EXISTS idx_users_organization;
DROP INDEX IF EXISTS idx_organizations_deleted_at;

ALTER TABLE user_schools DROP CONSTRAINT IF EXISTS fk_user_schools_school_org;
ALTER TABLE user_schools DROP CONSTRAINT IF EXISTS fk_user_schools_user_org;

DROP TABLE IF EXISTS user_schools;
DROP TABLE IF EXISTS schools;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

DROP TYPE IF EXISTS school_status;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS organization_status;

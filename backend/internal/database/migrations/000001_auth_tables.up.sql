-- Migration: 000001_auth_tables
-- Creates core authentication and identity tables for the auth module.

-- ============================================================================
-- CREATE TYPE: user_role
-- ============================================================================
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM (
        'SYSTEM_ADMIN',
        'SCHOOL_ADMIN',
        'TEACHER',
        'SUPPORT_STAFF'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================================
-- TABLE: tenants
-- Represents the core organization or corporate entity.
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenants (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    stytch_org_id VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants (slug);
CREATE INDEX IF NOT EXISTS idx_tenants_stytch_org_id ON tenants (stytch_org_id);

-- ============================================================================
-- TABLE: users
-- Every user identity in the system shares this core schema.
-- Authentication is managed externally by Stytch via external_auth_id.
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) NOT NULL,
    tenant_id        UUID        REFERENCES tenants(id),
    first_name       VARCHAR(255) NOT NULL DEFAULT '',
    last_name        VARCHAR(255) NOT NULL DEFAULT '',
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    external_auth_id VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_external_auth_id ON users (external_auth_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);

-- ============================================================================
-- TABLE: sessions
-- Server-side opaque session token storage for cookie-based auth.
-- Each session ties a user to their tenant and device fingerprint.
-- ============================================================================
CREATE TABLE IF NOT EXISTS sessions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    token             VARCHAR(128) NOT NULL UNIQUE,
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id         UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stytch_member_id  VARCHAR(255) NOT NULL,
    stytch_org_id     VARCHAR(255) NOT NULL,
    device_fingerprint VARCHAR(128) NOT NULL DEFAULT '',
    expires_at        TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions (token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant_id ON sessions (tenant_id);

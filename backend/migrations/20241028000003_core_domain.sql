-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tenant: organization boundary for data isolation
CREATE TABLE IF NOT EXISTS tenant (
    id UUID PRIMARY KEY DEFAULT gen_uuid_v7(),
    name TEXT NOT NULL UNIQUE,
    stytch_organization_id UUID UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'suspended')) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_stytch_org ON tenant (stytch_organization_id) WHERE stytch_organization_id IS NOT NULL;

-- User: identity per tenant
CREATE TABLE IF NOT EXISTS "user" (
    id UUID PRIMARY KEY DEFAULT gen_uuid_v7(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    email_normalized TEXT NOT NULL,
    stytch_member_id UUID UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, email_normalized)
);

CREATE INDEX IF NOT EXISTS idx_user_tenant ON "user" (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_stytch_member ON "user" (stytch_member_id) WHERE stytch_member_id IS NOT NULL;

-- School: institution within tenant
CREATE TABLE IF NOT EXISTS school (
    id UUID PRIMARY KEY DEFAULT gen_uuid_v7(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_school_tenant ON school (tenant_id);

-- Membership: User ↔ School role binding
CREATE TABLE IF NOT EXISTS membership (
    id UUID PRIMARY KEY DEFAULT gen_uuid_v7(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES school(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('tenant_owner', 'head', 'admin', 'teacher', 'finance', 'guardian', 'platform_operator')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, school_id)
);

CREATE INDEX IF NOT EXISTS idx_membership_user ON membership (user_id);
CREATE INDEX IF NOT EXISTS idx_membership_school ON membership (school_id);

-- PKCECode: per-auth PKCE verifier
CREATE TABLE IF NOT EXISTS pkce_code (
    pending_onboarding_id UUID NOT NULL REFERENCES pending_onboarding(id) ON DELETE CASCADE,
    code_verifier_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (pending_onboarding_id)
);

CREATE INDEX IF NOT EXISTS idx_pkce_expires ON pkce_code (expires_at);

-- +goose Down
DROP TABLE IF EXISTS pkce_code;
DROP TABLE IF EXISTS membership;
DROP TABLE IF EXISTS school;
DROP TABLE IF EXISTS "user";
DROP TABLE IF EXISTS tenant;
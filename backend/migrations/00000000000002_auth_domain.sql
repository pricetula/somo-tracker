-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE pending_onboarding_status AS ENUM ('pending','provisioning','completed','failed_retryable');

CREATE TABLE pending_onboarding (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    verified_email_normalized TEXT NOT NULL UNIQUE,
    status pending_onboarding_status NOT NULL DEFAULT 'pending',
    current_step TEXT NOT NULL DEFAULT '',
    form_data JSONB NOT NULL DEFAULT '{}',
    stytch_organization_id UUID,
    stytch_member_id UUID,
    tenant_id UUID,
    school_name TEXT,
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pending_onboarding_email ON pending_onboarding (verified_email_normalized);
CREATE INDEX IF NOT EXISTS idx_pending_onboarding_tenant ON pending_onboarding (tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pending_onboarding_deleted_at ON pending_onboarding (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE app_session (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    stytch_member_session_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_app_session_user ON app_session (user_id);
CREATE INDEX IF NOT EXISTS idx_app_session_tenant ON app_session (tenant_id);
CREATE INDEX IF NOT EXISTS idx_app_session_expires ON app_session (expires_at);
CREATE INDEX IF NOT EXISTS idx_app_session_deleted_at ON app_session (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE onboarding_ticket (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    pending_onboarding_id UUID NOT NULL REFERENCES pending_onboarding(id) ON DELETE CASCADE,
    ticket_hash BYTEA NOT NULL UNIQUE,
    ist_ciphertext BYTEA,
    ist_nonce BYTEA,
    ist_key_version SMALLINT NOT NULL DEFAULT 1,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_onboarding_ticket_pending ON onboarding_ticket (pending_onboarding_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_ticket_deleted_at ON onboarding_ticket (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE pkce_codes (
    pending_onboarding_id UUID NOT NULL REFERENCES pending_onboarding(id) ON DELETE CASCADE,
    code_verifier_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (pending_onboarding_id)
);

CREATE INDEX IF NOT EXISTS idx_pkce_expires ON pkce_codes (expires_at);
CREATE INDEX IF NOT EXISTS idx_pkce_codes_deleted_at ON pkce_codes (deleted_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS pkce_codes;
DROP TABLE IF EXISTS onboarding_ticket;
DROP TABLE IF EXISTS app_session;
DROP TABLE IF EXISTS pending_onboarding;
DROP TYPE IF EXISTS pending_onboarding_status;

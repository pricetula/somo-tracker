-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE OR REPLACE FUNCTION gen_uuid_v7()
RETURNS uuid AS $$
SELECT gen_random_uuid();
$$ LANGUAGE sql IMMUTABLE;

CREATE TABLE IF NOT EXISTS pending_onboarding (
    id UUID PRIMARY KEY DEFAULT gen_uuid_v7(),
    verified_email_normalized TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending','provisioning','completed','failed_retryable')),
    current_step TEXT NOT NULL DEFAULT '',
    form_data JSONB NOT NULL DEFAULT '{}',
    stytch_organization_id UUID,
    stytch_member_id UUID,
    tenant_id UUID,
    school_name TEXT,
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pending_onboarding_email ON pending_onboarding (verified_email_normalized);
CREATE INDEX IF NOT EXISTS idx_pending_onboarding_tenant ON pending_onboarding (tenant_id) WHERE tenant_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS pending_onboarding;

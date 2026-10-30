-- +goose Up
CREATE TABLE IF NOT EXISTS app_session (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    stytch_member_session_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_app_session_user ON app_session (user_id);
CREATE INDEX IF NOT EXISTS idx_app_session_tenant ON app_session (tenant_id);
CREATE INDEX IF NOT EXISTS idx_app_session_expires ON app_session (expires_at);

-- +goose Down
DROP TABLE IF EXISTS app_session;

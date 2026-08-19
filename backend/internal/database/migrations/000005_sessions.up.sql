-- Migration: 000005_sessions
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: sessions

CREATE TABLE IF NOT EXISTS sessions (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    token                VARCHAR(128) NULL,
    token_hash           TEXT         NULL,
    user_id              UUID         NOT NULL,
    tenant_id            UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stytch_member_id     VARCHAR(255) NOT NULL,
    stytch_org_id        VARCHAR(255) NOT NULL,
    stytch_session_token VARCHAR(512) NOT NULL DEFAULT '',
    device_fingerprint   VARCHAR(128) NOT NULL DEFAULT '',
    expires_at           TIMESTAMPTZ  NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_sessions_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_token                ON sessions (token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id              ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant_id            ON sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_sessions_stytch_session_token ON sessions (stytch_session_token);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions (token_hash);

COMMENT ON COLUMN sessions.token IS
    'DEPRECATED and will be removed in a future migration. This column is now nullable.
All lookups should use token_hash instead. New sessions will insert NULL here.';

COMMENT ON COLUMN sessions.token_hash IS
    'SHA-256 hash of the session token (hex-encoded). Use this for token
     lookups instead of the raw token column.';

COMMENT ON COLUMN sessions.stytch_session_token IS
    'TODO: stytch_session_token is a third-party session token from
     Stytch, not one this schema issues. Hashing strategy for Stytch tokens
     requires app-team sign-off — do not implement hashing for this column
     without a reviewed design doc.';

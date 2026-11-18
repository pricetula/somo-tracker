-- +goose Up

CREATE TABLE login_challenges (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email_normalized TEXT NOT NULL,
    purpose TEXT NOT NULL DEFAULT 'login',
    code_mac BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    used_at TIMESTAMPTZ,
    invalidated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_challenges_email ON login_challenges (email_normalized);
CREATE INDEX idx_login_challenges_expires ON login_challenges (expires_at);

CREATE TABLE email_outbox (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    challenge_id UUID NOT NULL REFERENCES login_challenges(id) ON DELETE CASCADE,
    message_type TEXT NOT NULL DEFAULT 'otp',
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    provider_message_id TEXT,
    encrypted_payload BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ
);

CREATE INDEX idx_email_outbox_challenge ON email_outbox (challenge_id);
CREATE INDEX idx_email_outbox_status ON email_outbox (status);
CREATE INDEX idx_email_outbox_next_attempt ON email_outbox (next_attempt_at) WHERE status = 'pending';

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    token_hash BYTEA NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    active_school_id UUID REFERENCES schools(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT
);

CREATE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_organization ON sessions (organization_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_expires;
DROP INDEX IF EXISTS idx_sessions_organization;
DROP INDEX IF EXISTS idx_sessions_user;
DROP INDEX IF EXISTS idx_sessions_token_hash;

DROP TABLE IF EXISTS sessions;

DROP INDEX IF EXISTS idx_email_outbox_next_attempt;
DROP INDEX IF EXISTS idx_email_outbox_status;
DROP INDEX IF EXISTS idx_email_outbox_challenge;

DROP TABLE IF EXISTS email_outbox;

DROP INDEX IF EXISTS idx_login_challenges_expires;
DROP INDEX IF EXISTS idx_login_challenges_email;

DROP TABLE IF EXISTS login_challenges;

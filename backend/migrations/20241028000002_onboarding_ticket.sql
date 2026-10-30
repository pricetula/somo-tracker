-- +goose Up
CREATE TABLE IF NOT EXISTS onboarding_ticket (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    pending_onboarding_id UUID NOT NULL REFERENCES pending_onboarding(id) ON DELETE CASCADE,
    ticket_hash BYTEA NOT NULL UNIQUE,
    encrypted_ist BYTEA,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_onboarding_ticket_pending ON onboarding_ticket (pending_onboarding_id);

-- +goose Down
DROP TABLE IF EXISTS onboarding_ticket;

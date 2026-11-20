-- +goose Up

-- Track when an organization has completed onboarding.
ALTER TABLE organizations
    ADD COLUMN onboarding_completed_at TIMESTAMPTZ;

-- Distinguish temporary (onboarding) sessions from full sessions.
ALTER TABLE sessions
    ADD COLUMN session_type TEXT NOT NULL DEFAULT 'full'
    CHECK (session_type IN ('temporary', 'full'));

-- User display name collected during onboarding.
ALTER TABLE users
    ADD COLUMN name TEXT;

-- +goose Down

ALTER TABLE users DROP COLUMN name;

ALTER TABLE sessions DROP COLUMN session_type;

ALTER TABLE organizations DROP COLUMN onboarding_completed_at;

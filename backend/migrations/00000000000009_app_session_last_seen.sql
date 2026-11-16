-- +goose Up
-- Add last_seen_at to app_session for idle timeout / sliding window session policy.
-- last_seen_at is updated only when older than a threshold (e.g., 5 minutes)
-- to avoid writing on every request.
ALTER TABLE app_session
ADD COLUMN last_seen_at TIMESTAMPTZ;

CREATE INDEX idx_app_session_last_seen ON app_session (last_seen_at);

-- +goose Down
DROP INDEX IF EXISTS idx_app_session_last_seen;
ALTER TABLE app_session
DROP COLUMN last_seen_at;

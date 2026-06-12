-- Migration: 000002_add_stytch_session_token
-- Adds stytch_session_token column to sessions table for storing the
-- actual Stytch session token alongside our opaque reference.

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS stytch_session_token VARCHAR(512) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_sessions_stytch_session_token ON sessions (stytch_session_token);

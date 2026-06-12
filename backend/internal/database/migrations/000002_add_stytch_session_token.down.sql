-- Migration: 000002_add_stytch_session_token (rollback)

DROP INDEX IF EXISTS idx_sessions_stytch_session_token;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS stytch_session_token;

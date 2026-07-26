-- Migration: 000015_deprecate_session_token_column (rollback)
-- Revert changes made to sessions.token column.

-- Make the 'token' column NOT NULL again.
-- This assumes no NULL values were inserted while the column was nullable.
ALTER TABLE sessions ALTER COLUMN token SET NOT NULL;

-- Re-add the unique constraint on 'token'.
ALTER TABLE sessions ADD CONSTRAINT sessions_token_key UNIQUE (token);

COMMENT ON COLUMN sessions.token IS
    'Raw session token. New code should read token_hash instead.
     This column will be dropped in a future migration after the app is
     confirmed fully migrated to hash-based lookups. Do NOT write to this
     column in new code.';

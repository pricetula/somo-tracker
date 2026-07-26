-- Migration: 000015_deprecate_session_token_column
-- Deprecate the sessions.token column by making it nullable and non-unique.
-- All lookups should use sessions.token_hash instead.

-- Drop the unique constraint on 'token' to allow multiple NULLs,
-- as new sessions will insert NULL for this column.
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_token_key;

-- Make the 'token' column nullable.
ALTER TABLE sessions ALTER COLUMN token DROP NOT NULL;

COMMENT ON COLUMN sessions.token IS
    'DEPRECATED and will be removed in a future migration. This column is now nullable.
All lookups should use token_hash instead. New sessions will insert NULL here.';

-- Migration: 000001_init_extensions
-- Purpose: Enable PostgreSQL extensions required by the Somotracker backend.
--
-- Extensions created here:
--   - uuid-ossp : built-in contrib — UUID v1/v4 generation (fallback for gen_random_uuid).
--   - pgcrypto  : built-in contrib — digest(), gen_random_uuid(), crypt(), etc.
--   - pg_uuidv7 : third-party extension (github.com/tvondra/pg_uuidv7).
--                 Installed only when present in the image, so subsequent migrations
--                 that rely on it must add a version-gate comment.
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- uuid-ossp ships with every postgres:16 image (contrib module).
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- pgcrypto ships with every postgres:16 image (contrib module).
-- Provides: digest(), gen_random_uuid(), gen_salt(), crypt(), hmac().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- pg_uuidv7 is a third-party extension. It is NOT included in the official
-- postgres image. If it has been installed into the running image (e.g. via a
-- custom Dockerfile that adds github.com/tvondra/pg_uuidv7), create it here.
-- If not installed, the DO block silently skips it and the migration succeeds
-- without it; downstream migrations that require it must document that
-- dependency explicitly.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_available_extensions
        WHERE name = 'pg_uuidv7'
    ) THEN
        CREATE EXTENSION IF NOT EXISTS "pg_uuidv7";
    ELSE
        RAISE NOTICE 'pg_uuidv7 extension not available in this image — skipping.';
    END IF;
END
$$;

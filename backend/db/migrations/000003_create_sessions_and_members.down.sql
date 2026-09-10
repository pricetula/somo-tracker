-- Migration: 000003_create_sessions_and_members
-- Purpose: Rollback sessions and members tables.
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

DROP TRIGGER IF EXISTS members_updated_at_trg ON members;
DROP TRIGGER IF EXISTS sessions_last_seen_trg ON sessions;
DROP FUNCTION IF EXISTS update_last_seen();

DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS sessions;

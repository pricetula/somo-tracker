-- Migration: 000005_create_subject_hierarchy (down)
-- Purpose: Rollback the subject–topic–sub-topic curriculum hierarchy.
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- Drop triggers first.
DROP TRIGGER IF EXISTS sub_topics_updated_at_trg ON sub_topics;
DROP TRIGGER IF EXISTS topics_updated_at_trg ON topics;
DROP TRIGGER IF EXISTS subjects_updated_at_trg ON subjects;

-- Drop tables in reverse dependency order.
DROP TABLE IF EXISTS sub_topics;
DROP TABLE IF EXISTS topics;
DROP TABLE IF EXISTS subjects;
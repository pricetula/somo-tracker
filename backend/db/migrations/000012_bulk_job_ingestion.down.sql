-- Migration: 000012_bulk_job_ingestion (down)
-- Purpose: Rollback generic bulk-job ingestion tables.
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- Drop triggers first (reuse order from up).
DROP TRIGGER IF EXISTS bulk_job_items_updated_at_trg ON bulk_job_items;
DROP TRIGGER IF EXISTS bulk_jobs_updated_at_trg ON bulk_jobs;

-- Drop indexes (reverse creation order, though IF EXISTS makes order safe).
DROP INDEX IF EXISTS bulk_job_items_payload_gin_idx;
DROP INDEX IF EXISTS bulk_job_items_row_order_idx;
DROP INDEX IF EXISTS bulk_job_items_retry_idx;

DROP INDEX IF EXISTS bulk_jobs_active_jobs_idx;
DROP INDEX IF EXISTS bulk_jobs_school_lookup_idx;

-- Drop tables in reverse dependency order (items before parent).
DROP TABLE IF EXISTS bulk_job_items;
DROP TABLE IF EXISTS bulk_jobs;

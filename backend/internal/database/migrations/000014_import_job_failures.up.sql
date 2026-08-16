-- Migration: 000014_import_job_failures
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: import_job_failures

CREATE TABLE IF NOT EXISTS import_job_failures (
    id            BIGSERIAL            PRIMARY KEY,
    import_job_id UUID                 NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    raw_payload   JSONB                NOT NULL,
    error_message TEXT                 NOT NULL,
    error_type    import_failure_type  NOT NULL DEFAULT 'DATABASE_CONSTRAINT',
    row_number    INT                  NULL,
    created_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_job_failures_job_id ON import_job_failures (import_job_id);



CREATE TRIGGER trg_import_job_failures_updated_at
    BEFORE UPDATE ON import_job_failures
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_failures.updated_at IS
    'Tracks when failure details were last modified.';

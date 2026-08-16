-- Migration: 000015_import_job_staging
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: import_job_staging

CREATE TABLE IF NOT EXISTS import_job_staging (
    id            UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID                  NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    tenant_id     UUID                  NOT NULL,
    school_id     UUID                  NOT NULL,
    row_number    INT                   NOT NULL,
    raw_data      JSONB                 NOT NULL,
    status        import_staging_status NOT NULL DEFAULT 'pending',
    processed_at  TIMESTAMPTZ           NULL,
    created_at    TIMESTAMPTZ           NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_job_staging_job_id ON import_job_staging (job_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_job_staging_job_row
    ON import_job_staging (job_id, row_number);

-- Composite key so cbc_students can reference (tenant_id, staging_row_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_job_staging_tenant
    ON import_job_staging (tenant_id, id);



CREATE TRIGGER trg_import_job_staging_updated_at
    BEFORE UPDATE ON import_job_staging
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_staging.updated_at IS
    'Tracks staging row processing: pending, succeeded, failed.';

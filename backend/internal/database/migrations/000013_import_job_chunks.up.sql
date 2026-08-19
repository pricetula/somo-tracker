-- Migration: 000013_import_job_chunks
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: import_job_chunks

CREATE TABLE IF NOT EXISTS import_job_chunks (
    id            UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID                NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    chunk_index   INT                 NOT NULL,
    status        import_chunk_status NOT NULL DEFAULT 'pending',
    row_start     INT                 NOT NULL DEFAULT 0,
    row_end       INT                 NOT NULL DEFAULT 0,
    claimed_at    TIMESTAMPTZ         NULL,
    completed_at  TIMESTAMPTZ         NULL,
    created_at    TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_import_job_chunks_job_chunk UNIQUE (job_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_import_job_chunks_job_id ON import_job_chunks (job_id);
CREATE INDEX IF NOT EXISTS idx_import_job_chunks_status ON import_job_chunks (job_id, status);



CREATE TRIGGER trg_import_job_chunks_updated_at
    BEFORE UPDATE ON import_job_chunks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_chunks.updated_at IS
    'Tracks chunk processing: pending, processing, completed, cancelled.';

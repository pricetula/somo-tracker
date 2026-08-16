-- Migration: 000012_import_jobs
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: import_jobs

CREATE TABLE IF NOT EXISTS import_jobs (
    id                   UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id            UUID              NOT NULL,
    job_type             import_job_type   NOT NULL,
    role                 user_role         NULL,
    created_by           UUID              NULL,
    status               import_job_status NOT NULL DEFAULT 'pending',
    total_records        INT               NOT NULL DEFAULT 0,
    processed_records    INT               NOT NULL DEFAULT 0,
    success_count        INT               NOT NULL DEFAULT 0,
    failed_count         INT               NOT NULL DEFAULT 0,
    idempotency_key      TEXT              NULL,
    payload_hash         TEXT              NULL,
    total_chunks         INT               NOT NULL DEFAULT 0,
    processed_chunks     INT               NOT NULL DEFAULT 0,
    metadata             JSONB             NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ       NULL,
    completed_at         TIMESTAMPTZ       NULL,
    last_progress_at    TIMESTAMPTZ       NULL,
    updated_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_import_jobs_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_import_jobs_tenant_created_by
        FOREIGN KEY (tenant_id, created_by)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL (created_by),
    CONSTRAINT chk_import_jobs_role_required_for_staff
        CHECK (
            (job_type IN ('STAFF_INVITE', 'PARENT_INVITE') AND role IS NOT NULL)
            OR (job_type NOT IN ('STAFF_INVITE', 'PARENT_INVITE'))
        )
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_id  ON import_jobs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_school_id  ON import_jobs (school_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_by ON import_jobs (created_by);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status     ON import_jobs (status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_jobs_tenant_idempotency
    ON import_jobs (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Composite key so invitations can reference (tenant_id, import_job_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_jobs_tenant ON import_jobs (tenant_id, id);

-- At most one active (processing or cancelling) import job per school at a time.
-- A new submission while one is active is rejected with import_already_in_progress.
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_jobs_one_active_per_school
    ON import_jobs (school_id)
    WHERE status IN ('processing'::import_job_status, 'cancelling'::import_job_status);



CREATE TRIGGER trg_import_jobs_updated_at
    BEFORE UPDATE ON import_jobs
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_jobs.updated_at IS
    'Tracks import lifecycle: pending, processing, completed, failed.';

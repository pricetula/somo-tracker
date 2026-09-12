-- Migration: 000012_bulk_job_ingestion
-- Purpose: Generic bulk-job ingestion system (reusable for admin invitations,
-- student import, exam results, staff import, etc.).
-- Dependencies:
--   - 000001_init_extensions (pgcrypto for gen_random_uuid())
--   - 000004_create_sis_schema (set_updated_at() trigger helper)
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: bulk_jobs
-- ============================================================================

CREATE TABLE bulk_jobs (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                              PRIMARY KEY,
    job_type            TEXT        NOT NULL,
    idempotency_key     TEXT        NOT NULL,
    school_id           UUID        NOT NULL    REFERENCES schools(id)
                                              ON DELETE CASCADE,
    tenant_id           UUID        NOT NULL    REFERENCES tenants(id)
                                              ON DELETE CASCADE,
    created_by          UUID        NOT NULL    REFERENCES users(id),
    status              TEXT        NOT NULL    DEFAULT 'QUEUED',
    total_records       INTEGER     NOT NULL,
    succeeded_count     INTEGER     NOT NULL    DEFAULT 0,
    failed_count        INTEGER     NOT NULL    DEFAULT 0,
    deferred_count      INTEGER     NOT NULL    DEFAULT 0,
    metadata            JSONB       NOT NULL    DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    -- Job types: extendable set. Start with ADMIN_INVITATION; future types
    -- (STUDENT_IMPORT, EXAM_RESULT, STAFF_IMPORT) can be added by altering
    -- the CHECK or moving to a lookup table.
    CONSTRAINT bulk_jobs_job_type_check CHECK (job_type IN ('ADMIN_INVITATION')),

    -- Status lifecycle.
    CONSTRAINT bulk_jobs_status_check CHECK (status IN
        ('QUEUED','PROCESSING','COMPLETED','COMPLETED_WITH_ERRORS','FAILED')),

    -- Idempotency guard per tenant.
    CONSTRAINT uq_bulk_jobs_idempotency_per_tenant UNIQUE (tenant_id, idempotency_key)
);

COMMENT ON TABLE bulk_jobs IS 'Generic bulk ingestion jobs. Reusable across admin invitations, student imports, exam results, staff imports, and future bulk operations.';
COMMENT ON COLUMN bulk_jobs.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN bulk_jobs.job_type IS 'Bulk operation category. Extendable via CHECK constraint or lookup table migration.';
COMMENT ON COLUMN bulk_jobs.idempotency_key IS 'Client-supplied idempotency token to prevent duplicate submissions.';
COMMENT ON COLUMN bulk_jobs.school_id IS 'School scope for the bulk operation.';
COMMENT ON COLUMN bulk_jobs.tenant_id IS 'Tenant isolation anchor.';
COMMENT ON COLUMN bulk_jobs.created_by IS 'User who submitted the bulk job.';
COMMENT ON COLUMN bulk_jobs.status IS 'Processing lifecycle: QUEUED, PROCESSING, COMPLETED, COMPLETED_WITH_ERRORS, FAILED.';
COMMENT ON COLUMN bulk_jobs.total_records IS 'Total rows submitted in the payload.';
COMMENT ON COLUMN bulk_jobs.succeeded_count IS 'Rows successfully processed.';
COMMENT ON COLUMN bulk_jobs.failed_count IS 'Rows that failed permanently.';
COMMENT ON COLUMN bulk_jobs.deferred_count IS 'Rows deferred (e.g. retry queued).';
COMMENT ON COLUMN bulk_jobs.metadata IS 'Job-type-specific summary info, e.g. source file name.';
COMMENT ON COLUMN bulk_jobs.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN bulk_jobs.updated_at IS 'UTC timestamp of last modification.';

-- Index: fast job history lookups per school.
CREATE INDEX bulk_jobs_school_lookup_idx ON bulk_jobs (tenant_id, school_id, job_type, created_at DESC);

-- Index: partial index for active job monitoring (ops/monitoring queries).
CREATE INDEX bulk_jobs_active_jobs_idx ON bulk_jobs (status)
    WHERE status IN ('QUEUED', 'PROCESSING');

-- ============================================================================
-- Section 2: bulk_job_items
-- ============================================================================

CREATE TABLE bulk_job_items (
    id                UUID        NOT NULL    DEFAULT gen_random_uuid()
                                              PRIMARY KEY,
    job_id            UUID        NOT NULL    REFERENCES bulk_jobs(id)
                                              ON DELETE CASCADE,
    row_index         INTEGER     NOT NULL,
    payload           JSONB       NOT NULL,
    result            JSONB,
    status            TEXT        NOT NULL    DEFAULT 'PENDING',
    attempt_count     INTEGER     NOT NULL    DEFAULT 0,
    last_error        TEXT,
    created_at        TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),

    CONSTRAINT bulk_job_items_status_check CHECK (status IN
        ('PENDING','PROCESSING','SUCCEEDED','FAILED','DEFERRED'))
);

COMMENT ON TABLE bulk_job_items IS 'Individual row-level records for each bulk ingestion job.';
COMMENT ON COLUMN bulk_job_items.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN bulk_job_items.job_id IS 'Parent bulk job reference.';
COMMENT ON COLUMN bulk_job_items.row_index IS 'Original array position in the submitted payload; maps errors back to client.';
COMMENT ON COLUMN bulk_job_items.payload IS 'Full submitted row, e.g. {"email":"...","full_name":"...","role":"..."}.';
COMMENT ON COLUMN bulk_job_items.result IS 'Output data on success, e.g. {"stytch_invite_id":"...","stytch_member_id":"..."}.';
COMMENT ON COLUMN bulk_job_items.status IS 'Per-row processing lifecycle.';
COMMENT ON COLUMN bulk_job_items.attempt_count IS 'Number of processing attempts made.';
COMMENT ON COLUMN bulk_job_items.last_error IS 'Human-readable or structured error message from the last failed attempt.';
COMMENT ON COLUMN bulk_job_items.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN bulk_job_items.updated_at IS 'UTC timestamp of last modification.';

-- Index: retry-failed queries, progress aggregation.
CREATE INDEX bulk_job_items_retry_idx ON bulk_job_items (job_id, status);

-- Index: ordered error reporting back to the client payload.
CREATE INDEX bulk_job_items_row_order_idx ON bulk_job_items (job_id, row_index);

-- GIN index for occasional ad-hoc JSONB payload queries.
CREATE INDEX bulk_job_items_payload_gin_idx ON bulk_job_items USING GIN (payload);

-- ============================================================================
-- Section 3: updated_at trigger functions (reuse existing set_updated_at)
-- ============================================================================

CREATE TRIGGER bulk_jobs_updated_at_trg
    BEFORE UPDATE ON bulk_jobs
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER bulk_job_items_updated_at_trg
    BEFORE UPDATE ON bulk_job_items
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

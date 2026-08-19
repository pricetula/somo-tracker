-- Migration: 000004_users
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: users

CREATE TABLE IF NOT EXISTS users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  VARCHAR(255) NOT NULL,
    tenant_id              UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name              VARCHAR(255) NOT NULL DEFAULT '',
    is_active              BOOLEAN      NOT NULL DEFAULT TRUE,
    external_auth_id       VARCHAR(255) UNIQUE,
    tsc_number             VARCHAR(15)  NULL,
    knec_panel_assessor_id VARCHAR(20)  NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_tenant UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email
    ON users (tenant_id, LOWER(email));

COMMENT ON INDEX idx_users_tenant_email IS
    'Per-tenant, case-insensitive unique constraint on email. Replaces the
     old global idx_users_email which prevented multi-tenant accounts and
     treated case variants as distinct. Added in 000003_fix.';
CREATE INDEX        IF NOT EXISTS idx_users_tenant ON users (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tsc_number
    ON users (tsc_number) WHERE tsc_number IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_knec_panel_assessor_id
    ON users (knec_panel_assessor_id) WHERE knec_panel_assessor_id IS NOT NULL;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN users.tsc_number IS
    'Teachers Service Commission registration number. Populated only for users
     with the TEACHER role. Required for TSC portal access and official deployment.';

COMMENT ON COLUMN users.knec_panel_assessor_id IS
    'Assigned ONLY to teachers formally appointed to KNEC national exam panels
     (KPSEA, KJSEA, KSSEA invigilation or marking). NOT required for classroom
     SBA delivery — all SBA uploads use the school knec_school_code, not teacher IDs.';

-- ---------------------------------------------------------------------------
-- SESSIONS
-- ---------------------------------------------------------------------------

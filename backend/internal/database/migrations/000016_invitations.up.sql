-- Migration: 000016_invitations
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: invitations

CREATE TABLE IF NOT EXISTS invitations (
    id                  UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id           UUID              NOT NULL,
    email               VARCHAR(255)      NOT NULL,
    role                user_role         NOT NULL,
    status              invitation_status NOT NULL DEFAULT 'pending',
    invited_by          UUID              NULL,
    token               TEXT              NOT NULL,
    token_hash          TEXT              NULL,
    expires_at          TIMESTAMPTZ       NOT NULL,
    accepted_at         TIMESTAMPTZ       NULL,
    full_name           VARCHAR(255)      NOT NULL,
    phone               VARCHAR(50)       NULL,
    registration_number VARCHAR(100)      NULL,
    stytch_member_id    VARCHAR(255)      NULL,
    import_job_id       UUID              NULL,
    error_message       TEXT              NULL,
    attempt_count       INT               NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invitations_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invitations_tenant_invited_by
        FOREIGN KEY (tenant_id, invited_by)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL (invited_by),
    CONSTRAINT fk_invitations_tenant_import_job
        FOREIGN KEY (tenant_id, import_job_id)
        REFERENCES import_jobs(tenant_id, id) ON DELETE SET NULL (import_job_id)
);

CREATE INDEX IF NOT EXISTS idx_invitations_tenant_id  ON invitations (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invitations_school_id  ON invitations (school_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email      ON invitations (email);
CREATE INDEX IF NOT EXISTS idx_invitations_status     ON invitations (status);
CREATE INDEX IF NOT EXISTS idx_invitations_import_job ON invitations (import_job_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_active_email
    ON invitations (tenant_id, school_id, email)
    WHERE status NOT IN ('expired', 'revoked');

-- Prevents race conditions where two concurrent chunks try to invite
-- the same email for the same school.
CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_school_email_pending
    ON invitations (school_id, email)
    WHERE status = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations (token_hash);

COMMENT ON COLUMN invitations.token IS
    'DEPRECATED — raw invitation token. New code should read token_hash instead.
     This column will be dropped in a future migration after the app is
     confirmed fully migrated to hash-based lookups. Do NOT write to this
     column in new code.';

COMMENT ON COLUMN invitations.token_hash IS
    'SHA-256 hash of the invitation token (hex-encoded). Backfilled from token
     column. Use this for token lookups instead of the raw token column.';



CREATE TRIGGER trg_invitations_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invitations.updated_at IS
    'Tracks status transitions (pending, accepted, expired, revoked).';

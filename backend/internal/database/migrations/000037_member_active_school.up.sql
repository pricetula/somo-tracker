-- Migration: 000037_member_active_school
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: member_active_school

CREATE TABLE IF NOT EXISTS member_active_school (
    user_id     UUID        NOT NULL,
    tenant_id   UUID        NOT NULL,
    school_id   UUID        NOT NULL,
    switched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id),

    CONSTRAINT fk_mas_tenant_user
        FOREIGN KEY (tenant_id, user_id)   REFERENCES users(tenant_id, id)            ON DELETE CASCADE,
    CONSTRAINT fk_mas_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id)      ON DELETE CASCADE,
    CONSTRAINT fk_mas_membership
        FOREIGN KEY (user_id, school_id)   REFERENCES memberships(user_id, school_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_mas_tenant_id ON member_active_school (tenant_id);

COMMENT ON TABLE member_active_school IS
    'Tracks the currently active school context for each user within a tenant.
     One row per user. Upsert on school switch. The chosen school_id is
     constrained to schools the user is an active member of via fk_mas_membership.';



CREATE TRIGGER trg_member_active_school_updated_at
    BEFORE UPDATE ON member_active_school
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN member_active_school.updated_at IS
    'Tracks active school context switches.';

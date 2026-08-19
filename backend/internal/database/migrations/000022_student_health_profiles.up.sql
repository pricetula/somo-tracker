-- Migration: 000022_student_health_profiles
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: student_health_profiles

CREATE TABLE IF NOT EXISTS student_health_profiles (
    id                     UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id             UUID    UNIQUE NOT NULL,
    blood_group            VARCHAR(5),
    allergies              TEXT[],
    chronic_conditions     TEXT[],
    emergency_instructions TEXT,
    logged_by              UUID        NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_student_health_profiles_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_student_health_profiles_tenant_logged_by
        FOREIGN KEY (tenant_id, logged_by)
        REFERENCES users(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_student_health_profiles_tenant_id ON student_health_profiles (tenant_id);



CREATE TRIGGER trg_student_health_profiles_updated_at
    BEFORE UPDATE ON student_health_profiles
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN student_health_profiles.updated_at IS
    'Tracks health profile updates (allergies, conditions, instructions).';

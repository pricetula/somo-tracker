-- Migration: 000043_grading_scale_profiles
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: grading_scale_profiles

CREATE TABLE IF NOT EXISTS grading_scale_profiles (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL,
    school_id  UUID         NOT NULL,
    name       VARCHAR(255) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_grading_scale_profiles_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_grading_scale_profiles_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_grading_scale_profiles_tenant
    ON grading_scale_profiles (tenant_id);
CREATE INDEX IF NOT EXISTS idx_grading_scale_profiles_school
    ON grading_scale_profiles (school_id);

DROP TRIGGER IF EXISTS trg_grading_scale_profiles_updated_at ON grading_scale_profiles;
CREATE TRIGGER trg_grading_scale_profiles_updated_at
    BEFORE UPDATE ON grading_scale_profiles
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE grading_scale_profiles IS
    'Directory of CBC grading scale profiles. Profiles define the translation
     from numeric percentages to CBC rubric levels (EE, ME, AE, BE). Once
     created, profile name and settings are read-only. To change a scale,
     create a new profile and mark the old one is_active = false.';

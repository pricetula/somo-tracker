-- Migration: 000044_grading_scale_ranges
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: grading_scale_ranges

CREATE TABLE IF NOT EXISTS grading_scale_ranges (
    id                        UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id                UUID                  NOT NULL,
    tenant_id                 UUID                  NOT NULL,
    performance_level         cbc_performance_level NOT NULL,
    min_percentage            NUMERIC(5,2)          NOT NULL CHECK (min_percentage >= 0 AND min_percentage <= 100),
    max_percentage            NUMERIC(5,2)          NOT NULL CHECK (max_percentage >= 0 AND max_percentage <= 100),
    default_percentage_mapping NUMERIC(5,2)          NULL,
    created_at                TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_range_bounds CHECK (max_percentage > min_percentage),
    CONSTRAINT uq_profile_level UNIQUE (profile_id, performance_level),
    CONSTRAINT excl_profile_range_no_overlap
        EXCLUDE USING gist (
            profile_id WITH =,
            numrange(min_percentage, max_percentage, '[]') WITH &&
        ),
    CONSTRAINT fk_grading_scale_ranges_tenant_profile
        FOREIGN KEY (tenant_id, profile_id)
        REFERENCES grading_scale_profiles(tenant_id, id) ON DELETE CASCADE
);

DROP TRIGGER IF EXISTS trg_grading_scale_ranges_updated_at ON grading_scale_ranges;
CREATE TRIGGER trg_grading_scale_ranges_updated_at
    BEFORE UPDATE ON grading_scale_ranges
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE INDEX IF NOT EXISTS idx_grading_scale_ranges_tenant
    ON grading_scale_ranges (tenant_id);

-- Write-once enforcement (000003 item 6a): block UPDATE/DELETE of ranges whose
-- profile is referenced by any assessment_sessions row.
CREATE OR REPLACE FUNCTION fn_block_grading_scale_range_modification()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM assessment_sessions
        WHERE grading_scale_profile_id = OLD.profile_id
        LIMIT 1
    ) THEN
        RAISE EXCEPTION 'Cannot modify or delete grading scale range (profile_id: %) — the grading profile is actively referenced by assessment sessions', OLD.profile_id
            USING ERRCODE = 'P0002';  -- assigned application-level error code
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_grading_scale_ranges_immutable ON grading_scale_ranges;
CREATE TRIGGER trg_grading_scale_ranges_immutable
    BEFORE UPDATE OR DELETE ON grading_scale_ranges
    FOR EACH ROW
    EXECUTE FUNCTION fn_block_grading_scale_range_modification();

COMMENT ON TRIGGER trg_grading_scale_ranges_immutable ON grading_scale_ranges IS
    'Enforces write-once semantics: prevents UPDATE or DELETE of grading scale
     ranges whose profile is referenced by any assessment_sessions row. Throws
     error code P0002 which the application can catch specifically.';

COMMENT ON TABLE grading_scale_ranges IS
    'Range definitions within a grading scale profile. The EXCLUDE constraint
     using numrange guarantees no overlapping percentage bands within the same
     profile. Rows are write-once — UPDATE and DELETE are blocked at the
     application layer once the profile is actively referenced by sessions.';

COMMENT ON COLUMN grading_scale_ranges.default_percentage_mapping IS
    'Optional midpoint value used as the default when converting a percentage
     to a performance level. If NULL, the system uses the midpoint of the range.
     Example: for range 80-100 → EE, default could be 90.';

COMMENT ON COLUMN grading_scale_ranges.tenant_id IS
    'Denormalised from grading_scale_profiles for RLS enforcement. Must match
     the tenant_id of the referenced profile.';

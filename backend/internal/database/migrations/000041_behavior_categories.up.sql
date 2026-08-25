-- Migration: 000040_behavior_categories
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: behavior_categories

CREATE TABLE IF NOT EXISTS behavior_categories (
    id               UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                NOT NULL,
    school_id        UUID                NOT NULL,
    name             VARCHAR(100)        NOT NULL,
    default_severity behavior_severity   NULL,
    is_active        BOOLEAN             NOT NULL DEFAULT true,
    category_type    behavior_category_type NOT NULL DEFAULT 'DISCIPLINARY',
    created_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_behavior_categories_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_behavior_category_name
        UNIQUE (tenant_id, school_id, name),
    -- Composite key so behavior_notes can reference (tenant_id, category_id)
    CONSTRAINT uq_behavior_categories_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_behavior_categories_tenant
    ON behavior_categories (tenant_id);
CREATE INDEX IF NOT EXISTS idx_behavior_categories_school
    ON behavior_categories (school_id);

COMMENT ON TABLE behavior_categories IS
    'School-configurable behavior/incident categories. Admins manage these
     per school rather than a fixed platform-wide enum. Categories are soft-
     deleted (is_active = false) to preserve historical behavior_notes.';

COMMENT ON COLUMN behavior_categories.category_type IS
    'Classification of the behavior category: COMMENDATION (positive/laudable
     behaviour), DISCIPLINARY (negative behaviour / infraction), or OTHER.
     Used by student_behavior_term_summaries to compute commendations_count
     and disciplinary_count. Defaults to DISCIPLINARY for existing categories.';



CREATE TRIGGER trg_behavior_categories_updated_at
    BEFORE UPDATE ON behavior_categories
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN behavior_categories.updated_at IS
    'Tracks category name changes and soft-delete toggles.';

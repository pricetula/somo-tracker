-- Migration: 000023_fee_categories
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: fee_categories

CREATE TABLE IF NOT EXISTS fee_categories (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL,
    school_id    UUID         NOT NULL,
    name         VARCHAR(150) NOT NULL,
    is_mandatory BOOLEAN      NOT NULL DEFAULT true,
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_fee_categories_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_fee_categories_tenant_school_name
        UNIQUE (tenant_id, school_id, name),
    -- Composite key so fee_templates / invoice_items can reference (tenant_id, fee_category_id)
    CONSTRAINT uq_fee_categories_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_fee_categories_tenant    ON fee_categories (tenant_id);
CREATE INDEX IF NOT EXISTS idx_fee_categories_school_id ON fee_categories (school_id);



CREATE TRIGGER trg_fee_categories_updated_at
    BEFORE UPDATE ON fee_categories
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN fee_categories.updated_at IS
    'Tracks fee category metadata changes.';

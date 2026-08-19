-- Migration: 000024_fee_templates
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: fee_templates

CREATE TABLE IF NOT EXISTS fee_templates (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID             NOT NULL,
    school_id        UUID             NOT NULL,
    academic_term_id UUID             NOT NULL,
    grade_level      cbc_grade_level  NOT NULL,
    fee_category_id  UUID             NOT NULL,
    amount           NUMERIC(12,2)    NOT NULL CHECK (amount >= 0),
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_fee_templates_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_fee_templates_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_fee_templates_tenant_fee_category
        FOREIGN KEY (tenant_id, fee_category_id)
        REFERENCES fee_categories(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_fee_template_rule
        UNIQUE (academic_term_id, grade_level, fee_category_id)
);

CREATE INDEX IF NOT EXISTS idx_fee_templates_tenant      ON fee_templates (tenant_id);
CREATE INDEX IF NOT EXISTS idx_fee_templates_school_term ON fee_templates (school_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_fee_templates_grade_level ON fee_templates (grade_level);



CREATE TRIGGER trg_fee_templates_updated_at
    BEFORE UPDATE ON fee_templates
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN fee_templates.updated_at IS
    'Tracks fee amount and configuration changes per term.';

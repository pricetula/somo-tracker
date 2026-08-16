-- Migration: 000031_performance_indicators
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: performance_indicators

CREATE TABLE IF NOT EXISTS performance_indicators (
    id             UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID     NOT NULL,
    sub_strand_id  UUID     NOT NULL,
    description    TEXT     NOT NULL,
    sequence_order SMALLINT NOT NULL DEFAULT 1,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_performance_indicators_tenant_sub_strand
        FOREIGN KEY (tenant_id, sub_strand_id)
        REFERENCES cbc_sub_strands(tenant_id, id) ON DELETE CASCADE,
    -- Composite key so student_assessment_outcome_grades can reference (tenant_id, performance_indicator_id)
    CONSTRAINT uq_performance_indicators_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_performance_indicators_sub_strand
    ON performance_indicators (sub_strand_id, sequence_order);
CREATE INDEX IF NOT EXISTS idx_performance_indicators_tenant
    ON performance_indicators (tenant_id);

COMMENT ON TABLE performance_indicators IS
    'Atomic CBC learning outcomes within a sub-strand, as defined in KICD
     curriculum designs. Leaf nodes of the hierarchy:
     Learning Area → Strand → Sub-Strand → Performance Indicator.';



CREATE TRIGGER trg_performance_indicators_updated_at
    BEFORE UPDATE ON performance_indicators
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN performance_indicators.updated_at IS
    'Tracks performance indicator revisions and re-sequencing.';

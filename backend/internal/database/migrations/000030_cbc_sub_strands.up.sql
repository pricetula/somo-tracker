-- Migration: 000030_cbc_sub_strands
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: cbc_sub_strands

CREATE TABLE IF NOT EXISTS cbc_sub_strands (
    id        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID         NOT NULL,
    strand_id UUID         NOT NULL,
    name      VARCHAR(255) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_sub_strands_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_cbc_sub_strands_tenant_strand
        FOREIGN KEY (tenant_id, strand_id)
        REFERENCES cbc_strands(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_sub_strands_strand_id ON cbc_sub_strands (strand_id);
CREATE INDEX IF NOT EXISTS idx_cbc_sub_strands_tenant ON cbc_sub_strands (tenant_id);



CREATE TRIGGER trg_cbc_sub_strands_updated_at
    BEFORE UPDATE ON cbc_sub_strands
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_sub_strands.updated_at IS
    'Tracks curriculum sub-strand revisions.';

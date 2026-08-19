-- Migration: 000019_cbc_student_parents
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: cbc_student_parents

CREATE TABLE IF NOT EXISTS cbc_student_parents (
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id   UUID        NOT NULL,
    parent_id    UUID        NOT NULL,
    relationship parent_relationship_type NULL, -- FATHER, MOTHER, GUARDIAN, OTHER
    is_primary   BOOLEAN     NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_cbc_student_parents PRIMARY KEY (tenant_id, student_id, parent_id),
    CONSTRAINT fk_cbc_student_parents_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_student_parents_tenant_parent
        FOREIGN KEY (tenant_id, parent_id)
        REFERENCES cbc_parents(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_junction_parent ON cbc_student_parents (parent_id);
CREATE INDEX IF NOT EXISTS idx_junction_tenant_student
    ON cbc_student_parents (tenant_id, student_id);

-- One primary parent per student (000003 item 5)
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_primary_parent_per_student
    ON cbc_student_parents (tenant_id, student_id) WHERE is_primary = true;

COMMENT ON COLUMN cbc_student_parents.relationship IS
    'Parent/guardian relationship to the student. Enum migrated from free-text
     in 000003_fix. Values: FATHER, MOTHER, GUARDIAN, OTHER.';



CREATE TRIGGER trg_cbc_student_parents_updated_at
    BEFORE UPDATE ON cbc_student_parents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_student_parents.updated_at IS
    'Tracks parent relationship and primary-contact changes.';

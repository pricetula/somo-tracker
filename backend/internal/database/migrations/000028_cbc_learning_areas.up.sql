-- Migration: 000028_cbc_learning_areas
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: cbc_learning_areas

CREATE TABLE IF NOT EXISTS cbc_learning_areas (
    id              UUID                NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID                NOT NULL,
    school_id       UUID                NOT NULL,
    name            VARCHAR(150)        NOT NULL,
    code            VARCHAR(50)         NOT NULL,
    education_level cbc_education_level NOT NULL,
    grade_level     cbc_grade_level     NOT NULL,

    PRIMARY KEY (id),
    CONSTRAINT fk_cbc_learning_areas_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_cbc_learning_areas_school_code_grade
        UNIQUE (tenant_id, school_id, code, grade_level),
    -- IMPROVE: expose (tenant_id, id) pair so downstream tables can composite-FK
    CONSTRAINT uq_cbc_learning_areas_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_tenant          ON cbc_learning_areas (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_school_id       ON cbc_learning_areas (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_education_level ON cbc_learning_areas (education_level);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_grade_level     ON cbc_learning_areas (grade_level);

COMMENT ON COLUMN cbc_learning_areas.education_level IS
    'The CBC tier this learning area belongs to, per KICD curriculum structure.
     Determines applicable KNEC assessment instruments and portal upload eligibility.';

COMMENT ON COLUMN cbc_learning_areas.code IS
    'Short KICD-defined code for this learning area, e.g. MATH, ENG, KISW,
     INT_SCI, PRE_TECH, SOC_STD. Unique within a school per grade level.';

COMMENT ON COLUMN cbc_learning_areas.grade_level IS
    'The specific CBC grade this learning area instance is taught in.
     Each grade has its own set of learning areas per KNEC/KICD curriculum.
     Combined with code to uniquely identify a learning area within a school.';

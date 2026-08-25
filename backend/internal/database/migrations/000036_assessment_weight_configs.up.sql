-- Migration: 000035_assessment_weight_configs
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: assessment_weight_configs

CREATE TABLE IF NOT EXISTS assessment_weight_configs (
    id                   UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    grade_level          cbc_grade_level NOT NULL,
    assessment_type_code VARCHAR(50)    NOT NULL,
    target_exam          VARCHAR(20)    NOT NULL,
    weight_percent       NUMERIC(5,2)   NOT NULL CHECK (weight_percent > 0 AND weight_percent <= 100),
    effective_from       INTEGER        NOT NULL,
    notes                TEXT           NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_assessment_weight_config
        UNIQUE (grade_level, assessment_type_code, target_exam, effective_from)
);

COMMENT ON TABLE assessment_weight_configs IS
    'Official KNEC assessment weighting formulas per grade level. These are
     nationally mandated and do not vary per school. Schema changes would be
     required if per-school overrides are ever needed.';

COMMENT ON COLUMN assessment_weight_configs.assessment_type_code IS
    'KNEC assessment type identifier, e.g. KNEC_SBA_Project, National_KPSEA, National_KJSEA.';
COMMENT ON COLUMN assessment_weight_configs.target_exam IS
    'The target national exam this weight contributes to: KPSEA, KJSEA, or KSSEA.';
COMMENT ON COLUMN assessment_weight_configs.weight_percent IS
    'Percentage contribution of this assessment component towards the target exam placement.';
COMMENT ON COLUMN assessment_weight_configs.effective_from IS
    'Academic year from which this weighting formula is effective.';

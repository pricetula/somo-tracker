-- +goose Up

-- Education system levels: concrete mapping of years per level within a specific education system.
CREATE TABLE education_system_levels (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    year_count INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (education_system_id, education_level_id)
);

CREATE INDEX idx_education_system_levels_system ON education_system_levels (education_system_id);
CREATE INDEX idx_education_system_levels_level ON education_system_levels (education_level_id);

-- Education system grading scales: default grading scale bands for an education system.
CREATE TABLE education_system_grading_scales (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_default BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_education_system_grading_scales_system ON education_system_grading_scales (education_system_id);

-- Education system grading bands: individual grade bands within a scale.
-- Supports both percentage-based and rubric-based (competency) grading.
CREATE TABLE education_system_grading_bands (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    grading_scale_id UUID NOT NULL REFERENCES education_system_grading_scales(id) ON DELETE CASCADE,
    grade TEXT NOT NULL,
    scale_type TEXT NOT NULL DEFAULT 'percentage' CHECK (scale_type IN ('percentage', 'rubric')),
    rubric_level TEXT,
    min_score DECIMAL NOT NULL,
    max_score DECIMAL NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rubric_level_check CHECK (
        (scale_type = 'percentage' AND rubric_level IS NULL)
        OR (scale_type = 'rubric' AND rubric_level IS NOT NULL)
    ),
    CONSTRAINT chk_score_range CHECK (min_score <= max_score)
);

CREATE INDEX idx_education_system_grading_bands_scale ON education_system_grading_bands (grading_scale_id);

-- +goose Down

DROP TABLE IF EXISTS education_system_grading_bands;

DROP TABLE IF EXISTS education_system_grading_scales;

DROP TABLE IF EXISTS education_system_levels;

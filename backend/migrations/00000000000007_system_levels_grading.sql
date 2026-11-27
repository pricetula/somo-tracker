-- +goose Up

CREATE TYPE grading_scale_type AS ENUM ('percentage', 'rubric');

-- Grading scales: grading scale definition, scoped to an education system and optionally a school.
-- When school_id is NULL, it is a system-level default; when set, it is a school-specific override.
CREATE TABLE grading_scales (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    school_id UUID REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_default BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_grading_scales_system ON grading_scales (education_system_id);
CREATE INDEX idx_grading_scales_school ON grading_scales (school_id);

-- Grading bands: individual grade bands within a scale.
CREATE TABLE grading_bands (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    grading_scale_id UUID NOT NULL REFERENCES grading_scales(id) ON DELETE CASCADE,
    grade TEXT NOT NULL,
    scale_type grading_scale_type NOT NULL DEFAULT 'percentage',
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

CREATE INDEX idx_grading_bands_scale ON grading_bands (grading_scale_id);

-- +goose Down

DROP TABLE IF EXISTS grading_bands;

DROP TABLE IF EXISTS grading_scales;

DROP TYPE IF EXISTS grading_scale_type;

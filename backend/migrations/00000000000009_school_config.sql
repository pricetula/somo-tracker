-- +goose Up

-- School grading scales: school-specific override of the education system default grading scale.
CREATE TABLE school_grading_scales (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_active BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_school_grading_scales_school ON school_grading_scales (school_id);

-- School grading bands: normalized grade bands for a school's active scale.
-- Supports both percentage-based and rubric-based (competency) grading.
CREATE TABLE school_grading_bands (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_grading_scale_id UUID NOT NULL REFERENCES school_grading_scales(id) ON DELETE CASCADE,
    grade TEXT NOT NULL,
    scale_type TEXT NOT NULL DEFAULT 'percentage' CHECK (scale_type IN ('percentage', 'rubric')),
    rubric_level TEXT,
    min_score DECIMAL NOT NULL,
    max_score DECIMAL NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_school_rubric_level_check CHECK (
        (scale_type = 'percentage' AND rubric_level IS NULL)
        OR (scale_type = 'rubric' AND rubric_level IS NOT NULL)
    ),
    CONSTRAINT chk_school_score_range CHECK (min_score <= max_score)
);

CREATE INDEX idx_school_grading_bands_scale ON school_grading_bands (school_grading_scale_id);

-- School subjects: subjects available at this school, per level.
CREATE TABLE school_subjects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    subject_code TEXT NOT NULL,
    subject_name TEXT NOT NULL,
    is_compulsory BOOL NOT NULL DEFAULT TRUE,
    is_active BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, education_level_id, subject_code)
);

CREATE INDEX idx_school_subjects_school ON school_subjects (school_id);
CREATE INDEX idx_school_subjects_level ON school_subjects (education_level_id);

-- School academic years.
CREATE TABLE school_academic_years (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    year_label TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_school_academic_years_school ON school_academic_years (school_id);

-- School academic terms.
CREATE TABLE school_academic_terms (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES school_academic_years(id) ON DELETE CASCADE,
    term_name TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, academic_year_id, sort_order)
);

CREATE INDEX idx_school_academic_terms_school ON school_academic_terms (school_id);
CREATE INDEX idx_school_academic_terms_year ON school_academic_terms (academic_year_id);

-- School grade years: concrete grade/year names within a level for this school.
CREATE TABLE school_grade_years (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    grade_name TEXT NOT NULL,
    grade_number INT NOT NULL,
    is_active BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, education_level_id, grade_number)
);

CREATE INDEX idx_school_grade_years_school ON school_grade_years (school_id);
CREATE INDEX idx_school_grade_years_level ON school_grade_years (education_level_id);

-- +goose Down

DROP TABLE IF EXISTS school_grade_years;

DROP TABLE IF EXISTS school_academic_terms;

DROP TABLE IF EXISTS school_academic_years;

DROP TABLE IF EXISTS school_subjects;

DROP TABLE IF EXISTS school_grading_bands;

DROP TABLE IF EXISTS school_grading_scales;

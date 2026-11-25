-- +goose Up

-- Education system subjects: default subject catalog per education system per level.
CREATE TABLE education_system_subjects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    subject_code TEXT NOT NULL,
    subject_name TEXT NOT NULL,
    is_compulsory_default BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (education_system_id, education_level_id, subject_code)
);

CREATE INDEX idx_education_system_subjects_system ON education_system_subjects (education_system_id);
CREATE INDEX idx_education_system_subjects_level ON education_system_subjects (education_level_id);

-- Education system term structures: default term names and count per education system.
CREATE TABLE education_system_term_structures (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    term_name TEXT NOT NULL,
    term_sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (education_system_id, term_sort_order),
    UNIQUE (education_system_id, term_name)
);

CREATE INDEX idx_education_system_term_structures_system ON education_system_term_structures (education_system_id);

-- +goose Down

DROP TABLE IF EXISTS education_system_term_structures;

DROP TABLE IF EXISTS education_system_subjects;

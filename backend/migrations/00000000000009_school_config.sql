-- +goose Up

-- School subjects: subjects available at this school, per level.
CREATE TABLE school_subjects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    is_compulsory BOOL NOT NULL DEFAULT TRUE,
    is_active BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, education_level_id, code)
);

CREATE INDEX idx_school_subjects_school ON school_subjects (school_id);
CREATE INDEX idx_school_subjects_level ON school_subjects (education_level_id);

-- Academic years.
CREATE TABLE academic_years (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    year_label TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_academic_years_school ON academic_years (school_id);

-- Academic terms.
CREATE TABLE academic_terms (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    term_name TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, academic_year_id, sort_order)
);

CREATE INDEX idx_academic_terms_school ON academic_terms (school_id);
CREATE INDEX idx_academic_terms_year ON academic_terms (academic_year_id);

-- Grade years: concrete grade/year names within a level for this school.
CREATE TABLE grade_years (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    grade_name TEXT NOT NULL,
    grade_number INT NOT NULL,
    is_active BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, education_level_id, grade_number),
    UNIQUE (id, school_id)
);

CREATE INDEX idx_grade_years_school ON grade_years (school_id);
CREATE INDEX idx_grade_years_level ON grade_years (education_level_id);

-- Streams: stream designations within a school (e.g. 'A', 'B', 'Yellow', 'Blue').
CREATE TABLE streams (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name),
    UNIQUE (id, school_id)
);

CREATE INDEX idx_streams_school ON streams (school_id);

-- Classes: concrete classes combining a grade year and stream (e.g. 'Grade 1A', 'Class 4Yellow').
-- Each class belongs to an education level through its grade year.
CREATE TABLE classes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    grade_year_id UUID NOT NULL REFERENCES grade_years(id) ON DELETE CASCADE,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    class_name TEXT NOT NULL,
    is_active BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (grade_year_id, stream_id),
    UNIQUE (id, school_id),
    FOREIGN KEY (grade_year_id, school_id)
        REFERENCES grade_years(id, school_id),
    FOREIGN KEY (stream_id, school_id)
        REFERENCES streams(id, school_id)
);

CREATE INDEX idx_classes_school ON classes (school_id);
CREATE INDEX idx_classes_grade_year ON classes (grade_year_id);
CREATE INDEX idx_classes_stream ON classes (stream_id);

-- +goose Down

DROP TABLE IF EXISTS classes;

DROP TABLE IF EXISTS streams;

DROP TABLE IF EXISTS grade_years;

DROP TABLE IF EXISTS academic_terms;

DROP TABLE IF EXISTS academic_years;

DROP TABLE IF EXISTS school_subjects;

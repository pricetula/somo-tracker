-- +goose Up

-- Countries: registry of active countries.
CREATE TABLE countries (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_countries_code ON countries (code);

-- Education levels: canonical levels shared across all systems.
CREATE TABLE education_levels (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL UNIQUE,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Education systems: one row per country representing its current system.
CREATE TABLE education_systems (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    years_primary INT NOT NULL,
    years_lower_secondary INT NOT NULL,
    years_upper_secondary INT NOT NULL,
    default_language TEXT NOT NULL DEFAULT 'English',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_education_systems_country ON education_systems (country_id);

-- Add foreign key constraints from schools to countries and education_systems.
ALTER TABLE schools
    ADD CONSTRAINT fk_schools_country
    FOREIGN KEY (country_id)
    REFERENCES countries(id);

ALTER TABLE schools
    ADD CONSTRAINT fk_schools_education_system
    FOREIGN KEY (education_system_id)
    REFERENCES education_systems(id);

-- +goose Down

ALTER TABLE schools DROP CONSTRAINT IF EXISTS fk_schools_education_system;

ALTER TABLE schools DROP CONSTRAINT IF EXISTS fk_schools_country;

DROP TABLE IF EXISTS education_systems;

DROP TABLE IF EXISTS education_levels;

DROP TABLE IF EXISTS countries;

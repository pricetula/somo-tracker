-- +goose Up

ALTER TABLE schools
    ADD COLUMN country_id UUID;

ALTER TABLE schools
    ADD COLUMN education_system_id UUID;

ALTER TABLE schools
    ADD COLUMN school_type TEXT;

ALTER TABLE schools
    ADD COLUMN language_of_instruction TEXT DEFAULT 'English';

ALTER TABLE schools
    ADD COLUMN registration_number TEXT;

-- +goose Down

ALTER TABLE schools DROP COLUMN registration_number;

ALTER TABLE schools DROP COLUMN language_of_instruction;

ALTER TABLE schools DROP COLUMN education_system_id;

ALTER TABLE schools DROP COLUMN country_id;

ALTER TABLE schools DROP COLUMN school_type;

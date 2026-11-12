-- +goose Up
-- Reference data (countries, education_systems, class_levels) has RLS enabled
-- and forced in migration 06 but no policies, making it completely inaccessible.
-- Add SELECT policies so all authenticated sessions can read global reference data.
CREATE POLICY countries_read_all ON countries USING (true);
CREATE POLICY education_systems_read_all ON education_systems USING (true);
CREATE POLICY class_levels_read_all ON class_levels USING (true);

-- +goose Down
DROP POLICY IF EXISTS class_levels_read_all ON class_levels;
DROP POLICY IF EXISTS education_systems_read_all ON education_systems;
DROP POLICY IF EXISTS countries_read_all ON countries;

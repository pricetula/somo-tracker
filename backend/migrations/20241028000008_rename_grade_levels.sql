-- +goose Up
ALTER TABLE grade_levels RENAME TO class_levels;

-- +goose Down
ALTER TABLE class_levels RENAME TO grade_levels;

-- Migration: 000020_replace_class_rooms_stream_with_stream_id (down)
ALTER TABLE class_rooms DROP COLUMN IF EXISTS stream_id;
ALTER TABLE class_rooms ADD COLUMN stream VARCHAR(64) DEFAULT NULL;
DROP INDEX IF EXISTS class_rooms_stream_id_idx;
DROP INDEX IF EXISTS class_rooms_school_year_grade_stream_uniq;
CREATE UNIQUE INDEX class_rooms_school_year_grade_stream_uniq
    ON class_rooms (school_id, academic_year_id, grade_level_id, NULLIF(stream, ''));
CREATE INDEX class_rooms_name_idx ON class_rooms (name);

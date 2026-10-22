-- Migration: 000020_replace_class_rooms_stream_with_stream_id
-- Purpose: Change class_rooms.stream (VARCHAR name) to stream_id (UUID FK to streams).
-- Dependencies:
--   - 000010_create_streams (streams table must exist)
--   - 000007_create_class_rooms_and_enrollments

ALTER TABLE class_rooms DROP COLUMN IF EXISTS stream;
ALTER TABLE class_rooms ADD COLUMN stream_id UUID REFERENCES streams(id) ON DELETE SET NULL;

CREATE INDEX class_rooms_stream_id_idx ON class_rooms (stream_id);

DROP INDEX IF EXISTS class_rooms_school_year_grade_stream_uniq;
CREATE UNIQUE INDEX class_rooms_school_year_grade_stream_uniq
    ON class_rooms (school_id, academic_year_id, grade_level_id, stream_id);

COMMENT ON COLUMN class_rooms.stream_id IS 'FK to streams(id). Optional stream for this classroom.';

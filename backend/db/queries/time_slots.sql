-- name: CreateTimeSlot :one
INSERT INTO time_slots (id, school_id, timetable_template_id, name, start_time, end_time, sequence_index, is_instructional)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: ListTimeSlotsByTemplate :many
SELECT id, school_id, timetable_template_id, name, start_time, end_time, sequence_index, is_instructional, created_at, updated_at
FROM time_slots
WHERE timetable_template_id = $1
ORDER BY sequence_index ASC;

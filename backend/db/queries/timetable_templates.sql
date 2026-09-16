-- name: CreateTimetableTemplate :one
INSERT INTO timetable_templates (id, school_id, name, description)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: ListTimetableTemplatesBySchool :many
SELECT id, school_id, name, description, created_at, updated_at
FROM timetable_templates
WHERE school_id = $1
ORDER BY created_at DESC;

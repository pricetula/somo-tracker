-- name: ListStudentNotes :many
SELECT id, school_id, student_id, author_id, content, created_at, updated_at
FROM student_notes
WHERE school_id = $1 AND student_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CreateStudentNote :one
INSERT INTO student_notes (school_id, student_id, author_id, content)
VALUES ($1, $2, $3, $4)
RETURNING id, school_id, student_id, author_id, content, created_at, updated_at;

-- name: UpdateStudentNote :one
UPDATE student_notes
SET content = $1, updated_at = NOW()
WHERE id = $2 AND school_id = $3
RETURNING id, school_id, student_id, author_id, content, created_at, updated_at;

-- name: DeleteStudentNote :exec
DELETE FROM student_notes WHERE id = $1 AND school_id = $2;

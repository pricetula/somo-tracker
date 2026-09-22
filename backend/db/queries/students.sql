-- name: ListStudents :many
SELECT
    student_id,
    school_id,
    admission_number,
    full_name,
    date_of_birth,
    gender,
    metadata,
    created_at,
    updated_at
FROM students
WHERE school_id = $1
  AND (
    $2::text = '' OR
    full_name ILIKE '%' || $2 || '%' OR
    admission_number ILIKE '%' || $2 || '%'
  )
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountStudents :one
SELECT COUNT(*)
FROM students
WHERE school_id = $1
  AND (
    $2::text = '' OR
    full_name ILIKE '%' || $2 || '%' OR
    admission_number ILIKE '%' || $2 || '%'
  );

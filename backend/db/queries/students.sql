-- name: ListStudents :many
SELECT
    s.student_id,
    s.school_id,
    s.admission_number,
    s.full_name,
    s.date_of_birth,
    s.gender,
    s.metadata,
    s.created_at,
    s.updated_at,
    c.id AS class_id,
    c.name AS class_name
FROM students s
LEFT JOIN LATERAL (
    SELECT sce.class_room_id
    FROM student_class_enrollments sce
    WHERE sce.student_id = s.student_id
      AND sce.status = 'ACTIVE'
    ORDER BY sce.enrolled_at DESC
    LIMIT 1
) e ON true
LEFT JOIN class_rooms c ON c.id = e.class_room_id
WHERE s.school_id = $1
  AND (
    $2::text = '' OR
    s.full_name ILIKE '%' || $2 || '%' OR
    s.admission_number ILIKE '%' || $2 || '%'
  )
  AND ($5::uuid IS NULL OR c.id = $5)
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountStudents :one
SELECT COUNT(*)
FROM students s
LEFT JOIN LATERAL (
    SELECT sce.class_room_id
    FROM student_class_enrollments sce
    WHERE sce.student_id = s.student_id
      AND sce.status = 'ACTIVE'
    ORDER BY sce.enrolled_at DESC
    LIMIT 1
) e ON true
LEFT JOIN class_rooms c ON c.id = e.class_room_id
WHERE s.school_id = $1
  AND (
    $2::text = '' OR
    s.full_name ILIKE '%' || $2 || '%' OR
    s.admission_number ILIKE '%' || $2 || '%'
  )
  AND ($3::uuid IS NULL OR c.id = $3);

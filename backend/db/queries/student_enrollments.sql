-- name: CreateEnrollment :one
INSERT INTO student_class_enrollments (
    school_id, student_id, class_room_id, academic_year_id, academic_term_id,
    status, enrolled_at, metadata
) VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6, $7)
RETURNING *;

-- name: ListEnrollments :many
SELECT sce.*,
    cr.name as class_name, cr.grade_level_id, gl.local_label as grade,
    cr.stream, s.admission_number, s.full_name as student_name
FROM student_class_enrollments sce
JOIN class_rooms cr ON cr.id = sce.class_room_id
JOIN grade_levels gl ON gl.id = cr.grade_level_id
JOIN students s ON s.student_id = sce.student_id
WHERE sce.school_id = $1
  AND ($2::uuid IS NULL OR sce.class_room_id = $2)
  AND ($3::uuid IS NULL OR sce.student_id = $3)
  AND ($4::text IS NULL OR $4 = '' OR sce.status::text = $4)
ORDER BY sce.enrolled_at DESC
LIMIT $5 OFFSET $6;

-- name: CountEnrollments :one
SELECT COUNT(*) FROM student_class_enrollments
WHERE school_id = $1
  AND ($2::uuid IS NULL OR class_room_id = $2)
  AND ($3::uuid IS NULL OR student_id = $3)
  AND ($4::text IS NULL OR $4 = '' OR status::text = $4);

-- name: UpdateEnrollment :one
UPDATE student_class_enrollments
SET class_room_id = COALESCE($2, class_room_id),
    status = COALESCE($3, status),
    metadata = COALESCE($4, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEnrollment :exec
DELETE FROM student_class_enrollments WHERE id = $1;

-- name: GetActiveEnrollmentByStudent :one
SELECT * FROM student_class_enrollments
WHERE student_id = $1 AND academic_year_id = $2 AND status = 'ACTIVE';

-- name: ListUnassignedStudents :many
SELECT s.student_id, s.full_name, s.admission_number
FROM students s
WHERE s.school_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM student_class_enrollments sce
    WHERE sce.student_id = s.student_id
      AND sce.academic_year_id = $2
      AND sce.status = 'ACTIVE'
  )
ORDER BY s.full_name
LIMIT $3 OFFSET $4;

-- name: CountUnassignedStudents :one
SELECT COUNT(*)
FROM students s
WHERE s.school_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM student_class_enrollments sce
    WHERE sce.student_id = s.student_id
      AND sce.academic_year_id = $2
      AND sce.status = 'ACTIVE'
  );

-- name: GetClassRoom :one
SELECT id, academic_year_id, school_id FROM class_rooms WHERE id = $1;

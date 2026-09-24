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

-- name: DeleteStudents :exec
DELETE FROM students
WHERE school_id = $1
  AND student_id = ANY($2::uuid[]);

-- name: GetStudentSummary :one
SELECT
  gc.total_count AS total_students,
  gc.male_count AS male_count,
  gc.female_count AS female_count,
  gc.total_count - (SELECT COUNT(DISTINCT student_id)
                     FROM student_class_enrollments
                     WHERE student_class_enrollments.school_id = $1
                       AND academic_term_id = $2
                       AND status = 'ACTIVE') AS unassigned_count,
  gc.total_count - (SELECT COUNT(DISTINCT gsl.student_id)
                     FROM guardian_student_links gsl
                     JOIN school_memberships sm ON sm.id = gsl.school_membership_id
                     WHERE sm.school_id = $1) AS unlinked_guardians_count
FROM student_gender_counts gc
WHERE gc.school_id = $1;

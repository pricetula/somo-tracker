-- name: GetCurrentAcademicYearBySchool :one
SELECT id FROM academic_years WHERE school_id = $1 ORDER BY start_date DESC LIMIT 1;

-- name: CountClassRoomsBySchoolAndYear :one
SELECT COUNT(*) FROM class_rooms WHERE school_id = $1 AND academic_year_id = $2;

-- name: ListClassRoomsBySchoolAndYear :many
SELECT cr.id, cr.name, gl.local_label as grade, COALESCE(cr.stream,'') as stream
FROM class_rooms cr
JOIN grade_levels gl ON gl.id = cr.grade_level_id
WHERE cr.school_id = $1 AND cr.academic_year_id = $2;

-- name: ListClassEnrollmentsSummaryBySchoolAndYear :many
SELECT 
  cr.id as class_id,
  cr.name as class_name,
  gl.local_label as grade,
  COALESCE(cr.stream,'') as stream,
  COUNT(sce.id) FILTER (WHERE sce.status = 'ACTIVE') as student_count
FROM class_rooms cr
JOIN grade_levels gl ON gl.id = cr.grade_level_id
LEFT JOIN student_class_enrollments sce 
  ON sce.class_room_id = cr.id 
  AND sce.academic_year_id = $2 
  AND sce.status = 'ACTIVE'
WHERE cr.school_id = $1 AND cr.academic_year_id = $2
GROUP BY cr.id, cr.name, gl.local_label, cr.stream;

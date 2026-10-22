-- name: ListStudentTermAttendanceStats :many
SELECT school_id, academic_term_id, student_id, present_count, absent_count, late_count, attendance_rate, last_updated_at
FROM student_term_attendance_stats
WHERE school_id = $1 AND academic_term_id = $2
ORDER BY attendance_rate DESC
LIMIT $3 OFFSET $4;

-- name: ListStudentTermAttendanceStatsByStudent :one
SELECT school_id, academic_term_id, student_id, present_count, absent_count, late_count, attendance_rate, last_updated_at
FROM student_term_attendance_stats
WHERE school_id = $1 AND academic_term_id = $2 AND student_id = $3;

-- name: ListClassTermAttendanceStats :many
SELECT stas.student_id, s.full_name, s.admission_number, stas.present_count, stas.absent_count, stas.late_count, stas.attendance_rate
FROM student_term_attendance_stats stas
JOIN students s ON s.student_id = stas.student_id
JOIN student_class_enrollments sce ON sce.student_id = stas.student_id AND sce.academic_term_id = stas.academic_term_id AND sce.class_room_id = $3
WHERE stas.school_id = $1 AND stas.academic_term_id = $2
ORDER BY stas.attendance_rate DESC
LIMIT $4 OFFSET $5;

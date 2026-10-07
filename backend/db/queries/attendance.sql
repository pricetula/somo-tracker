-- name: CreateTimetableAttendance :one
INSERT INTO timetable_attendance (school_id, student_id, class_timetable_slot_id, attendance_date, status, remarks, recorded_by_membership_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTimetableAttendanceBySlotAndDate :many
SELECT id, school_id, student_id, class_timetable_slot_id, attendance_date, status, remarks, recorded_by_membership_id, created_at, updated_at
FROM timetable_attendance
WHERE class_timetable_slot_id = $1 AND attendance_date = $2;

-- name: GetTimetableAttendanceByStudentAndDate :many
SELECT id, school_id, student_id, class_timetable_slot_id, attendance_date, status, remarks, recorded_by_membership_id, created_at, updated_at
FROM timetable_attendance
WHERE student_id = $1 AND attendance_date = $2;
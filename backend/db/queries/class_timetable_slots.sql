-- name: CreateClassTimetableSlot :one
INSERT INTO class_timetable_slots (school_id, class_room_id, academic_term_id, day_of_week, time_slot_id, subject_id, teacher_membership_id, room_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: DeleteClassTimetableSlot :exec
DELETE FROM class_timetable_slots WHERE id = $1;

-- name: GetClassTimetableSlotsByTemplate :many
SELECT
  cts.id,
  cts.school_id,
  cts.class_room_id,
  cts.academic_term_id,
  cts.day_of_week,
  cts.time_slot_id,
  cts.subject_id,
  cts.teacher_membership_id,
  cts.room_id,
  cts.created_at,
  cts.updated_at,
  ts.name AS time_slot_name,
  ts.start_time,
  ts.end_time,
  ts.sequence_index,
  ts.is_instructional
FROM class_timetable_slots cts
JOIN time_slots ts ON ts.id = cts.time_slot_id
WHERE cts.class_room_id = $1
  AND ts.timetable_template_id = $2
ORDER BY cts.day_of_week ASC, ts.sequence_index ASC;

-- name: GetClassTimetableSlotsByTemplateWithDetails :many
SELECT
  cts.id,
  cts.school_id,
  cts.class_room_id,
  cts.academic_term_id,
  cts.day_of_week,
  cts.time_slot_id,
  cts.subject_id,
  cts.teacher_membership_id,
  cts.room_id,
  cts.created_at,
  cts.updated_at,
  cr.name AS class_name,
  cr.stream AS class_stream,
  gl.local_label AS grade_name,
  ts.name AS time_slot_name,
  ts.start_time,
  ts.end_time,
  ts.sequence_index,
  ts.is_instructional,
  s.name AS subject_name,
  u.full_name AS teacher_name,
  u.email AS teacher_email,
  r.name AS room_name
FROM class_timetable_slots cts
JOIN time_slots ts ON ts.id = cts.time_slot_id
JOIN class_rooms cr ON cr.id = cts.class_room_id
LEFT JOIN grade_levels gl ON gl.id = cr.grade_level_id
LEFT JOIN subjects s ON s.id = cts.subject_id
LEFT JOIN school_memberships sm ON sm.id = cts.teacher_membership_id
LEFT JOIN users u ON u.id = sm.user_id
LEFT JOIN rooms r ON r.id = cts.room_id
WHERE cts.class_room_id = $1
  AND ts.timetable_template_id = $2
ORDER BY cts.day_of_week ASC, ts.sequence_index ASC;


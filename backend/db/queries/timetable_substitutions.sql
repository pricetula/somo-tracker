-- name: CreateTimetableSubstitution :one
INSERT INTO timetable_substitutions (school_id, class_timetable_slot_id, substitution_date, original_teacher_membership_id, substitute_teacher_membership_id, status, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, school_id, class_timetable_slot_id, substitution_date, original_teacher_membership_id, substitute_teacher_membership_id, status, reason, created_at, updated_at;

-- name: UpdateTimetableSubstitution :exec
UPDATE timetable_substitutions
SET substitute_teacher_membership_id = COALESCE($3, substitute_teacher_membership_id),
    status = COALESCE($4, status),
    reason = COALESCE($5, reason),
    updated_at = NOW()
WHERE id = $1 AND school_id = $2;

-- name: DeleteTimetableSubstitution :exec
DELETE FROM timetable_substitutions WHERE id = $1;

-- name: GetTimetableSubstitution :one
SELECT id, school_id, class_timetable_slot_id, substitution_date, original_teacher_membership_id, substitute_teacher_membership_id, status, reason, created_at, updated_at
FROM timetable_substitutions
WHERE id = $1 AND school_id = $2;

-- name: ListTimetableSubstitutions :many
SELECT ts.*,
    cts.class_room_id, cr.name as class_name,
    s.name as subject_name,
    tm.user_id as original_teacher_id, u.full_name as original_teacher_name,
    stm.user_id as substitute_teacher_id, su.full_name as substitute_teacher_name,
    tsl.name as time_slot_name, tsl.start_time, tsl.end_time
FROM timetable_substitutions ts
JOIN class_timetable_slots cts ON cts.id = ts.class_timetable_slot_id
JOIN class_rooms cr ON cr.id = cts.class_room_id
JOIN subjects s ON s.id = cts.subject_id
JOIN school_memberships tm ON tm.id = cts.teacher_membership_id
JOIN users u ON u.id = tm.user_id
LEFT JOIN school_memberships stm ON stm.id = ts.substitute_teacher_membership_id
LEFT JOIN users su ON su.id = stm.user_id
JOIN time_slots tsl ON tsl.id = cts.time_slot_id
WHERE ts.school_id = $1
  AND ($2::date IS NULL OR ts.substitution_date >= $2)
  AND ($3::date IS NULL OR ts.substitution_date <= $3)
  AND ($4::text IS NULL OR ts.status = $4)
ORDER BY ts.substitution_date, tsl.start_time
LIMIT $5 OFFSET $6;

-- name: CountTimetableSubstitutions :one
SELECT COUNT(*) FROM timetable_substitutions ts
WHERE ts.school_id = $1
  AND ($2::date IS NULL OR ts.substitution_date >= $2)
  AND ($3::date IS NULL OR ts.substitution_date <= $3)
  AND ($4::text IS NULL OR ts.status = $4);

-- name: CountSubstitutionConflicts :one
SELECT COUNT(*) FROM timetable_substitutions
WHERE school_id = $1
  AND class_timetable_slot_id = $2
  AND substitution_date = $3
  AND substitute_teacher_membership_id = $4
  AND status IN ('PENDING', 'ASSIGNED')
  AND ($5::uuid IS NULL OR id != $5);

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
WHERE id = $1;

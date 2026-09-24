-- name: GetTeacherSummary :one
SELECT
  COUNT(*) AS total_teachers,
  COUNT(*) FILTER (WHERE NOT EXISTS (
    SELECT 1
    FROM class_timetable_slots cts
    WHERE cts.teacher_membership_id = sm.id
      AND cts.academic_term_id = $2
  )) AS teachers_without_assignment
FROM school_memberships sm
WHERE sm.school_id = $1
  AND sm.role = 'TEACHER';

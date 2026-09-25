-- name: GetGuardianSummary :one
SELECT
  COUNT(*) AS total_guardians,
  COUNT(*) FILTER (WHERE NOT EXISTS (
    SELECT 1 FROM guardian_student_links gsl WHERE gsl.school_membership_id = sm.id
  )) AS guardians_without_student
FROM school_memberships sm
WHERE sm.school_id = $1 AND sm.role = 'GUARDIAN' AND sm.is_active = true;

-- name: GetGuardianSummary :one
SELECT
  COUNT(*) AS total_guardians,
  COUNT(*) FILTER (WHERE NOT EXISTS (
    SELECT 1 FROM guardian_student_links gsl WHERE gsl.school_membership_id = sm.id
  )) AS guardians_without_student
FROM school_memberships sm
WHERE sm.school_id = $1 AND sm.role = 'GUARDIAN' AND sm.is_active = true;

-- name: ListGuardians :many
SELECT sm.id, sm.user_id, u.email, u.full_name, sm.invited_at, sm.accepted_at, sm.is_active, sm.created_at
FROM school_memberships sm
JOIN users u ON u.id = sm.user_id
WHERE sm.school_id = $1
  AND sm.role = 'GUARDIAN'
  AND ($2::text = '' OR u.email ILIKE '%' || $2 || '%' OR u.full_name ILIKE '%' || $2 || '%')
  AND (
    $3::text = '' OR
    ($3 = 'invited' AND sm.invited_at IS NOT NULL AND sm.accepted_at IS NULL) OR
    ($3 = 'accepted' AND sm.accepted_at IS NOT NULL)
  )
ORDER BY sm.created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountGuardians :one
SELECT COUNT(*)
FROM school_memberships sm
JOIN users u ON u.id = sm.user_id
WHERE sm.school_id = $1
  AND sm.role = 'GUARDIAN'
  AND ($2::text = '' OR u.email ILIKE '%' || $2 || '%' OR u.full_name ILIKE '%' || $2 || '%')
  AND (
    $3::text = '' OR
    ($3 = 'invited' AND sm.invited_at IS NOT NULL AND sm.accepted_at IS NULL) OR
    ($3 = 'accepted' AND sm.accepted_at IS NOT NULL)
  );

-- name: DeleteGuardians :exec
DELETE FROM school_memberships
WHERE school_id = $1 AND user_id = ANY($2::uuid[]) AND role = 'GUARDIAN';

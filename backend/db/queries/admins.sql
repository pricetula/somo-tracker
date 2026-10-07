-- name: ListAdmins :many
SELECT sm.id, sm.user_id, u.email, u.full_name, sm.invited_at, sm.accepted_at, sm.is_active, sm.created_at
FROM school_memberships sm
JOIN users u ON u.id = sm.user_id
WHERE sm.school_id = $1
  AND sm.role = 'ADMIN'
  AND ($2::text = '' OR u.email ILIKE '%' || $2 || '%' OR u.full_name ILIKE '%' || $2 || '%')
  AND (
    $3::text = '' OR
    ($3 = 'invited' AND sm.invited_at IS NOT NULL AND sm.accepted_at IS NULL) OR
    ($3 = 'accepted' AND sm.accepted_at IS NOT NULL)
  )
ORDER BY sm.created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountAdmins :one
SELECT COUNT(*)
FROM school_memberships sm
JOIN users u ON u.id = sm.user_id
WHERE sm.school_id = $1
  AND sm.role = 'ADMIN'
  AND ($2::text = '' OR u.email ILIKE '%' || $2 || '%' OR u.full_name ILIKE '%' || $2 || '%')
  AND (
    $3::text = '' OR
    ($3 = 'invited' AND sm.invited_at IS NOT NULL AND sm.accepted_at IS NULL) OR
    ($3 = 'accepted' AND sm.accepted_at IS NOT NULL)
  );

-- name: DeleteAdmins :exec
DELETE FROM school_memberships
WHERE school_id = $1 AND user_id = ANY($2::uuid[]);

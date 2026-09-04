-- name: GetMemberByStytchMemberID :one
SELECT
    id,
    stytch_member_id,
    user_id,
    tenant_id,
    roles,
    stytch_member_raw,
    created_at,
    updated_at
FROM members
WHERE stytch_member_id = $1
LIMIT 1;

-- name: CreateMember :one
INSERT INTO members (stytch_member_id, user_id, tenant_id, roles, stytch_member_raw)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, stytch_member_id, user_id, tenant_id, roles, stytch_member_raw, created_at, updated_at;

-- name: UpdateMemberRoles :exec
UPDATE members
SET roles = $2, updated_at = NOW()
WHERE id = $1;

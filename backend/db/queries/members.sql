-- name: GetMemberByStytchMemberID :one
SELECT
    id,
    stytch_member_id,
    user_id,
    tenant_id,
    stytch_member_raw,
    created_at,
    updated_at
FROM members
WHERE stytch_member_id = $1
LIMIT 1;

-- name: CreateMember :one
INSERT INTO members (stytch_member_id, user_id, tenant_id, stytch_member_raw)
VALUES ($1, $2, $3, $4)
RETURNING id, stytch_member_id, user_id, tenant_id, stytch_member_raw, created_at, updated_at;

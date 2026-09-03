-- name: GetUserByID :one
SELECT
    id,
    email,
    tenant_id,
    full_name,
    is_active,
    external_auth_id,
    created_at,
    updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT
    id,
    email,
    tenant_id,
    full_name,
    is_active,
    external_auth_id,
    created_at,
    updated_at
FROM users
WHERE email = $1
  AND tenant_id = $2
LIMIT 1;

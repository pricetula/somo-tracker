-- name: GetAuthUserByEmail :one
SELECT id FROM users WHERE email = $1 LIMIT 1;

-- name: GetAuthTenantByStytchOrgID :one
SELECT id FROM tenants WHERE stytch_org_id = $1 LIMIT 1;

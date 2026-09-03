-- name: GetTenantByStytchOrgID :one
SELECT
    id,
    name,
    slug,
    stytch_org_id,
    created_at
FROM tenants
WHERE stytch_org_id = $1
LIMIT 1;

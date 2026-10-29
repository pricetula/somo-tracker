-- name: CreateTenant :one
INSERT INTO tenant (name, stytch_organization_id, status)
VALUES ($1, $2, 'active')
RETURNING id, name, stytch_organization_id, status, created_at, updated_at;

-- name: GetTenantByID :one
SELECT id, name, stytch_organization_id, status, created_at, updated_at
FROM tenant
WHERE id = $1
LIMIT 1;

-- name: GetTenantByStytchOrgID :one
SELECT id, name, stytch_organization_id, status, created_at, updated_at
FROM tenant
WHERE stytch_organization_id = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO "user" (tenant_id, email_normalized, stytch_member_id, name)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM "user"
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmailAndTenant :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM "user"
WHERE tenant_id = $1 AND email_normalized = $2
LIMIT 1;

-- name: GetUserByStytchMemberID :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM "user"
WHERE stytch_member_id = $1
LIMIT 1;

-- name: CreateSchool :one
INSERT INTO school (tenant_id, name, code)
VALUES ($1, $2, $3)
RETURNING id, tenant_id, name, code, created_at, updated_at;

-- name: GetSchoolByID :one
SELECT id, tenant_id, name, code, created_at, updated_at
FROM school
WHERE id = $1
LIMIT 1;

-- name: CreateMembership :one
INSERT INTO membership (user_id, school_id, role)
VALUES ($1, $2, $3)
RETURNING id, user_id, school_id, role, created_at;

-- name: GetMembershipsByUserID :many
SELECT id, user_id, school_id, role, created_at
FROM membership
WHERE user_id = $1;

-- name: GetMembershipByUserAndSchool :one
SELECT id, user_id, school_id, role, created_at
FROM membership
WHERE user_id = $1 AND school_id = $2
LIMIT 1;

-- name: CreatePKCECode :exec
INSERT INTO pkce_code (pending_onboarding_id, code_verifier_hash, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (pending_onboarding_id) DO UPDATE SET
  code_verifier_hash = EXCLUDED.code_verifier_hash,
  expires_at = EXCLUDED.expires_at;

-- name: GetPKCECode :one
SELECT pending_onboarding_id, code_verifier_hash, expires_at, created_at
FROM pkce_code
WHERE pending_onboarding_id = $1 AND expires_at > NOW()
LIMIT 1;

-- name: DeletePKCECode :exec
DELETE FROM pkce_code WHERE pending_onboarding_id = $1;

-- name: GetTenantMembershipRole :one
SELECT m.role
FROM membership m
JOIN "user" u ON m.user_id = u.id
WHERE u.id = $1 AND u.tenant_id = $2
LIMIT 1;
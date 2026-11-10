-- name: CreateTenant :one
INSERT INTO tenants (name, stytch_organization_id, tenant_status)
VALUES ($1, $2, 'ACTIVE')
RETURNING id, name, stytch_organization_id, tenant_status, created_at, updated_at;

-- name: GetTenantByID :one
SELECT id, name, stytch_organization_id, tenant_status, created_at, updated_at
FROM tenants
WHERE id = $1
LIMIT 1;

-- name: GetTenantByStytchOrgID :one
SELECT id, name, stytch_organization_id, tenant_status, created_at, updated_at
FROM tenants
WHERE stytch_organization_id = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (tenant_id, email_normalized, stytch_member_id, name)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmailAndTenant :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM users
WHERE tenant_id = $1 AND email_normalized = $2
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM users
WHERE email_normalized = $1
LIMIT 1;

-- name: GetUserByStytchMemberID :one
SELECT id, tenant_id, email_normalized, stytch_member_id, name, created_at, updated_at
FROM users
WHERE stytch_member_id = $1
LIMIT 1;

-- name: CreateSchool :one
INSERT INTO schools (tenant_id, name, code, country_id, education_system_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, name, code, country_id, education_system_id, created_at, updated_at;

-- name: GetSchoolByID :one
SELECT id, tenant_id, name, code, country_id, education_system_id, created_at, updated_at
FROM schools
WHERE id = $1
LIMIT 1;

-- name: CreateMembership :one
INSERT INTO memberships (user_id, school_id, role)
VALUES ($1, $2, $3)
RETURNING id, user_id, school_id, role, created_at;

-- name: GetMembershipsByUserID :many
SELECT id, user_id, school_id, role, created_at
FROM memberships
WHERE user_id = $1;

-- name: GetMembershipByUserAndSchool :one
SELECT id, user_id, school_id, role, created_at
FROM memberships
WHERE user_id = $1 AND school_id = $2
LIMIT 1;

-- name: CreatePKCECode :exec
INSERT INTO pkce_codes (pending_onboarding_id, code_verifier_hash, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (pending_onboarding_id) DO UPDATE SET
  code_verifier_hash = EXCLUDED.code_verifier_hash,
  expires_at = EXCLUDED.expires_at;

-- name: GetPKCECode :one
SELECT pending_onboarding_id, code_verifier_hash, expires_at, created_at
FROM pkce_codes
WHERE pending_onboarding_id = $1 AND expires_at > NOW()
LIMIT 1;

-- name: DeletePKCECode :exec
DELETE FROM pkce_codes WHERE pending_onboarding_id = $1;

-- name: GetTenantMembershipRole :one
SELECT m.role
FROM memberships m
JOIN users u ON m.user_id = u.id
WHERE u.id = $1 AND u.tenant_id = $2
LIMIT 1;
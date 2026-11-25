-- name: GetOrCreateUser :one
INSERT INTO users (organization_id, email, normalized_email, status, email_verified_at)
VALUES ($1, $2, $3, 'ACTIVE', NULL)
ON CONFLICT (normalized_email) DO NOTHING
RETURNING id, organization_id, email, normalized_email, status, email_verified_at, created_at, updated_at, deleted_at;

-- name: GetUserByNormalizedEmail :one
SELECT id, organization_id, email, normalized_email, name, status, email_verified_at, created_at, updated_at, deleted_at
FROM users
WHERE normalized_email = $1 AND deleted_at IS NULL;

-- name: CreateOrganization :one
INSERT INTO organizations (name, status)
VALUES ($1, 'ACTIVE')
RETURNING id, name, status, created_at, updated_at, deleted_at, onboarding_completed_at;

-- name: CreateUser :one
INSERT INTO users (organization_id, email, normalized_email, name, status, email_verified_at)
VALUES ($1, $2, $3, $4, 'ACTIVE', NOW())
RETURNING id, organization_id, email, normalized_email, name, status, email_verified_at, created_at, updated_at, deleted_at;

-- name: CreateSchool :one
INSERT INTO schools (organization_id, name, status)
VALUES ($1, $2, 'ACTIVE')
RETURNING id, organization_id, name, status, created_at, updated_at, deleted_at;

-- name: CreateUserSchool :one
INSERT INTO user_schools (organization_id, user_id, school_id, role, status)
VALUES ($1, $2, $3, 'owner', 'ACTIVE')
RETURNING id, organization_id, user_id, school_id, role, status, created_at, revoked_at;

-- name: UpdateOrganization :exec
UPDATE organizations
SET name = $1, onboarding_completed_at = NOW(), updated_at = NOW()
WHERE id = $2;

-- name: UpdateUser :exec
UPDATE users
SET name = $1, updated_at = NOW()
WHERE id = $2;

-- name: UpdateSessionType :exec
UPDATE sessions
SET session_type = 'full'
WHERE user_id = $1 AND session_type = 'temporary';

-- name: GetSessionByTokenHash :one
SELECT id, token_hash, user_id, organization_id, active_school_id, session_type, created_at, last_seen_at, expires_at, revoked_at, revocation_reason
FROM sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: CreateLoginChallenge :one
INSERT INTO login_challenges (id, email_normalized, purpose, code_mac, expires_at, attempt_count, max_attempts)
VALUES ($1, $2, 'login', $3, $4, 0, 5)
RETURNING id, email_normalized, purpose, code_mac, expires_at, attempt_count, max_attempts, used_at, invalidated_at, created_at;

-- name: GetLoginChallengeByID :one
SELECT id, email_normalized, purpose, code_mac, expires_at, attempt_count, max_attempts, used_at, invalidated_at, created_at
FROM login_challenges
WHERE id = $1;

-- name: MarkChallengeUsed :exec
UPDATE login_challenges
SET used_at = NOW()
WHERE id = $1 AND used_at IS NULL;

-- name: InvalidateChallengesByEmail :exec
UPDATE login_challenges
SET invalidated_at = NOW()
WHERE email_normalized = $1 AND used_at IS NULL AND invalidated_at IS NULL;

-- name: CreateEmailOutbox :one
INSERT INTO email_outbox (challenge_id, message_type, status, encrypted_payload)
VALUES ($1, 'otp', 'pending', $2)
RETURNING id, challenge_id, message_type, status, attempts, next_attempt_at, provider_message_id, encrypted_payload, created_at, sent_at;

-- name: GetActiveSession :one
SELECT id, token_hash, user_id, organization_id, active_school_id, created_at, last_seen_at, expires_at, revoked_at, revocation_reason
FROM sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: CreateSession :one
INSERT INTO sessions (token_hash, user_id, organization_id, active_school_id, session_type, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, token_hash, user_id, organization_id, active_school_id, session_type, created_at, last_seen_at, expires_at, revoked_at, revocation_reason;

-- name: GetOrganizationsByUser :many
SELECT o.id, o.name, o.status, o.created_at, o.updated_at, o.deleted_at, o.onboarding_completed_at
FROM organizations o
JOIN users u ON u.organization_id = o.id
WHERE u.id = $1 AND u.deleted_at IS NULL AND o.deleted_at IS NULL;

-- name: GetSchoolsByUser :many
SELECT s.id, s.organization_id, s.name, s.status, s.created_at, s.updated_at, s.deleted_at
FROM schools s
JOIN user_schools us ON us.school_id = s.id AND us.organization_id = s.organization_id
WHERE us.user_id = $1 AND us.status = 'ACTIVE' AND s.deleted_at IS NULL;

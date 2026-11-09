-- name: CreateAppSession :one
INSERT INTO app_session (user_id, tenant_id, token_hash, stytch_member_session_id, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at;

-- name: GetAppSessionByTokenHash :one
SELECT id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at
FROM app_session
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
LIMIT 1;

-- name: GetAppSessionByID :one
SELECT id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at
FROM app_session
WHERE id = $1;

-- name: GetSessionWithMemberships :one
SELECT s.id, s.user_id, s.tenant_id, s.token_hash, s.stytch_member_session_id, s.created_at, s.expires_at, s.revoked_at,
       m.id as membership_id, m.school_id, m.role, m.is_active
FROM app_session s
LEFT JOIN memberships m ON m.user_id = s.user_id AND m.is_active = true
WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
LIMIT 1;

-- name: RevokeAppSession :exec
UPDATE app_session
SET revoked_at = NOW()
WHERE id = $1;

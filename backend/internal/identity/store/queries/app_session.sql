-- name: CreateAppSession :one
INSERT INTO app_session (user_id, tenant_id, token_hash, stytch_member_session_id, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at;

-- name: GetAppSessionByTokenHash :one
SELECT id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at
FROM app_session
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
LIMIT 1;

-- name: RevokeAppSession :exec
UPDATE app_session
SET revoked_at = NOW()
WHERE id = $1;

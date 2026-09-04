-- name: GetSessionByToken :one
SELECT
    id,
    token,
    stytch_session_id,
    user_id,
    tenant_id,
    expires_at,
    created_at,
    last_seen_at
FROM sessions
WHERE token = $1
LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (token, stytch_session_id, user_id, tenant_id, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, token, stytch_session_id, user_id, tenant_id, expires_at, created_at, last_seen_at;

-- name: UpdateSessionLastSeen :exec
UPDATE sessions
SET last_seen_at = NOW()
WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = $1;

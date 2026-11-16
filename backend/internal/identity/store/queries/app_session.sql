-- name: CreateAppSession :one
INSERT INTO app_session (user_id, tenant_id, token_hash, stytch_member_session_id, expires_at, last_seen_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at, last_seen_at;

-- name: GetAppSessionByTokenHash :one
SELECT id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at, last_seen_at
FROM app_session
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
LIMIT 1;

-- name: GetAppSessionByID :one
SELECT id, user_id, tenant_id, token_hash, stytch_member_session_id, created_at, expires_at, revoked_at, last_seen_at
FROM app_session
WHERE id = $1;

-- name: GetSessionWithMemberships :one
-- Resolves opaque session + the user's single active membership in one query.
-- INNER JOIN users fails closed: missing user or tenant mismatch returns no row -> ErrSessionNotFound.
-- LEFT JOIN memberships keeps the session result when no usable membership exists;
-- memberships are returned only when their school belongs to the session tenant
-- (defense in depth per ADR-0002).
-- memberships_active_user_uniq_idx guarantees at most one active membership per user.
SELECT s.id, s.user_id, s.tenant_id, s.token_hash, s.stytch_member_session_id, s.created_at, s.expires_at, s.revoked_at, s.last_seen_at,
       m.id as membership_id, m.school_id, m.role, m.is_active
FROM app_session s
JOIN users u ON u.id = s.user_id AND u.tenant_id = s.tenant_id
LEFT JOIN memberships m
       ON m.user_id = s.user_id
      AND m.is_active = true
      AND EXISTS (
          SELECT 1
          FROM schools sc
          WHERE sc.id = m.school_id
            AND sc.tenant_id = s.tenant_id
      )
WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
LIMIT 1;

-- name: UpdateSessionLastSeen :exec
UPDATE app_session
SET last_seen_at = NOW()
WHERE id = $1 AND last_seen_at < NOW() - $2;

-- name: RevokeAppSession :exec
UPDATE app_session
SET revoked_at = NOW()
WHERE id = $1;

-- name: RevokeAllSessionsForUser :exec
UPDATE app_session
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

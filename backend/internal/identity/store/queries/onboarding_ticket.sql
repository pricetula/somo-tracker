-- name: CreateOnboardingTicket :one
INSERT INTO onboarding_ticket (pending_onboarding_id, ticket_hash, encrypted_ist, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, pending_onboarding_id, ticket_hash, encrypted_ist, used_at, expires_at, created_at;

-- name: GetOnboardingTicketByHash :one
SELECT id, pending_onboarding_id, ticket_hash, encrypted_ist, used_at, expires_at, created_at
FROM onboarding_ticket
WHERE ticket_hash = $1 AND used_at IS NULL AND expires_at > NOW()
LIMIT 1;

-- name: MarkOnboardingTicketUsed :exec
UPDATE onboarding_ticket
SET used_at = NOW()
WHERE id = $1;

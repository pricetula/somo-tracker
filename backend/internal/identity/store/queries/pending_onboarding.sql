-- name: GetPendingOnboardingByEmail :one
SELECT id, verified_email_normalized, status, current_step, form_data, stytch_organization_id, stytch_member_id, tenant_id, school_name, idempotency_key, created_at, updated_at, last_activity_at
FROM pending_onboarding
WHERE verified_email_normalized = $1
LIMIT 1;

-- name: UpsertPendingOnboarding :one
INSERT INTO pending_onboarding (verified_email_normalized, status, current_step, form_data, stytch_organization_id, stytch_member_id, tenant_id, school_name, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (verified_email_normalized) DO UPDATE SET
  status = EXCLUDED.status,
  current_step = EXCLUDED.current_step,
  form_data = EXCLUDED.form_data,
  stytch_organization_id = COALESCE(EXCLUDED.stytch_organization_id, pending_onboarding.stytch_organization_id),
  stytch_member_id = COALESCE(EXCLUDED.stytch_member_id, pending_onboarding.stytch_member_id),
  tenant_id = COALESCE(EXCLUDED.tenant_id, pending_onboarding.tenant_id),
  school_name = COALESCE(EXCLUDED.school_name, pending_onboarding.school_name),
  idempotency_key = COALESCE(EXCLUDED.idempotency_key, pending_onboarding.idempotency_key),
  updated_at = NOW(),
  last_activity_at = NOW()
RETURNING id, verified_email_normalized, status, current_step, form_data, stytch_organization_id, stytch_member_id, tenant_id, school_name, idempotency_key, created_at, updated_at, last_activity_at;

-- name: UpdatePendingOnboardingStep :exec
UPDATE pending_onboarding
SET current_step = $2, form_data = $3, updated_at = NOW(), last_activity_at = NOW()
WHERE id = $1;

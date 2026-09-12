-- name: UpsertSchoolMembership :one
-- Upserts (or updates) a school_membership during bulk invitation.
-- On conflict over the school/user unique constraint, updates invitation state,
-- role, active flag, and invitation metadata. Returns the full row.
INSERT INTO school_memberships (
    school_id,
    user_id,
    role,
    is_active,
    invited_at,
    invited_by,
    accepted_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (school_id, user_id) DO UPDATE SET
    role = EXCLUDED.role,
    is_active = EXCLUDED.is_active,
    invited_at = EXCLUDED.invited_at,
    invited_by = EXCLUDED.invited_by,
    accepted_at = EXCLUDED.accepted_at,
    updated_at = NOW()
RETURNING id, school_id, user_id, role, is_active, invited_at, invited_by, accepted_at, created_at, updated_at;

-- name: UpdateMembershipInvitationState :exec
-- Updates invitation/acceptance timestamps and role for an existing membership.
UPDATE school_memberships
SET
    invited_at = $1,
    invited_by = $2,
    accepted_at = $3,
    role = $4,
    is_active = $5,
    updated_at = NOW()
WHERE id = $6;

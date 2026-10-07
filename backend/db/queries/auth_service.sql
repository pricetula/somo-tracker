-- name: GetActiveSchoolMembershipByUser :one
SELECT school_id FROM school_memberships WHERE user_id = $1 AND is_active = true LIMIT 1;

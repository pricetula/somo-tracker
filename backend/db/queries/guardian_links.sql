-- name: LinkGuardianToStudent :one
INSERT INTO guardian_student_links (school_membership_id, student_id, relationship_type)
VALUES ($1, $2, $3)
ON CONFLICT (school_membership_id, student_id) DO UPDATE SET relationship_type = EXCLUDED.relationship_type, updated_at = NOW()
RETURNING id;

-- name: UnlinkGuardianFromStudent :exec
DELETE FROM guardian_student_links WHERE school_membership_id = $1 AND student_id = $2;

-- name: ListGuardiansForStudent :many
SELECT gsl.id, gsl.school_membership_id, gsl.student_id, gsl.relationship_type, gsl.created_at, gsl.updated_at,
       u.id as user_id, u.full_name, u.email, sm.role
FROM guardian_student_links gsl
JOIN school_memberships sm ON sm.id = gsl.school_membership_id
JOIN users u ON u.id = sm.user_id
WHERE gsl.student_id = $1;

-- name: ListStudentsForGuardian :many
SELECT gsl.id, gsl.school_membership_id, gsl.student_id, gsl.relationship_type, gsl.created_at, gsl.updated_at,
       s.full_name as student_name, s.admission_number
FROM guardian_student_links gsl
JOIN students s ON s.student_id = gsl.student_id
WHERE gsl.school_membership_id = $1;

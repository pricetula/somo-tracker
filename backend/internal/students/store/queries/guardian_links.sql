-- name: ListGuardianLinks :many
SELECT id, guardian_id, student_id, relationship_type, is_emergency_contact, created_at, updated_at, deleted_at
FROM guardian_student_links
WHERE student_id = $1
  AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: CreateGuardianLink :one
INSERT INTO guardian_student_links (guardian_id, student_id, relationship_type, is_emergency_contact)
VALUES ($1, $2, $3, $4)
RETURNING id, guardian_id, student_id, relationship_type, is_emergency_contact, created_at, updated_at, deleted_at;

-- name: DeleteGuardianLink :exec
DELETE FROM guardian_student_links
WHERE student_id = $1
  AND guardian_id = $2
  AND deleted_at IS NULL;

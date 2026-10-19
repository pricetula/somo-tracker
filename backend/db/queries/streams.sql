-- name: CreateStream :one
INSERT INTO streams (school_id, name, color)
VALUES ($1, $2, $3)
RETURNING id, school_id, name, color, created_at, updated_at;

-- name: ListStreamsBySchool :many
SELECT id, school_id, name, color, created_at, updated_at FROM streams WHERE school_id = $1;

-- name: ListStreamsBySchoolPaginated :many
SELECT id, school_id, name, color, created_at, updated_at FROM streams WHERE school_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountStreamsBySchool :one
SELECT COUNT(*) FROM streams WHERE school_id = $1;

-- name: GetStream :one
SELECT id, school_id, name, color FROM streams WHERE id = $1;

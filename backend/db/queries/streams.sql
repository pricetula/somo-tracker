-- name: CreateStream :one
INSERT INTO streams (school_id, name, color)
VALUES ($1, $2, $3)
RETURNING id, school_id, name, color, created_at, updated_at;

-- name: CreateRoom :one
INSERT INTO rooms (school_id, name, capacity, room_type)
VALUES ($1, $2, $3, $4)
RETURNING id, school_id, name, capacity, room_type, created_at, updated_at;

-- name: UpdateRoom :one
UPDATE rooms
SET name = COALESCE(NULLIF($3::text, ''), name),
    capacity = COALESCE($4, capacity),
    room_type = COALESCE(NULLIF($5::text, ''), room_type),
    updated_at = NOW()
WHERE id = $1 AND school_id = $2
RETURNING id, school_id, name, capacity, room_type, created_at, updated_at;

-- name: DeleteRoom :exec
DELETE FROM rooms WHERE id = $1 AND school_id = $2;

-- name: ListRoomsBySchool :many
SELECT id, school_id, name, capacity, room_type, created_at, updated_at
FROM rooms
WHERE school_id = $1
ORDER BY name ASC;

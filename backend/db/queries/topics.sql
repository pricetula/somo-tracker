-- name: ListTopicsBySubject :many
SELECT id, subject_id, name, sequence_index FROM topics WHERE subject_id = $1 ORDER BY sequence_index ASC;

-- name: GetTopicByID :one
SELECT id, subject_id, name, sequence_index FROM topics WHERE id = $1;

-- name: ListTopics :many
SELECT id, subject_id, name, sequence_index FROM topics ORDER BY subject_id, sequence_index ASC LIMIT $1 OFFSET $2;

-- name: CountTopicsBySubject :one
SELECT COUNT(*) FROM topics WHERE subject_id = $1;

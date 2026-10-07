-- name: ListSubTopicsByTopic :many
SELECT id, topic_id, name, sequence_index FROM sub_topics WHERE topic_id = $1 ORDER BY sequence_index ASC;

-- name: GetSubTopicByID :one
SELECT id, topic_id, name, sequence_index FROM sub_topics WHERE id = $1;

-- name: ListSubTopics :many
SELECT id, topic_id, name, sequence_index FROM sub_topics ORDER BY topic_id, sequence_index ASC LIMIT $1 OFFSET $2;

-- name: CountSubTopicsByTopic :one
SELECT COUNT(*) FROM sub_topics WHERE topic_id = $1;

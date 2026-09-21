-- name: GetGradeLevelByID :one
SELECT id, country_id, education_system_id, local_label FROM grade_levels WHERE id = $1;

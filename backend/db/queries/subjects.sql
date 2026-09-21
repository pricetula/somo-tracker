-- name: ListSubjectsBySystem :many
SELECT id, education_system_id, name, code, type, grade_level_id, color, created_at, updated_at FROM subjects WHERE education_system_id = $1 ORDER BY name ASC;

-- name: GetSubjectByID :one
SELECT id, education_system_id, name, code, type, grade_level_id, color, created_at, updated_at FROM subjects WHERE id = $1;

-- name: ListSubjects :many
SELECT s.id, s.grade_level_id, s.name, s.code, COALESCE(s.color, ''), COALESCE(gl.local_label, '') FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id ORDER BY s.name ASC LIMIT $1 OFFSET $2;

-- name: CountSubjects :one
SELECT COUNT(*) FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id;

-- name: SearchSubjects :many
SELECT s.id, s.grade_level_id, s.name, s.code, COALESCE(s.color, ''), COALESCE(gl.local_label, '') FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id WHERE (s.name ILIKE $1 OR s.code ILIKE $1) ORDER BY s.name ASC LIMIT $2 OFFSET $3;

-- name: CountSearchSubjects :one
SELECT COUNT(*) FROM subjects s LEFT JOIN grade_levels gl ON gl.id = s.grade_level_id WHERE (s.name ILIKE $1 OR s.code ILIKE $1);

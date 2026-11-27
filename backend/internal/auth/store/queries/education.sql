-- name: ListCountries :many
SELECT id, name, code, created_at, updated_at
FROM countries
ORDER BY name;

-- name: GetCountry :one
SELECT id, name, code, created_at, updated_at
FROM countries
WHERE id = $1;

-- name: ListEducationSystems :many
SELECT es.id, es.country_id, es.name, es.created_at, es.updated_at,
       c.id AS country_id__1, c.name AS country_name, c.code AS country_code, c.created_at AS country_created_at, c.updated_at AS country_updated_at
FROM education_systems es
JOIN countries c ON c.id = es.country_id
ORDER BY c.name, es.name;

-- name: GetEducationSystem :one
SELECT es.id, es.country_id, es.name, es.created_at, es.updated_at,
       c.id AS country_id__1, c.name AS country_name, c.code AS country_code, c.created_at AS country_created_at, c.updated_at AS country_updated_at
FROM education_systems es
JOIN countries c ON c.id = es.country_id
WHERE es.id = $1;

-- name: ListEducationSystemLevels :many
SELECT el.id, el.education_system_id, el.id AS education_level_id, el.year_count, el.created_at,
       el.id AS education_level_id__1, el.name AS level_name, el.sort_order, el.created_at AS level_created_at
FROM education_levels el
WHERE el.education_system_id = $1
ORDER BY el.sort_order;

-- name: ListEducationSystemGradingBands :many
SELECT gb.id, gb.grading_scale_id, gb.grade, gb.scale_type, gb.rubric_level, gb.min_score, gb.max_score, gb.sort_order, gb.created_at,
       gs.id AS grading_scale_id__1, gs.name AS scale_name, gs.is_default
FROM grading_bands gb
JOIN grading_scales gs ON gs.id = gb.grading_scale_id
WHERE gs.education_system_id = $1
ORDER BY gs.is_default DESC, gb.sort_order;

-- name: ListEducationSystemSubjects :many
SELECT s.id, s.education_system_id, s.education_level_id, s.code, s.name, s.is_compulsory_default, s.created_at,
       el.id AS education_level_id__1, el.name AS level_name, el.sort_order
FROM subjects s
JOIN education_levels el ON el.id = s.education_level_id
WHERE s.education_system_id = $1
ORDER BY el.sort_order, s.code;



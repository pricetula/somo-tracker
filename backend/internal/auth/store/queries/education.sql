-- name: ListCountries :many
SELECT id, name, code, created_at, updated_at
FROM countries
ORDER BY name;

-- name: GetCountry :one
SELECT id, name, code, created_at, updated_at
FROM countries
WHERE id = $1;

-- name: ListEducationSystems :many
SELECT es.id, es.country_id, es.name, es.years_primary, es.years_lower_secondary, es.years_upper_secondary, es.default_language, es.created_at, es.updated_at,
       c.id AS country_id__1, c.name AS country_name, c.code AS country_code, c.created_at AS country_created_at, c.updated_at AS country_updated_at
FROM education_systems es
JOIN countries c ON c.id = es.country_id
ORDER BY c.name, es.name;

-- name: GetEducationSystem :one
SELECT es.id, es.country_id, es.name, es.years_primary, es.years_lower_secondary, es.years_upper_secondary, es.default_language, es.created_at, es.updated_at,
       c.id AS country_id__1, c.name AS country_name, c.code AS country_code, c.created_at AS country_created_at, c.updated_at AS country_updated_at
FROM education_systems es
JOIN countries c ON c.id = es.country_id
WHERE es.id = $1;

-- name: ListEducationSystemLevels :many
SELECT esl.id, esl.education_system_id, esl.education_level_id, esl.year_count, esl.created_at,
       el.id AS education_level_id__1, el.name AS level_name, el.sort_order, el.created_at AS level_created_at
FROM education_system_levels esl
JOIN education_levels el ON el.id = esl.education_level_id
WHERE esl.education_system_id = $1
ORDER BY el.sort_order;

-- name: ListEducationSystemGradingBands :many
SELECT esgb.id, esgb.grading_scale_id, esgb.grade, esgb.scale_type, esgb.rubric_level, esgb.min_score, esgb.max_score, esgb.sort_order, esgb.created_at,
       esgs.id AS grading_scale_id__1, esgs.name AS scale_name, esgs.is_default
FROM education_system_grading_bands esgb
JOIN education_system_grading_scales esgs ON esgs.id = esgb.grading_scale_id
WHERE esgs.education_system_id = $1
ORDER BY esgs.is_default DESC, esgb.sort_order;

-- name: ListEducationSystemSubjects :many
SELECT ess.id, ess.education_system_id, ess.education_level_id, ess.subject_code, ess.subject_name, ess.is_compulsory_default, ess.created_at,
       el.id AS education_level_id__1, el.name AS level_name, el.sort_order
FROM education_system_subjects ess
JOIN education_levels el ON el.id = ess.education_level_id
WHERE ess.education_system_id = $1
ORDER BY el.sort_order, ess.subject_code;

-- name: ListEducationSystemTermStructures :many
SELECT est.id, est.education_system_id, est.term_name, est.term_sort_order, est.created_at
FROM education_system_term_structures est
WHERE est.education_system_id = $1
ORDER BY est.term_sort_order;

-- name: ListCountries :many
SELECT id, country_name, country_code, created_at, updated_at, deleted_at
FROM countries
WHERE deleted_at IS NULL
ORDER BY country_name;

-- name: ListEducationSystems :many
SELECT id, country_id, system_name, description, created_at, updated_at, deleted_at
FROM education_systems
WHERE country_id = $1 AND deleted_at IS NULL
ORDER BY system_name;

-- name: GetCountryByName :one
SELECT id FROM countries WHERE country_name = $1 LIMIT 1;

-- name: GetEducationSystemByCountryAndName :one
SELECT id FROM education_systems WHERE country_id = $1 AND system_name ILIKE $2 LIMIT 1;

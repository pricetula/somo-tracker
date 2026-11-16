-- name: GetSchool :one
SELECT id, tenant_id, name, code, country_id, education_system_id, created_at, updated_at, deleted_at
FROM schools
WHERE id = $1
  AND deleted_at IS NULL
  AND tenant_id = current_tenant_id();

-- name: ListSchools :many
SELECT id, tenant_id, name, code, country_id, education_system_id, created_at, updated_at, deleted_at
FROM schools
WHERE deleted_at IS NULL
  AND tenant_id = current_tenant_id()
  AND ($2::text IS NULL OR name > $2::text OR (name = $2::text AND id > $3::uuid))
ORDER BY name ASC, id ASC
LIMIT $1;

-- name: CreateSchool :one
INSERT INTO schools (tenant_id, name, code, country_id, education_system_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, name, code, country_id, education_system_id, created_at, updated_at, deleted_at;

-- name: UpdateSchool :one
UPDATE schools
SET name = $2,
    code = $3,
    country_id = $4,
    education_system_id = $5
WHERE id = $1
  AND deleted_at IS NULL
  AND tenant_id = current_tenant_id()
RETURNING id, tenant_id, name, code, country_id, education_system_id, created_at, updated_at, deleted_at;

-- name: DeactivateSchools :exec
UPDATE schools
SET deleted_at = NOW()
WHERE id = ANY($1::uuid[])
  AND tenant_id = current_tenant_id()
  AND deleted_at IS NULL;

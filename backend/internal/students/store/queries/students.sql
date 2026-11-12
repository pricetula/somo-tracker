-- name: GetStudent :one
SELECT student_id, school_id, admission_number, national_number, full_name, date_of_birth, gender, metadata, created_at, updated_at, deleted_at
FROM students
WHERE student_id = $1
  AND deleted_at IS NULL
  AND school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id())
LIMIT 1;

-- name: ListStudents :many
SELECT student_id, school_id, admission_number, national_number, full_name, date_of_birth, gender, metadata, created_at, updated_at, deleted_at
FROM students
WHERE deleted_at IS NULL
  AND school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id())
  AND ($2::text IS NULL OR full_name > $2::text OR (full_name = $2::text AND student_id > $3::uuid))
ORDER BY full_name ASC, student_id ASC
LIMIT $1;

-- name: CreateStudent :one
INSERT INTO students (school_id, admission_number, national_number, full_name, date_of_birth, gender, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING student_id, school_id, admission_number, national_number, full_name, date_of_birth, gender, metadata, created_at, updated_at, deleted_at;

-- name: UpdateStudent :one
UPDATE students
SET admission_number = $2,
    national_number = $3,
    full_name = $4,
    date_of_birth = $5,
    gender = $6,
    metadata = $7
WHERE student_id = $1
  AND deleted_at IS NULL
  AND school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id())
RETURNING student_id, school_id, admission_number, national_number, full_name, date_of_birth, gender, metadata, created_at, updated_at, deleted_at;

-- name: DeleteStudent :exec
UPDATE students
SET deleted_at = NOW()
WHERE student_id = $1
  AND deleted_at IS NULL
  AND school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id());

-- name: ListStudentsForGuardian :many
SELECT s.student_id, s.school_id, s.admission_number, s.national_number, s.full_name, s.date_of_birth, s.gender, s.metadata, s.created_at, s.updated_at, s.deleted_at
FROM students s
INNER JOIN guardian_student_links gsl ON gsl.student_id = s.student_id
WHERE s.deleted_at IS NULL
  AND gsl.guardian_id = $1
  AND gsl.deleted_at IS NULL
  AND s.school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id())
ORDER BY s.full_name ASC, s.student_id ASC
LIMIT $2;

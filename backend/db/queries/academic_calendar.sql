-- name: CreateAcademicYear :one
INSERT INTO academic_years (school_id, name, start_date, end_date)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateAcademicTerm :one
INSERT INTO academic_terms (academic_year_id, name, start_date, end_date)
VALUES ($1, $2, $3, $4)
RETURNING *;

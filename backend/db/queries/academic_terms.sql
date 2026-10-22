-- name: GetCurrentAcademicTermBySchool :one
SELECT at.id, at.start_date, at.end_date
FROM academic_terms at
JOIN academic_years ay ON at.academic_year_id = ay.id
WHERE ay.school_id = $1 AND CURRENT_DATE BETWEEN at.start_date AND at.end_date
ORDER BY at.start_date DESC
LIMIT 1;

-- name: GetLatestAcademicTermBySchool :one
SELECT at.id
FROM academic_terms at
JOIN academic_years ay ON at.academic_year_id = ay.id
WHERE ay.school_id = $1
ORDER BY at.start_date DESC
LIMIT 1;

-- name: ListAcademicTermsBySchool :many
SELECT at.id, at.academic_year_id, at.name, at.start_date, at.end_date, ay.name AS academic_year_name
FROM academic_terms at
JOIN academic_years ay ON at.academic_year_id = ay.id
WHERE ay.school_id = $1
ORDER BY at.start_date ASC;

-- name: GetAcademicTermByID :one
SELECT at.id, at.academic_year_id, at.name, at.start_date, at.end_date
FROM academic_terms at
WHERE at.id = $1;

-- name: GetAcademicYearByID :one
SELECT id, school_id, name, start_date, end_date
FROM academic_years
WHERE id = $1;

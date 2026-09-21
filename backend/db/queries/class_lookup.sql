-- name: GetCurrentAcademicYearBySchool :one
SELECT id FROM academic_years WHERE school_id = $1 ORDER BY start_date DESC LIMIT 1;

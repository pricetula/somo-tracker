-- name: GetLatestAcademicYearBySchool :one
SELECT id, start_date FROM academic_years WHERE school_id = $1 ORDER BY start_date DESC LIMIT 1;

-- name: GetAcademicTermRangeBySchool :one
SELECT at.start_date, at.end_date FROM academic_terms at JOIN academic_years ay ON ay.id = at.academic_year_id WHERE ay.school_id = $1 AND at.start_date <= $2 AND at.end_date >= $3 ORDER BY at.start_date DESC LIMIT 1;

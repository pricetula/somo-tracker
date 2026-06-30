package students

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository implements StudentRepository backed by Postgres.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// List returns a paginated list of students enrolled at the given school,
// optionally filtered by search, class, or gender.
func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Student, int, error) {
	// Count query
	countQuery := `
		SELECT COUNT(DISTINCT s.id)
		FROM cbc_students s
		JOIN cbc_student_enrollments e ON e.student_id = s.id AND e.tenant_id = s.tenant_id
		WHERE s.tenant_id = $1
		  AND e.school_id = $2
		  AND e.status = 'ACTIVE'
	`
	countArgs := []interface{}{filter.TenantID, filter.SchoolID}

	whereClause := ""
	argIdx := 3

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND s.full_name ILIKE $%d", argIdx)
		countArgs = append(countArgs, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.ClassID != "" {
		whereClause += fmt.Sprintf(" AND e.class_id = $%d", argIdx)
		countArgs = append(countArgs, filter.ClassID)
		argIdx++
	}
	if filter.Gender != "" {
		whereClause += fmt.Sprintf(" AND s.gender = $%d", argIdx)
		countArgs = append(countArgs, filter.Gender)
		argIdx++
	}

	countQuery += whereClause

	var total int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("students.Repository.List: count: %w", err)
	}

	if total == 0 {
		return []Student{}, 0, nil
	}

	// Data query
	offset := (filter.Page - 1) * filter.Limit
	dataArgs := countArgs
	dataQuery := `
		SELECT s.id, s.full_name, s.gender, s.date_of_birth::text, s.upi_number,
		       s.knec_assessment_number, c.display_label, e.class_id, s.is_active, s.created_at::text
		FROM cbc_students s
		JOIN cbc_student_enrollments e ON e.student_id = s.id AND e.tenant_id = s.tenant_id
		LEFT JOIN (
			SELECT DISTINCT ON (cc.id) cc.id, cc.display_label
			FROM cbc_classes cc
		) c ON c.id = e.class_id
		WHERE s.tenant_id = $1
		  AND e.school_id = $2
		  AND e.status = 'ACTIVE'
	`
	dataQuery += whereClause
	dataQuery += fmt.Sprintf(" ORDER BY s.full_name ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	dataArgs = append(dataArgs, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("students.Repository.List: query: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var s Student
		var dateOfBirth, upiNumber, knecNumber, className, classID *string
		err := rows.Scan(
			&s.ID, &s.FullName, &s.Gender, &dateOfBirth, &upiNumber,
			&knecNumber, &className, &classID, &s.IsActive, &s.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("students.Repository.List: scan: %w", err)
		}
		if dateOfBirth != nil {
			s.DateOfBirth = dateOfBirth
		}
		if upiNumber != nil {
			s.UPINumber = upiNumber
		}
		if knecNumber != nil {
			s.KNECAssessmentNumber = knecNumber
		}
		if className != nil {
			s.ClassName = className
		}
		if classID != nil {
			s.ClassID = classID
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("students.Repository.List: rows: %w", err)
	}

	if students == nil {
		students = []Student{}
	}

	return students, total, nil
}

// compile-time interface check
var _ StudentRepository = (*PgRepository)(nil)

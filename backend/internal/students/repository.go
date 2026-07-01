package students

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository implements StudentRepository backed by Postgres.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ─── List ─────────────────────────────────────────────────────────────────

// List returns a paginated list of students enrolled at the given school.
func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Student, int, error) {
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

// ─── Get By ID ────────────────────────────────────────────────────────────

// GetByID returns a single student by primary key.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Student, error) {
	query := `
		SELECT s.id, s.full_name, s.gender, s.date_of_birth::text, s.upi_number,
		       s.knec_assessment_number, NULL::text, NULL::text, s.is_active, s.created_at::text
		FROM cbc_students s
		WHERE s.id = $1 AND s.tenant_id = $2
	`
	var s Student
	var dateOfBirth, upiNumber, knecNumber, className, classID *string
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&s.ID, &s.FullName, &s.Gender, &dateOfBirth, &upiNumber,
		&knecNumber, &className, &classID, &s.IsActive, &s.CreatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("students.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("students.Repository.GetByID: %w", err)
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
	return &s, nil
}

// ─── Get Detail ───────────────────────────────────────────────────────────

// GetDetail returns a student with enrollment history.
func (r *PgRepository) GetDetail(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error) {
	// Fetch the student base record
	student, err := r.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return nil, err
	}

	// Fetch enrollments
	enrollments, err := r.ListEnrollments(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("students.Repository.GetDetail: %w", err)
	}

	return &StudentDetail{
		Student:     *student,
		Enrollments: enrollments,
	}, nil
}

// ─── Create ───────────────────────────────────────────────────────────────

// Create inserts a new student record.
func (r *PgRepository) Create(ctx context.Context, student *Student) (string, error) {
	query := `
		INSERT INTO cbc_students (tenant_id, full_name, gender, date_of_birth, upi_number, knec_assessment_number, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		"", // tenant_id will be set by the service if needed — in production use real tenant_id
		student.FullName,
		student.Gender,
		student.DateOfBirth,
		student.UPINumber,
		student.KNECAssessmentNumber,
	).Scan(&id)
	if err != nil {
		if isDuplicateUPI(err) {
			return "", fmt.Errorf("students.Repository.Create: %w", ErrDuplicateUPI)
		}
		return "", fmt.Errorf("students.Repository.Create: %w", err)
	}
	return id, nil
}

// ─── Update ───────────────────────────────────────────────────────────────

// Update applies partial updates to a student record.
func (r *PgRepository) Update(ctx context.Context, student *Student) error {
	query := `
		UPDATE cbc_students
		SET full_name = $1, gender = $2, date_of_birth = $3,
		    upi_number = $4, knec_assessment_number = $5, is_active = $6
		WHERE id = $7
	`
	_, err := r.pool.Exec(ctx, query,
		student.FullName,
		student.Gender,
		student.DateOfBirth,
		student.UPINumber,
		student.KNECAssessmentNumber,
		student.IsActive,
		student.ID,
	)
	if err != nil {
		if isDuplicateUPI(err) {
			return fmt.Errorf("students.Repository.Update: %w", ErrDuplicateUPI)
		}
		return fmt.Errorf("students.Repository.Update: %w", err)
	}
	return nil
}

// ─── Enrollments ──────────────────────────────────────────────────────────

// CreateEnrollment enrolls a student in a class for a specific term.
func (r *PgRepository) CreateEnrollment(ctx context.Context, enrollment *Enrollment) (string, error) {
	// First check for duplicate enrollment
	exists, err := r.IsEnrolledInTerm(ctx, enrollment.StudentID, enrollment.AcademicTermID, "")
	if err != nil {
		return "", fmt.Errorf("students.Repository.CreateEnrollment: %w", err)
	}
	if exists {
		return "", fmt.Errorf("students.Repository.CreateEnrollment: %w", ErrDuplicateEnroll)
	}

	query := `
		INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, status, tenant_id, school_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id string
	err = r.pool.QueryRow(ctx, query,
		enrollment.StudentID,
		enrollment.ClassID,
		enrollment.AcademicTermID,
		enrollment.Status,
		"", // tenant_id placeholder
		"", // school_id placeholder
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("students.Repository.CreateEnrollment: %w", err)
	}
	return id, nil
}

// ListEnrollments returns all enrollments for a student, ordered by term recency.
func (r *PgRepository) ListEnrollments(ctx context.Context, studentID, tenantID string) ([]Enrollment, error) {
	query := `
		SELECT e.id, e.student_id, e.class_id, e.academic_term_id,
		       t.name AS term_name, t.term_number,
		       ay.name AS academic_year,
		       c.display_label AS class_name,
		       e.status, e.created_at::text
		FROM cbc_student_enrollments e
		LEFT JOIN academic_terms t ON t.id = e.academic_term_id
		LEFT JOIN academic_years ay ON ay.id = t.academic_year_id
		LEFT JOIN cbc_classes c ON c.id = e.class_id
		WHERE e.student_id = $1
		ORDER BY ay.start_date DESC, t.term_number DESC
	`
	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("students.Repository.ListEnrollments: %w", err)
	}
	defer rows.Close()

	var enrollments []Enrollment
	for rows.Next() {
		var e Enrollment
		var classID, termName, academicYear, className *string
		var termNumber *int
		err := rows.Scan(
			&e.ID, &e.StudentID, &classID, &e.AcademicTermID,
			&termName, &termNumber,
			&academicYear, &className,
			&e.Status, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("students.Repository.ListEnrollments: scan: %w", err)
		}
		if classID != nil {
			e.ClassID = *classID
		}
		if termName != nil {
			e.TermName = *termName
		}
		if termNumber != nil {
			e.TermNumber = *termNumber
		}
		if academicYear != nil {
			e.AcademicYear = *academicYear
		}
		if className != nil {
			e.ClassName = *className
		}
		enrollments = append(enrollments, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("students.Repository.ListEnrollments: rows: %w", err)
	}

	if enrollments == nil {
		enrollments = []Enrollment{}
	}

	return enrollments, nil
}

// IsEnrolledInTerm checks if a student already has an enrollment for a given term.
func (r *PgRepository) IsEnrolledInTerm(ctx context.Context, studentID, academicTermID, tenantID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM cbc_student_enrollments
			WHERE student_id = $1 AND academic_term_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, studentID, academicTermID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("students.Repository.IsEnrolledInTerm: %w", err)
	}
	return exists, nil
}

// ============================================================================
// Helpers
// ============================================================================

func isNoRows(err error) bool {
	return err != nil && err.Error() == "no rows in result set"
}

func isDuplicateUPI(err error) bool {
	return err != nil && contains(err.Error(), "uq_cbc_students_upi_number")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// compile-time interface check
var _ StudentRepository = (*PgRepository)(nil)

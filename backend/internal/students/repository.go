package students

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// PgRepository implements StudentRepository backed by Postgres.
type PgRepository struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools, logger *zap.SugaredLogger) *PgRepository {
	return &PgRepository{pool: pools.PG, logger: logger}
}

// ─── List ─────────────────────────────────────────────────────────────────

// List returns a paginated list of students enrolled at the given school.
// Supports search (full_name, upi_number, knec_assessment_number, admission_number),
// curriculum filters (education_level, grade_level), and lifecycle filter (enrollment_status).
func (r *PgRepository) List(ctx context.Context, filter ListFilter) ([]Student, int, error) {
	// ── Build WHERE clauses ─────────────────────────────────────────────
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Always scoped to tenant + school
	conditions = append(conditions, fmt.Sprintf("s.tenant_id = $%d", argIdx))
	args = append(args, filter.TenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("s.school_id = $%d", argIdx))
	args = append(args, filter.SchoolID)
	argIdx++

	// ── Search (full_name, upi_number, knec_assessment_number, admission_number) ──
	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf(`(
			s.full_name ILIKE $%d OR
			s.upi_number::text ILIKE $%d OR
			s.knec_assessment_number::text ILIKE $%d OR
			s.admission_number::text ILIKE $%d
		)`, argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, pattern, pattern, pattern, pattern)
		argIdx += 4
	}

	// ── Education Level multi-select ────────────────────────────────────
	if len(filter.EducationLevels) > 0 {
		ors := make([]string, len(filter.EducationLevels))
		for i, el := range filter.EducationLevels {
			ors[i] = fmt.Sprintf("la.education_level::text = $%d", argIdx)
			args = append(args, el)
			argIdx++
		}
		conditions = append(conditions, "("+joinStrings(ors, " OR ")+")")
	}

	// ── Grade Level multi-select ────────────────────────────────────────
	if len(filter.GradeLevels) > 0 {
		ors := make([]string, len(filter.GradeLevels))
		for i, gl := range filter.GradeLevels {
			ors[i] = fmt.Sprintf("c.grade_level::text = $%d", argIdx)
			args = append(args, gl)
			argIdx++
		}
		conditions = append(conditions, "("+joinStrings(ors, " OR ")+")")
	}

	// ── Enrollment Status filter (Lifecycle Group) ──────────────────────
	if filter.EnrollmentStatus != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argIdx))
		args = append(args, filter.EnrollmentStatus)
		argIdx++
	}

	// ── Build full query ────────────────────────────────────────────────
	whereClause := joinStrings(conditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT s.id)
		FROM cbc_students s
		LEFT JOIN LATERAL (
			SELECT e.status
			FROM cbc_student_enrollments e
			WHERE e.student_id = s.id
			ORDER BY e.created_at DESC
			LIMIT 1
		) e ON TRUE
		LEFT JOIN LATERAL (
			SELECT cc.grade_level
			FROM cbc_student_enrollments e2
			JOIN cbc_classes cc ON cc.id = e2.class_id
			WHERE e2.student_id = s.id
			ORDER BY e2.created_at DESC
			LIMIT 1
		) c ON TRUE
		LEFT JOIN LATERAL (
			SELECT la.education_level
			FROM cbc_student_enrollments e3
			JOIN cbc_classes cc2 ON cc2.id = e3.class_id
			JOIN cbc_learning_areas la ON la.grade_level = cc2.grade_level
			WHERE e3.student_id = s.id
			ORDER BY e3.created_at DESC
			LIMIT 1
		) la ON TRUE
		WHERE %s
	`, whereClause)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("students.Repository.List: count: %w", err)
	}

	// ── Data query ─────────────────────────────────────────────────────
	dataQuery := fmt.Sprintf(`
		SELECT s.id, s.full_name, s.gender, s.date_of_birth::text,
		       s.upi_number, s.knec_assessment_number, s.admission_number,
		       c.class_name, c.class_id, s.is_active, s.created_at::text
		FROM cbc_students s
		LEFT JOIN LATERAL (
			SELECT cc.grade_level || ' ' || COALESCE(cs.name, '') AS class_name,
			       cc.id AS class_id, cc.grade_level
			FROM cbc_student_enrollments e
			JOIN cbc_classes cc ON cc.id = e.class_id
			LEFT JOIN cbc_streams cs ON cs.id = cc.stream_id AND cs.tenant_id = cc.tenant_id
			WHERE e.student_id = s.id
			ORDER BY e.created_at DESC
			LIMIT 1
		) c ON TRUE
		LEFT JOIN LATERAL (
			SELECT e.status
			FROM cbc_student_enrollments e
			WHERE e.student_id = s.id
			ORDER BY e.created_at DESC
			LIMIT 1
		) e ON TRUE
		LEFT JOIN LATERAL (
			SELECT la.education_level
			FROM cbc_student_enrollments e3
			JOIN cbc_classes cc2 ON cc2.id = e3.class_id
			JOIN cbc_learning_areas la ON la.grade_level = cc2.grade_level
			WHERE e3.student_id = s.id
			ORDER BY e3.created_at DESC
			LIMIT 1
		) la ON TRUE
		WHERE %s
		ORDER BY s.full_name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("students.Repository.List: query: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var s Student
		var dateOfBirth, upiNumber, knecNumber, admissionNumber, className, classID *string
		err := rows.Scan(
			&s.ID, &s.FullName, &s.Gender, &dateOfBirth,
			&upiNumber, &knecNumber, &admissionNumber,
			&className, &classID, &s.IsActive, &s.CreatedAt,
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
		if admissionNumber != nil {
			s.AdmissionNumber = admissionNumber
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

// GetDetail returns a student with enrollment history and linked parents.
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

	// Fetch linked parents
	linkedParents, err := r.ListLinkedParents(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("students.Repository.GetDetail: linked parents: %w", err)
	}

	return &StudentDetail{
		Student:       *student,
		Enrollments:   enrollments,
		LinkedParents: linkedParents,
	}, nil
}

// ListLinkedParents returns all parents linked to a student.
func (r *PgRepository) ListLinkedParents(ctx context.Context, studentID, tenantID string) ([]LinkedParent, error) {
	const query = `
		SELECT cp.id, u.full_name, u.email, cp.phone_number, sp.relationship, sp.is_primary
		FROM cbc_student_parents sp
		JOIN cbc_parents cp ON cp.id = sp.parent_id
		JOIN users u ON u.id = cp.user_id AND u.tenant_id = $2
		WHERE sp.student_id = $1 AND cp.tenant_id = $2
		ORDER BY sp.is_primary DESC, u.full_name ASC
	`
	rows, err := r.pool.Query(ctx, query, studentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("students.Repository.ListLinkedParents: %w", err)
	}
	defer rows.Close()

	var parents []LinkedParent
	for rows.Next() {
		var lp LinkedParent
		if err := rows.Scan(&lp.ParentID, &lp.FullName, &lp.Email, &lp.PhoneNumber, &lp.Relationship, &lp.IsPrimary); err != nil {
			return nil, fmt.Errorf("students.Repository.ListLinkedParents: scan: %w", err)
		}
		parents = append(parents, lp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("students.Repository.ListLinkedParents: rows: %w", err)
	}
	if parents == nil {
		parents = []LinkedParent{}
	}
	return parents, nil
}

// ─── Create ───────────────────────────────────────────────────────────────

// Create inserts a new student record.
func (r *PgRepository) Create(ctx context.Context, student *Student) (string, error) {
	query := `
		INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, date_of_birth, upi_number, knec_assessment_number, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		student.TenantID,
		student.SchoolID,
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

// CreateBatch inserts multiple student records in a single transaction.
// Returns the IDs of all successfully created students. On any failure,
// the entire batch is rolled back (all-or-nothing).
func (r *PgRepository) CreateBatch(ctx context.Context, students []*Student) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("students.Repository.CreateBatch: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			r.logger.Warnw("students.Repository.CreateBatch: rollback",
				"error", rbErr.Error())
		}
	}()

	query := `
		INSERT INTO cbc_students (tenant_id, full_name, gender, date_of_birth, upi_number, knec_assessment_number, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id
	`

	ids := make([]string, 0, len(students))
	for _, student := range students {
		var id string
		err := tx.QueryRow(ctx, query,
			"",
			student.FullName,
			student.Gender,
			student.DateOfBirth,
			student.UPINumber,
			student.KNECAssessmentNumber,
		).Scan(&id)
		if err != nil {
			if isDuplicateUPI(err) {
				return nil, fmt.Errorf("students.Repository.CreateBatch: %w (UPI: %s)", ErrDuplicateUPI, nullString(student.UPINumber))
			}
			return nil, fmt.Errorf("students.Repository.CreateBatch: %w", err)
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("students.Repository.CreateBatch: commit: %w", err)
	}

	return ids, nil
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

// ─── Delete ────────────────────────────────────────────────────────────────

// Delete hard-deletes a student record (and cascade-enrollments) by ID.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	// Delete enrollments first (cascade)
	_, err := r.pool.Exec(ctx, `DELETE FROM cbc_student_enrollments WHERE student_id = $1`, id)
	if err != nil {
		return fmt.Errorf("students.Repository.Delete: delete enrollments: %w", err)
	}

	const query = `
		DELETE FROM cbc_students
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("students.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("students.Repository.Delete: %w", ErrNotFound)
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
		INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, academic_year_id, status, tenant_id, school_id)
		VALUES ($1, $2, $3, (SELECT academic_year_id FROM academic_terms WHERE id = $3), $4, $5, $6)
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
		SELECT e.id, e.student_id, e.class_id, e.academic_term_id, e.academic_year_id,
		       t.name AS term_name, t.term_number,
		       ay.name AS academic_year,
		       c.grade_level || ' ' || COALESCE(cs.name, '') AS class_name,
		       e.status, e.created_at::text
		FROM cbc_student_enrollments e
		LEFT JOIN academic_terms t ON t.id = e.academic_term_id
		LEFT JOIN academic_years ay ON ay.id = t.academic_year_id
		LEFT JOIN cbc_classes c ON c.id = e.class_id
		LEFT JOIN cbc_streams cs ON cs.id = c.stream_id AND cs.tenant_id = c.tenant_id
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
			&e.ID, &e.StudentID, &classID, &e.AcademicTermID, &e.AcademicYearID,
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
	return err != nil && errors.Is(err, pgx.ErrNoRows)
}

func isDuplicateUPI(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" &&
			(pgErr.ConstraintName == "uq_cbc_students_upi" ||
				pgErr.ConstraintName == "uq_cbc_students_upi_number")
	}
	return false
}

func nullString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func joinStrings(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for i := 1; i < len(elems); i++ {
		result += sep + elems[i]
	}
	return result
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// ============================================================================
// ImportRepository Implementation
// ============================================================================

// ValidateClassExists checks that a class exists and belongs to the given tenant and school.
func (r *PgRepository) ValidateClassExists(ctx context.Context, tenantID, schoolID, classID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM cbc_classes
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, classID, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("students.Repository.ValidateClassExists: %w", err)
	}
	return exists, nil
}

// CheckSchoolAdminMembership verifies the caller has SCHOOL_ADMIN for the school.
func (r *PgRepository) CheckSchoolAdminMembership(ctx context.Context, userID, tenantID, schoolID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM memberships
			WHERE user_id = $1 AND tenant_id = $2 AND school_id = $3 AND role = 'SCHOOL_ADMIN'::user_role AND is_active = true
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("students.Repository.CheckSchoolAdminMembership: %w", err)
	}
	return exists, nil
}

// CheckExistingFieldValues checks which of the provided field values already
// exist in cbc_students for the given tenant/school. Returns separate lists
// of existing values for each field. All values in a single list are distinct.
// Empty/nil inputs return empty results. The query is scoped by tenant and
// school so values from other tenants/schools are never reported.
func (r *PgRepository) CheckExistingFieldValues(ctx context.Context, tenantID, schoolID string,
	admissionNumbers, upiNumbers, knecNumbers []string) (
	existingAdmissionNumbers, existingUPINumbers, existingKnecNumbers []string, err error) {

	// If all inputs are empty, return immediately with empty results
	if len(admissionNumbers) == 0 && len(upiNumbers) == 0 && len(knecNumbers) == 0 {
		return []string{}, []string{}, []string{}, nil
	}

	// Deduplicate inputs to avoid unnecessary array elements in the query
	admSet := uniqueStrings(admissionNumbers)
	upiSet := uniqueStrings(upiNumbers)
	knecSet := uniqueStrings(knecNumbers)

	query := `
		SELECT
			COALESCE(array_agg(DISTINCT admission_number) FILTER (WHERE admission_number = ANY($3)), ARRAY[]::text[]),
			COALESCE(array_agg(DISTINCT upi_number) FILTER (WHERE upi_number = ANY($4)), ARRAY[]::text[]),
			COALESCE(array_agg(DISTINCT knec_assessment_number) FILTER (WHERE knec_assessment_number = ANY($5)), ARRAY[]::text[])
		FROM cbc_students
		WHERE tenant_id = $1 AND school_id = $2
		  AND (
		    (admission_number IS NOT NULL AND admission_number = ANY($3)) OR
		    (upi_number IS NOT NULL AND upi_number = ANY($4)) OR
		    (knec_assessment_number IS NOT NULL AND knec_assessment_number = ANY($5))
		  )
	`

	admPG := make([]string, len(admSet))
	copy(admPG, admSet)
	upiPG := make([]string, len(upiSet))
	copy(upiPG, upiSet)
	knecPG := make([]string, len(knecSet))
	copy(knecPG, knecSet)

	var admResult, upiResult, knecResult []string
	err = r.pool.QueryRow(ctx, query,
		tenantID, schoolID, admPG, upiPG, knecPG,
	).Scan(&admResult, &upiResult, &knecResult)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("students.Repository.CheckExistingFieldValues: %w", err)
	}

	if admResult == nil {
		admResult = []string{}
	}
	if upiResult == nil {
		upiResult = []string{}
	}
	if knecResult == nil {
		knecResult = []string{}
	}

	return admResult, upiResult, knecResult, nil
}

// ─── Batch Enrollments ───────────────────────────────────────────────────

// CreateBatchEnrollments creates multiple enrollment records in a single transaction.
// Skips students already enrolled in the given term/school. Returns IDs of all created records.
func (r *PgRepository) CreateBatchEnrollments(ctx context.Context, enrollments []*Enrollment, tenantID, schoolID string) ([]string, error) {
	if len(enrollments) == 0 {
		return []string{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("students.Repository.CreateBatchEnrollments: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			r.logger.Warnw("students.Repository.CreateBatchEnrollments: rollback",
				"error", rbErr.Error())
		}
	}()

	query := `
		INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, academic_year_id, status, tenant_id, school_id)
		VALUES ($1, $2, $3, (SELECT academic_year_id FROM academic_terms WHERE id = $3), $4, $5, $6)
		ON CONFLICT (student_id, school_id, academic_term_id)
		DO UPDATE SET 
			class_id = EXCLUDED.class_id,
			academic_year_id = EXCLUDED.academic_year_id,
			status = EXCLUDED.status
		RETURNING id
	`

	var ids []string
	for _, enrollment := range enrollments {
		var id string
		err := tx.QueryRow(ctx, query,
			enrollment.StudentID,
			enrollment.ClassID,
			enrollment.AcademicTermID,
			enrollment.Status,
			tenantID,
			schoolID,
		).Scan(&id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Conflict — skip (already enrolled)
				continue
			}
			return nil, fmt.Errorf("students.Repository.CreateBatchEnrollments: insert: %w", err)
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("students.Repository.CreateBatchEnrollments: commit: %w", err)
	}

	return ids, nil
}

// compile-time interface checks
var _ StudentRepository = (*PgRepository)(nil)
var _ ImportRepository = (*PgRepository)(nil)

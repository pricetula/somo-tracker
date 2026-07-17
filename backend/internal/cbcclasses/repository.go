package cbcclasses

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles class database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// List returns a paginated list of classes with student counts.
func (r *PgRepository) List(ctx context.Context, filter ClassListFilter) (*ClassListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM cbc_classes c
		WHERE c.tenant_id = $1
		  AND c.school_id = $2
		  AND c.academic_year_id = $3
	`
	countArgs := []interface{}{filter.TenantID, filter.SchoolID, filter.AcademicYearID}
	argIdx := 4

	if len(filter.GradeLevels) > 0 {
		placeholders := makeInPlaceholders(len(filter.GradeLevels), argIdx)
		countQuery += fmt.Sprintf(" AND c.grade_level::text IN (%s)", placeholders)
		for _, gl := range filter.GradeLevels {
			countArgs = append(countArgs, gl)
		}
		argIdx += len(filter.GradeLevels)
	}
	if len(filter.StreamIDs) > 0 {
		placeholders := makeInPlaceholders(len(filter.StreamIDs), argIdx)
		countQuery += fmt.Sprintf(" AND c.stream_id IN (%s)", placeholders)
		for _, sid := range filter.StreamIDs {
			countArgs = append(countArgs, sid)
		}
		argIdx += len(filter.StreamIDs)
	}
	if filter.Search != "" {
		countQuery += fmt.Sprintf(" AND (c.grade_level::text || ' ' || COALESCE(s.name, '')) ILIKE $%d", argIdx)
		countArgs = append(countArgs, "%"+filter.Search+"%")
	}

	var totalRecords int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&totalRecords)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.List: count: %w", err)
	}

	// Data query with student count per term
	// Use COALESCE(s.name, '') to guard against null stream names
	dataQuery := `
		SELECT
			c.id,
			c.grade_level,
			COALESCE(s.name, '') AS stream_name,
			COALESCE(s.color, '') AS stream_color,
			c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
			c.stream_id,
			COUNT(e.student_id) AS student_count
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		LEFT JOIN cbc_student_enrollments e
			ON e.class_id = c.id AND e.academic_term_id = $4
		WHERE
			c.tenant_id = $1
			AND c.school_id = $2
			AND c.academic_year_id = $3
	`

	dataArgs := []interface{}{
		filter.TenantID,
		filter.SchoolID,
		filter.AcademicYearID,
		filter.AcademicTermID,
	}
	argIdx = 5

	if len(filter.GradeLevels) > 0 {
		placeholders := makeInPlaceholders(len(filter.GradeLevels), argIdx)
		dataQuery += fmt.Sprintf(" AND c.grade_level::text IN (%s)", placeholders)
		for _, gl := range filter.GradeLevels {
			dataArgs = append(dataArgs, gl)
		}
		argIdx += len(filter.GradeLevels)
	}
	if len(filter.StreamIDs) > 0 {
		placeholders := makeInPlaceholders(len(filter.StreamIDs), argIdx)
		dataQuery += fmt.Sprintf(" AND c.stream_id IN (%s)", placeholders)
		for _, sid := range filter.StreamIDs {
			dataArgs = append(dataArgs, sid)
		}
		argIdx += len(filter.StreamIDs)
	}
	if filter.Search != "" {
		dataQuery += fmt.Sprintf(" AND (c.grade_level::text || ' ' || COALESCE(s.name, '')) ILIKE $%d", argIdx)
		dataArgs = append(dataArgs, "%"+filter.Search+"%")
		argIdx++
	}

	dataQuery += fmt.Sprintf(`
		GROUP BY c.id, c.grade_level, s.name, s.color, c.stream_id
		ORDER BY c.grade_level ASC, s.name ASC
		LIMIT $%d OFFSET $%d
	`, argIdx, argIdx+1)

	dataArgs = append(dataArgs, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.List: query: %w", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var cls Class
		if err := rows.Scan(
			&cls.ID,
			&cls.GradeLevel,
			&cls.StreamName,
			&cls.StreamColor,
			&cls.DisplayLabel,
			&cls.StreamID,
			&cls.StudentCount,
		); err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.List: scan: %w", err)
		}
		classes = append(classes, cls)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.List: rows: %w", err)
	}

	if classes == nil {
		classes = []Class{}
	}

	return &ClassListResult{
		Items: classes,
		Total: totalRecords,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

// GetByID retrieves a class by ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Class, error) {
	const query = `
		SELECT c.id, c.grade_level, COALESCE(s.name, '') AS stream_name,
		       COALESCE(s.color, '') AS stream_color,
		       c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
		       c.stream_id
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		WHERE c.id = $1 AND c.tenant_id = $2 AND c.school_id = $3
	`

	var cls Class
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&cls.ID, &cls.GradeLevel, &cls.StreamName, &cls.StreamColor, &cls.DisplayLabel, &cls.StreamID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcclasses.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcclasses.Repository.GetByID: %w", err)
	}
	return &cls, nil
}

// Create inserts a new class and batch-enrolls students.
func (r *PgRepository) Create(ctx context.Context, params CreateClassParams) (*Class, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Create: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.WarnContext(ctx, "cbcclasses.Repository.Create: rollback error",
					slog.String("error", rbErr.Error()),
				)
			}
		}
	}()

	// Step 1: Insert class
	const insertClass = `
		INSERT INTO cbc_classes (tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var classID string
	err = tx.QueryRow(ctx, insertClass,
		params.TenantID, params.SchoolID, params.AcademicYearID,
		params.GradeLevel, params.StreamID,
	).Scan(&classID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Create: insert class: %w", err)
	}

	// Step 2: Batch enroll students
	if len(params.StudentIDs) > 0 {
		const enrollStudents = `
			INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, tenant_id, school_id)
			SELECT unnest($1::uuid[]), $2, $3, $4, $5
			ON CONFLICT (student_id, school_id, academic_term_id)
			DO UPDATE SET class_id = EXCLUDED.class_id
		`
		_, err = tx.Exec(ctx, enrollStudents,
			params.StudentIDs, classID, params.AcademicTermID,
			params.TenantID, params.SchoolID,
		)
		if err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.Create: enroll students: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Create: commit tx: %w", err)
	}

	// Fetch the created class with display_label
	const fetchClass = `
		SELECT c.id, c.grade_level, COALESCE(s.name, '') AS stream_name,
		       COALESCE(s.color, '') AS stream_color,
		       c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
		       c.stream_id
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		WHERE c.id = $1
	`
	var cls Class
	err = r.pool.QueryRow(ctx, fetchClass, classID).Scan(
		&cls.ID, &cls.GradeLevel, &cls.StreamName, &cls.StreamColor, &cls.DisplayLabel, &cls.StreamID,
	)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Create: fetch class: %w", err)
	}

	return &cls, nil
}

// Update performs a differential sync of enrollments and updates class fields.
func (r *PgRepository) Update(ctx context.Context, params UpdateClassParams) (*Class, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Update: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.WarnContext(ctx, "cbcclasses.Repository.Update: rollback error",
					slog.String("error", rbErr.Error()),
				)
			}
		}
	}()

	// Step 1: Remove students no longer in the roster
	if len(params.StudentIDs) > 0 {
		const removeStudents = `
			DELETE FROM cbc_student_enrollments
			WHERE class_id = $1
			  AND academic_term_id = $2
			  AND student_id != ALL($3::uuid[])
		`
		_, err = tx.Exec(ctx, removeStudents, params.ClassID, params.AcademicTermID, params.StudentIDs)
		if err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.Update: remove students: %w", err)
		}
	} else {
		// No incoming students — remove all enrollments for this class + term
		const removeAll = `
			DELETE FROM cbc_student_enrollments
			WHERE class_id = $1 AND academic_term_id = $2
		`
		_, err = tx.Exec(ctx, removeAll, params.ClassID, params.AcademicTermID)
		if err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.Update: remove all students: %w", err)
		}
	}

	// Step 2: Upsert incoming roster
	if len(params.StudentIDs) > 0 {
		const upsertStudents = `
			INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, tenant_id, school_id)
			SELECT unnest($1::uuid[]), $2, $3, $4, $5
			ON CONFLICT (student_id, school_id, academic_term_id)
			DO UPDATE SET class_id = EXCLUDED.class_id
		`
		_, err = tx.Exec(ctx, upsertStudents,
			params.StudentIDs, params.ClassID, params.AcademicTermID,
			params.TenantID, params.SchoolID,
		)
		if err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.Update: upsert students: %w", err)
		}
	}

	// Step 3: Update class fields
	const updateClass = `
		UPDATE cbc_classes
		SET grade_level = $1, stream_id = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err = tx.Exec(ctx, updateClass, params.GradeLevel, params.StreamID, params.ClassID)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Update: update class: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Update: commit tx: %w", err)
	}

	// Fetch updated class
	const fetchClass = `
		SELECT c.id, c.grade_level, COALESCE(s.name, '') AS stream_name,
		       COALESCE(s.color, '') AS stream_color,
		       c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
		       c.stream_id
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		WHERE c.id = $1
	`
	var cls Class
	err = r.pool.QueryRow(ctx, fetchClass, params.ClassID).Scan(
		&cls.ID, &cls.GradeLevel, &cls.StreamName, &cls.StreamColor, &cls.DisplayLabel, &cls.StreamID,
	)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Update: fetch class: %w", err)
	}

	return &cls, nil
}

// BulkDelete removes multiple classes.
func (r *PgRepository) BulkDelete(ctx context.Context, ids []string, tenantID, schoolID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cbcclasses.Repository.BulkDelete: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.WarnContext(ctx, "cbcclasses.Repository.BulkDelete: rollback error",
					slog.String("error", rbErr.Error()),
				)
			}
		}
	}()

	const deleteQuery = `
		DELETE FROM cbc_classes
		WHERE id = ANY($1::uuid[])
		  AND tenant_id = $2
		  AND school_id = $3
	`
	_, err = tx.Exec(ctx, deleteQuery, ids, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("cbcclasses.Repository.BulkDelete: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cbcclasses.Repository.BulkDelete: commit tx: %w", err)
	}
	return nil
}

// ValidateAcademicYear checks that the academic year belongs to the tenant + school.
func (r *PgRepository) ValidateAcademicYear(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM academic_years
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcclasses.Repository.ValidateAcademicYear: %w", err)
	}
	return exists, nil
}

// ValidateAcademicTerm checks that the academic term belongs to the given academic year.
func (r *PgRepository) ValidateAcademicTerm(ctx context.Context, id, academicYearID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM academic_terms
			WHERE id = $1 AND academic_year_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, id, academicYearID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcclasses.Repository.ValidateAcademicTerm: %w", err)
	}
	return exists, nil
}

// ─── GetRoster ───────────────────────────────────────────────────────────

// GetRoster returns a paginated list of students enrolled in a class for the given term.
func (r *PgRepository) GetRoster(ctx context.Context, classID, tenantID, schoolID, academicTermID string, limit, offset int, search string) (*RosterListResult, error) {
	// ── Count query ───────────────────────────────────────────
	countQuery := `
		SELECT COUNT(*)
		FROM cbc_student_enrollments e
		JOIN cbc_students s ON s.id = e.student_id
		WHERE e.class_id = $1
		  AND e.academic_term_id = $2
		  AND e.tenant_id = $3
		  AND e.school_id = $4
		  AND e.status = 'ACTIVE'
		  AND s.is_active = true
	`
	countArgs := []interface{}{classID, academicTermID, tenantID, schoolID}

	if search != "" {
		countQuery += ` AND (s.full_name ILIKE $4 OR s.admission_number ILIKE $4)`
		countArgs = append(countArgs, "%"+search+"%")
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.GetRoster: count: %w", err)
	}

	// ── Data query ────────────────────────────────────────────
	dataQuery := `
		SELECT s.id, s.full_name, s.gender,
		       COALESCE(s.admission_number, '') AS admission_number,
		       COALESCE(s.upi_number, '') AS upi_number,
		       e.created_at::text AS enrolled_at
		FROM cbc_student_enrollments e
		JOIN cbc_students s ON s.id = e.student_id
		WHERE e.class_id = $1
		  AND e.academic_term_id = $2
		  AND e.tenant_id = $3
		  AND e.school_id = $4
		  AND e.status = 'ACTIVE'
		  AND s.is_active = true
	`
	dataArgs := []interface{}{classID, academicTermID, tenantID, schoolID}

	if search != "" {
		dataQuery += ` AND (s.full_name ILIKE $4 OR s.admission_number ILIKE $4)`
		dataArgs = append(dataArgs, "%"+search+"%")
	}

	dataQuery += ` ORDER BY s.full_name ASC LIMIT $` + fmt.Sprintf("%d", len(dataArgs)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(dataArgs)+2)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.GetRoster: %w", err)
	}
	defer rows.Close()

	var entries []RosterEntry
	for rows.Next() {
		var e RosterEntry
		if err := rows.Scan(&e.ID, &e.FullName, &e.Gender, &e.AdmissionNumber, &e.UPINumber, &e.EnrolledAt); err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.GetRoster: scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.GetRoster: rows: %w", err)
	}

	if entries == nil {
		entries = []RosterEntry{}
	}

	return &RosterListResult{
		Items: entries,
		Total: total,
		Page:  offset/limit + 1,
		Limit: limit,
	}, nil
}

// ─── BatchEnrollStudents ──────────────────────────────────────────────────

// BatchEnrollStudents atomically enrolls multiple students into a class for a term.
// If any student is already enrolled in a DIFFERENT class for the same term,
// the entire batch is rolled back and ErrEnrollmentConflict is returned.
// Students already enrolled in THIS class are silently skipped (idempotent).
func (r *PgRepository) BatchEnrollStudents(ctx context.Context, classID, tenantID, schoolID, academicTermID string, studentIDs []string) (int, error) {
	if len(studentIDs) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				slog.WarnContext(ctx, "cbcclasses.Repository.BatchEnrollStudents: rollback error",
					slog.String("error", rbErr.Error()),
				)
			}
		}
	}()

	// Step 1: Check for enrollment conflicts — students already enrolled in
	// a DIFFERENT class for this term (not this class).
	const checkConflicts = `
		SELECT s.id, s.full_name, cc.grade_level || ' ' || COALESCE(cs.name, '') AS current_class
		FROM cbc_student_enrollments e
		JOIN cbc_students s ON s.id = e.student_id
		JOIN cbc_classes cc ON cc.id = e.class_id
		LEFT JOIN cbc_streams cs ON cs.id = cc.stream_id AND cs.tenant_id = cc.tenant_id
		WHERE e.student_id = ANY($1::uuid[])
		  AND e.academic_term_id = $2
		  AND e.class_id != $3
		  AND e.tenant_id = $4
	`
	rows, checkErr := tx.Query(ctx, checkConflicts, studentIDs, academicTermID, classID, tenantID)
	if checkErr != nil {
		return 0, fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: check conflicts: %w", checkErr)
	}
	defer rows.Close()

	type conflictInfo struct {
		id, name, currentClass string
	}
	var conflicts []conflictInfo
	for rows.Next() {
		var c conflictInfo
		if scanErr := rows.Scan(&c.id, &c.name, &c.currentClass); scanErr != nil {
			return 0, fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: scan conflict: %w", scanErr)
		}
		conflicts = append(conflicts, c)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: rows: %w", err)
	}
	rows.Close()

	if len(conflicts) > 0 {
		// Build a descriptive error message
		conflictNames := make([]string, len(conflicts))
		for i, c := range conflicts {
			conflictNames[i] = c.name + " (currently in " + c.currentClass + ")"
		}
		err = fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: %w: conflicts: %s",
			ErrEnrollmentConflict, strings.Join(conflictNames, ", "))
		return 0, err
	}

	// Step 2: Insert enrollments (skip if already enrolled in this class)
	const insertEnrollments = `
		INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, tenant_id, school_id, status)
		SELECT unnest($1::uuid[]), $2, $3, $4, $5, 'ACTIVE'
		ON CONFLICT (student_id, school_id, academic_term_id)
		DO UPDATE SET class_id = EXCLUDED.class_id, status = 'ACTIVE', updated_at = NOW()
		WHERE cbc_student_enrollments.class_id IS DISTINCT FROM EXCLUDED.class_id
	`
	tag, insertErr := tx.Exec(ctx, insertEnrollments, studentIDs, classID, academicTermID, tenantID, schoolID)
	if insertErr != nil {
		return 0, fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: insert: %w", insertErr)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("cbcclasses.Repository.BatchEnrollStudents: commit: %w", err)
	}

	enrolledCount := int(tag.RowsAffected())
	return enrolledCount, nil
}

// ─── UnenrollStudent ──────────────────────────────────────────────────────

// UnenrollStudent removes a single student from a class for the given term.
// Sets class_id to NULL on the enrollment record instead of deleting it,
// preserving attendance history.
// The academic_term_id parameter is REQUIRED to scope the unenrollment to
// a single term — omitting it would unenroll the student from ALL terms
// for this class.
func (r *PgRepository) UnenrollStudent(ctx context.Context, classID, studentID, tenantID, schoolID, academicTermID string) error {
	const query = `
		UPDATE cbc_student_enrollments
		SET class_id = NULL, status = 'SUSPENDED', updated_at = NOW()
		WHERE class_id = $1
		  AND student_id = $2
		  AND tenant_id = $3
		  AND academic_term_id = $4
	`
	tag, err := r.pool.Exec(ctx, query, classID, studentID, tenantID, academicTermID)
	if err != nil {
		return fmt.Errorf("cbcclasses.Repository.UnenrollStudent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cbcclasses.Repository.UnenrollStudent: %w", ErrStudentNotInClass)
	}
	return nil
}

// ─── GetAvailableStudents ─────────────────────────────────────────────────

// GetAvailableStudents returns students NOT enrolled in the given class for the active term,
// with optional search by name, admission_number, or upi_number.
func (r *PgRepository) GetAvailableStudents(ctx context.Context, filter AvailableStudentsFilter) (*AvailableStudentsResponse, error) {
	offset := (filter.Page - 1) * filter.Limit
	if offset < 0 {
		offset = 0
	}

	// ── Count query ──────────────────────────────────────────────────────
	countQuery := `
		SELECT COUNT(*)
		FROM cbc_students s
		WHERE s.tenant_id = $1
		  AND s.school_id = $2
		  AND s.is_active = true
		  AND s.id NOT IN (
		      SELECT e.student_id
		      FROM cbc_student_enrollments e
		      WHERE e.academic_term_id = $3
		        AND e.class_id = $4
		        AND e.tenant_id = $1
		        AND e.status = 'ACTIVE'
		  )
	`
	countArgs := []interface{}{filter.TenantID, filter.SchoolID, filter.AcademicTermID, filter.ClassID}
	argIdx := 5

	if filter.Search != "" {
		countQuery += fmt.Sprintf(` AND (
			s.full_name ILIKE $%d OR
			s.admission_number::text ILIKE $%d OR
			s.upi_number::text ILIKE $%d
		)`, argIdx, argIdx+1, argIdx+2)
		pattern := "%" + filter.Search + "%"
		countArgs = append(countArgs, pattern, pattern, pattern)
	}

	var total int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.GetAvailableStudents: count: %w", err)
	}

	// ── Data query ───────────────────────────────────────────────────────
	dataQuery := `
		SELECT s.id, s.full_name, s.gender,
		       s.admission_number, s.upi_number,
		       cc.display_label AS current_class,
		       cc.id AS current_class_id
		FROM cbc_students s
		LEFT JOIN LATERAL (
			SELECT cc2.grade_level || ' ' || COALESCE(cs2.name, '') AS display_label,
			       cc2.id
			FROM cbc_student_enrollments e2
			JOIN cbc_classes cc2 ON cc2.id = e2.class_id
			LEFT JOIN cbc_streams cs2 ON cs2.id = cc2.stream_id
			WHERE e2.student_id = s.id
			  AND e2.academic_term_id = $3
			  AND e2.status = 'ACTIVE'
			LIMIT 1
		) cc ON TRUE
		WHERE s.tenant_id = $1
		  AND s.school_id = $2
		  AND s.is_active = true
		  AND s.id NOT IN (
		      SELECT e3.student_id
		      FROM cbc_student_enrollments e3
		      WHERE e3.academic_term_id = $3
		        AND e3.class_id = $4
		        AND e3.tenant_id = $1
		        AND e3.status = 'ACTIVE'
		  )
	`
	dataArgs := []interface{}{filter.TenantID, filter.SchoolID, filter.AcademicTermID, filter.ClassID}
	dataArgIdx := 5

	if filter.Search != "" {
		dataQuery += fmt.Sprintf(` AND (
			s.full_name ILIKE $%d OR
			s.admission_number::text ILIKE $%d OR
			s.upi_number::text ILIKE $%d
		)`, dataArgIdx, dataArgIdx+1, dataArgIdx+2)
		pattern := "%" + filter.Search + "%"
		dataArgs = append(dataArgs, pattern, pattern, pattern)
		dataArgIdx += 3
	}

	dataQuery += fmt.Sprintf(`
		ORDER BY s.full_name ASC
		LIMIT $%d OFFSET $%d
	`, dataArgIdx, dataArgIdx+1)
	dataArgs = append(dataArgs, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.GetAvailableStudents: query: %w", err)
	}
	defer rows.Close()

	var students []AvailableStudent
	for rows.Next() {
		var s AvailableStudent
		var admissionNumber, upiNumber, currentClass, currentClassID *string
		if err := rows.Scan(
			&s.ID, &s.FullName, &s.Gender,
			&admissionNumber, &upiNumber,
			&currentClass, &currentClassID,
		); err != nil {
			return nil, fmt.Errorf("cbcclasses.Repository.GetAvailableStudents: scan: %w", err)
		}
		if admissionNumber != nil && *admissionNumber != "" {
			s.AdmissionNumber = admissionNumber
		}
		if upiNumber != nil && *upiNumber != "" {
			s.UPINumber = upiNumber
		}
		s.CurrentClass = currentClass
		s.CurrentClassID = currentClassID
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.GetAvailableStudents: rows: %w", err)
	}

	if students == nil {
		students = []AvailableStudent{}
	}

	return &AvailableStudentsResponse{
		Items: students,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

// ValidateStream checks that the stream belongs to the tenant + school.
func (r *PgRepository) ValidateStream(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_streams
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcclasses.Repository.ValidateStream: %w", err)
	}
	return exists, nil
}

// makeInPlaceholders generates a comma-separated list of pgx placeholders.
// Example: makeInPlaceholders(3, 5) returns "$5, $6, $7".
func makeInPlaceholders(count, startIdx int) string {
	if count == 0 {
		return ""
	}
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", startIdx+i)
	}
	return strings.Join(placeholders, ", ")
}

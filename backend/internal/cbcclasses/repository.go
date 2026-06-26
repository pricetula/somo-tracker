package cbcclasses

import (
	"context"
	"fmt"
	"math"

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

	if filter.GradeLevel != nil {
		countQuery += fmt.Sprintf(" AND c.grade_level = $%d", argIdx)
		countArgs = append(countArgs, *filter.GradeLevel)
		argIdx++
	}
	if filter.StreamID != nil {
		countQuery += fmt.Sprintf(" AND c.stream_id = $%d", argIdx)
		countArgs = append(countArgs, *filter.StreamID)
	}

	var totalRecords int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&totalRecords)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.List: count: %w", err)
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(filter.Limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	// Data query with student count per term
	// Use COALESCE(s.name, '') to guard against null stream names
	dataQuery := `
		SELECT
			c.id,
			c.grade_level,
			COALESCE(s.name, '') AS stream_name,
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

	if filter.GradeLevel != nil {
		dataQuery += fmt.Sprintf(" AND c.grade_level = $%d", argIdx)
		dataArgs = append(dataArgs, *filter.GradeLevel)
		argIdx++
	}
	if filter.StreamID != nil {
		dataQuery += fmt.Sprintf(" AND c.stream_id = $%d", argIdx)
		dataArgs = append(dataArgs, *filter.StreamID)
		argIdx++
	}

	dataQuery += fmt.Sprintf(`
		GROUP BY c.id, c.grade_level, s.name, c.stream_id
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
		Data:         classes,
		TotalRecords: totalRecords,
		CurrentPage:  filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}, nil
}

// GetByID retrieves a class by ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Class, error) {
	const query = `
		SELECT c.id, c.grade_level, COALESCE(s.name, '') AS stream_name,
		       c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
		       c.stream_id
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		WHERE c.id = $1 AND c.tenant_id = $2 AND c.school_id = $3
	`

	var cls Class
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(
		&cls.ID, &cls.GradeLevel, &cls.StreamName, &cls.DisplayLabel, &cls.StreamID,
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
			_ = tx.Rollback(ctx)
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
			ON CONFLICT (student_id, academic_term_id)
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
		       c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
		       c.stream_id
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		WHERE c.id = $1
	`
	var cls Class
	err = r.pool.QueryRow(ctx, fetchClass, classID).Scan(
		&cls.ID, &cls.GradeLevel, &cls.StreamName, &cls.DisplayLabel, &cls.StreamID,
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
			_ = tx.Rollback(ctx)
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
			ON CONFLICT (student_id, academic_term_id)
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
		       c.grade_level || ' ' || COALESCE(s.name, '') AS display_label,
		       c.stream_id
		FROM cbc_classes c
		JOIN cbc_streams s ON s.id = c.stream_id
		WHERE c.id = $1
	`
	var cls Class
	err = r.pool.QueryRow(ctx, fetchClass, params.ClassID).Scan(
		&cls.ID, &cls.GradeLevel, &cls.StreamName, &cls.DisplayLabel, &cls.StreamID,
	)
	if err != nil {
		return nil, fmt.Errorf("cbcclasses.Repository.Update: fetch class: %w", err)
	}

	return &cls, nil
}

// BulkDelete removes multiple classes after ensuring no assessment records exist.
func (r *PgRepository) BulkDelete(ctx context.Context, ids []string, tenantID, schoolID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cbcclasses.Repository.BulkDelete: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
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

// HasAssessmentSessions checks if a specific class has any assessment records.
func (r *PgRepository) HasAssessmentSessions(ctx context.Context, classID, tenantID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM assessment_sessions
			WHERE class_id = $1 AND tenant_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, classID, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcclasses.Repository.HasAssessmentSessions: %w", err)
	}
	return exists, nil
}

// HasAnyAssessmentSessions checks if any of the given classes have assessment records.
func (r *PgRepository) HasAnyAssessmentSessions(ctx context.Context, classIDs []string, tenantID string) (bool, error) {
	if len(classIDs) == 0 {
		return false, nil
	}

	const query = `
		SELECT EXISTS (
			SELECT 1 FROM assessment_sessions
			WHERE class_id = ANY($1::uuid[])
			  AND tenant_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, classIDs, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcclasses.Repository.HasAnyAssessmentSessions: %w", err)
	}
	return exists, nil
}

// ValidateAcademicYear checks that the academic year belongs to the tenant + school.
func (r *PgRepository) ValidateAcademicYear(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM academic_years
			WHERE id = $1 AND tenant_id = $2 AND school_id = $3 AND deleted_at IS NULL
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
			WHERE id = $1 AND academic_year_id = $2 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, id, academicYearID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcclasses.Repository.ValidateAcademicTerm: %w", err)
	}
	return exists, nil
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

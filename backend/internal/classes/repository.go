package classes

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository handles database operations for classes.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pools *database.Pools) *Repository {
	return &Repository{pool: pools.PG}
}

// GetPrimarySchoolID returns the primary active school for a tenant.
func (r *Repository) GetPrimarySchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	const membershipQuery = `
		SELECT school_id FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		LIMIT 1
	`
	var schoolID string
	err := r.pool.QueryRow(ctx, membershipQuery, tenantID, userID).Scan(&schoolID)
	if err == nil {
		return schoolID, nil
	}

	const fallbackQuery = `
		SELECT id FROM schools
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at ASC
		LIMIT 1
	`
	err = r.pool.QueryRow(ctx, fallbackQuery, tenantID).Scan(&schoolID)
	if err != nil {
		return "", fmt.Errorf("no active school found for tenant %s: %w", tenantID, err)
	}
	return schoolID, nil
}

// GetCurrentAcademicYear returns the current academic year for a school.
type academicYearInfo struct {
	ID                string
	EducationSystemID string
}

func (r *Repository) GetCurrentAcademicYear(ctx context.Context, schoolID, tenantID string) (*academicYearInfo, error) {
	const query = `
		SELECT ay.id, s.education_system_id
		FROM academic_years ay
		JOIN schools s ON s.id = ay.school_id AND s.tenant_id = ay.tenant_id
		WHERE ay.school_id = $1 AND ay.tenant_id = $2 AND ay.is_current = true
		LIMIT 1
	`
	var info academicYearInfo
	err := r.pool.QueryRow(ctx, query, schoolID, tenantID).Scan(&info.ID, &info.EducationSystemID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get current academic year: %w", err)
	}
	return &info, nil
}

// GetSchoolGrades returns all grade records for the school's education system,
// ordered by sequence_order ascending.
type gradeInfo struct {
	ID   string
	Name string
}

func (r *Repository) GetSchoolGrades(ctx context.Context, schoolID, tenantID string) ([]gradeInfo, error) {
	// First get the school's education_system_id
	const edQuery = `SELECT education_system_id FROM schools WHERE id = $1 AND tenant_id = $2`
	var edID string
	if err := r.pool.QueryRow(ctx, edQuery, schoolID, tenantID).Scan(&edID); err != nil {
		return nil, fmt.Errorf("get school education system: %w", err)
	}

	// Then fetch all grades for that system, ordered by sequence_order
	const gradeQuery = `
		SELECT id, name FROM grades
		WHERE education_system_id = $1
		ORDER BY sequence_order ASC
	`
	rows, err := r.pool.Query(ctx, gradeQuery, edID)
	if err != nil {
		return nil, fmt.Errorf("get grades: %w", err)
	}
	defer rows.Close()

	var grades []gradeInfo
	for rows.Next() {
		var g gradeInfo
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("scan grade: %w", err)
		}
		grades = append(grades, g)
	}
	return grades, nil
}

// ListClasses returns filtered classes for the school's current academic year.
func (r *Repository) ListClasses(ctx context.Context, schoolID, tenantID string, params ListClassesParams) ([]Class, error) {
	query := `
		SELECT c.id, c.tenant_id, c.school_id, c.academic_year_id,
		       c.education_system_id, c.grade_id, c.name, c.stream, c.is_active
		FROM classes c
		JOIN academic_years ay ON ay.id = c.academic_year_id
		WHERE c.school_id = $1 AND c.tenant_id = $2 AND ay.is_current = true
	`
	args := []any{schoolID, tenantID}
	argIdx := 3

	if len(params.GradeIDs) > 0 {
		placeholders := make([]string, 0, len(params.GradeIDs))
		for _, gid := range params.GradeIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
			args = append(args, gid)
			argIdx++
		}
		query += fmt.Sprintf(" AND c.grade_id IN (%s)", joinStrings(placeholders, ", "))
	}

	if params.Search != "" {
		query += fmt.Sprintf(" AND c.name ILIKE $%d", argIdx)
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}

	if params.IsActive != nil {
		query += fmt.Sprintf(" AND c.is_active = $%d", argIdx)
		args = append(args, *params.IsActive)
	}

	query += ` ORDER BY c.name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var c Class
		var stream *string
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.SchoolID, &c.AcademicYearID,
			&c.EducationSystemID, &c.GradeID, &c.Name, &stream, &c.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}
		if stream != nil {
			c.Stream = *stream
		}
		classes = append(classes, c)
	}
	return classes, nil
}

// BeginTx starts a transaction.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// ListGrades returns all grades for the school's education system.
func (r *Repository) ListGrades(ctx context.Context, schoolID, tenantID string) ([]GradeInfo, error) {
	const query = `
		SELECT g.id, g.name, g.sequence_order
		FROM grades g
		JOIN schools s ON s.education_system_id = g.education_system_id
		WHERE s.id = $1 AND s.tenant_id = $2
		ORDER BY g.sequence_order ASC
	`
	rows, err := r.pool.Query(ctx, query, schoolID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list grades: %w", err)
	}
	defer rows.Close()

	var grades []GradeInfo
	for rows.Next() {
		var g GradeInfo
		if err := rows.Scan(&g.ID, &g.Name, &g.SequenceOrder); err != nil {
			return nil, fmt.Errorf("scan grade: %w", err)
		}
		grades = append(grades, g)
	}
	return grades, nil
}

// BulkInsertClasses inserts all generated class records in a single batch.
func (r *Repository) BulkInsertClasses(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, schoolID, academicYearID string,
	inputs []classInput,
) ([]Class, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// Resolve education_system_id from the school
	var edID string
	const edQuery = `SELECT education_system_id FROM schools WHERE id = $1 AND tenant_id = $2`
	if err := tx.QueryRow(ctx, edQuery, schoolID, tenantID).Scan(&edID); err != nil {
		return nil, fmt.Errorf("get school education system: %w", err)
	}

	// Batch insert using a single multi-row INSERT ... RETURNING
	batchSize := len(inputs)
	rows := make([][]any, batchSize)
	inserted := make([]Class, 0, batchSize)

	const insertSQL = `
		INSERT INTO classes (tenant_id, school_id, academic_year_id, education_system_id, grade_id, name, stream, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id, tenant_id, school_id, academic_year_id, education_system_id, grade_id, name, stream, is_active, NOW()
	`

	for i, in := range inputs {
		rows[i] = []any{tenantID, schoolID, academicYearID, edID, in.GradeID, in.Name, in.Stream}
	}

	for i, args := range rows {
		var c Class
		var stream *string
		var createdAt any
		err := tx.QueryRow(ctx, insertSQL, args...).Scan(
			&c.ID, &c.TenantID, &c.SchoolID, &c.AcademicYearID,
			&c.EducationSystemID, &c.GradeID, &c.Name, &stream, &c.IsActive, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert class %d (%s): %w", i, inputs[i].Name, err)
		}
		if stream != nil {
			c.Stream = *stream
		}
		inserted = append(inserted, c)
	}

	return inserted, nil
}

// joinStrings joins string slices with a separator (replaces strings.Join for clarity in SQL building).
func joinStrings(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for _, e := range elems[1:] {
		result += sep + e
	}
	return result
}

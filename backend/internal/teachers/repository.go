package teachers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles teacher database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ListBySchool returns paginated teachers for a given school.
// When includeInactive is true, both active and inactive memberships are returned.
func (r *PgRepository) ListBySchool(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
	// Build the WHERE clause for active/inactive filter
	activeFilter := "TRUE"
	if !includeInactive {
		activeFilter = "m.is_active = true"
	}

	// Count total
	countArgs := []interface{}{tenantID, schoolID}
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = 'TEACHER'
		  AND %s
	`, activeFilter)

	argIdx := 3
	if search != "" {
		pattern := "%" + search + "%"
		countQuery += fmt.Sprintf(` AND (u.full_name ILIKE $%d OR u.email ILIKE $%d)`, argIdx, argIdx+1)
		countArgs = append(countArgs, pattern, pattern)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("teachers.Repository.Count: %w", err)
	}

	// Fetch data with teacher-specific fields
	dataArgs := []interface{}{tenantID, schoolID}
	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.email, u.full_name,
		       u.tsc_number, u.knec_panel_assessor_id,
		       cct.teacher_role,
		       m.is_active, m.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN LATERAL (
			SELECT teacher_role::text
			FROM cbc_class_teachers
			WHERE user_id = u.id
			  AND tenant_id = $1
			LIMIT 1
		) cct ON TRUE
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = 'TEACHER'
		  AND %s
	`, activeFilter)

	dataArgIdx := 3
	if search != "" {
		pattern := "%" + search + "%"
		dataQuery += fmt.Sprintf(` AND (u.full_name ILIKE $%d OR u.email ILIKE $%d)`, dataArgIdx, dataArgIdx+1)
		dataArgs = append(dataArgs, pattern, pattern)
		dataArgIdx += 2
	}

	dataQuery += fmt.Sprintf(` ORDER BY u.full_name LIMIT $%d OFFSET $%d`, dataArgIdx, dataArgIdx+1)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("teachers.Repository.ListBySchool: %w", err)
	}
	defer rows.Close()

	var teachers []Teacher
	for rows.Next() {
		var t Teacher
		if err := rows.Scan(
			&t.ID, &t.Email, &t.FullName,
			&t.TSCNumber, &t.KNECPanelAssessor,
			&t.TeacherRole,
			&t.IsActive, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("teachers.Repository.Scan: %w", err)
		}
		teachers = append(teachers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("teachers.Repository.Rows: %w", err)
	}

	if teachers == nil {
		teachers = []Teacher{}
	}

	return teachers, total, nil
}

// ToggleActive sets the is_active flag on a teacher's membership.
func (r *PgRepository) ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
	const query = `
		UPDATE memberships
		SET is_active = $1
		WHERE tenant_id = $2 AND school_id = $3 AND user_id = $4 AND role::text = 'TEACHER'
	`

	tag, err := r.pool.Exec(ctx, query, isActive, tenantID, schoolID, userID)
	if err != nil {
		return fmt.Errorf("teachers.Repository.ToggleActive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("teachers.Repository.ToggleActive: %w", ErrNotFound)
	}

	return nil
}

// GetActiveSchoolID returns the active school ID for a user in a tenant.
func (r *PgRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	const query = `
		SELECT school_id FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		ORDER BY
			CASE role
				WHEN 'SCHOOL_ADMIN'::user_role THEN 1
				WHEN 'TEACHER'::user_role THEN 2
				WHEN 'NURSE'::user_role THEN 3
				WHEN 'FINANCE'::user_role THEN 4
			END
		LIMIT 1
	`

	var schoolID string
	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(&schoolID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("teachers.Repository.GetActiveSchoolID: %w", ErrNotFound)
		}
		return "", fmt.Errorf("teachers.Repository.GetActiveSchoolID: %w", err)
	}
	return schoolID, nil
}

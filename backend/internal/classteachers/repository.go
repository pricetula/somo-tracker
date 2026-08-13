package classteachers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles class teacher database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// isUniqueViolation checks for PostgreSQL unique violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// Create inserts a new class teacher assignment.
func (r *PgRepository) Create(ctx context.Context, params CreateClassTeacherParams) (string, error) {
	const query = `
		INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, teacher_role)
		VALUES ($1, $2, $3, $4, $5::teacher_role)
		RETURNING id
	`
	var id string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query,
		params.TenantID,
		params.ClassID,
		params.UserID,
		params.LearningAreaID,
		params.TeacherRole,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return "", fmt.Errorf("classteachers.Repository.Create: %w", ErrAlreadyExists)
		}
		return "", fmt.Errorf("classteachers.Repository.Create: %w", err)
	}
	return id, nil
}

// GetByID retrieves a class teacher assignment by ID.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
	const query = `
		SELECT ct.id, ct.tenant_id, ct.class_id, ct.user_id,
		       u.full_name AS teacher_name,
		       ct.learning_area_id, la.name AS learning_area,
		       ct.teacher_role::text, ct.created_at
		FROM cbc_class_teachers ct
		JOIN users u ON u.id = ct.user_id
		LEFT JOIN cbc_learning_areas la ON la.id = ct.learning_area_id
		WHERE ct.id = $1 AND ct.tenant_id = $2
	`
	var ct ClassTeacher
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID).Scan(
		&ct.ID, &ct.TenantID, &ct.ClassID, &ct.UserID,
		&ct.TeacherName,
		&ct.LearningAreaID, &ct.LearningArea,
		&ct.TeacherRole, &ct.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("classteachers.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("classteachers.Repository.GetByID: %w", err)
	}
	return &ct, nil
}

// ListByClass returns all teacher assignments for a given class.
func (r *PgRepository) ListByClass(ctx context.Context, classID, tenantID string) ([]ClassTeacher, error) {
	const query = `
		SELECT ct.id, ct.tenant_id, ct.class_id, ct.user_id,
		       u.full_name AS teacher_name,
		       ct.learning_area_id, la.name AS learning_area,
		       ct.teacher_role::text, ct.created_at
		FROM cbc_class_teachers ct
		JOIN users u ON u.id = ct.user_id
		LEFT JOIN cbc_learning_areas la ON la.id = ct.learning_area_id
		WHERE ct.class_id = $1 AND ct.tenant_id = $2
		ORDER BY ct.teacher_role, u.full_name
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, classID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Repository.ListByClass: %w", err)
	}
	defer rows.Close()

	var items []ClassTeacher
	for rows.Next() {
		var ct ClassTeacher
		if err := rows.Scan(
			&ct.ID, &ct.TenantID, &ct.ClassID, &ct.UserID,
			&ct.TeacherName,
			&ct.LearningAreaID, &ct.LearningArea,
			&ct.TeacherRole, &ct.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("classteachers.Repository.Scan: %w", err)
		}
		items = append(items, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("classteachers.Repository.Rows: %w", err)
	}
	if items == nil {
		items = []ClassTeacher{}
	}
	return items, nil
}

// ListByTeacher returns all class assignments for a given teacher.
func (r *PgRepository) ListByTeacher(ctx context.Context, userID, tenantID string) ([]ClassTeacher, error) {
	const query = `
		SELECT ct.id, ct.tenant_id, ct.class_id, ct.user_id,
		       u.full_name AS teacher_name,
		       ct.learning_area_id, la.name AS learning_area,
		       ct.teacher_role::text, ct.created_at
		FROM cbc_class_teachers ct
		JOIN users u ON u.id = ct.user_id
		LEFT JOIN cbc_learning_areas la ON la.id = ct.learning_area_id
		WHERE ct.user_id = $1 AND ct.tenant_id = $2
		ORDER BY ct.created_at DESC
	`
	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Repository.ListByTeacher: %w", err)
	}
	defer rows.Close()

	var items []ClassTeacher
	for rows.Next() {
		var ct ClassTeacher
		if err := rows.Scan(
			&ct.ID, &ct.TenantID, &ct.ClassID, &ct.UserID,
			&ct.TeacherName,
			&ct.LearningAreaID, &ct.LearningArea,
			&ct.TeacherRole, &ct.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("classteachers.Repository.Scan: %w", err)
		}
		items = append(items, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("classteachers.Repository.Rows: %w", err)
	}
	if items == nil {
		items = []ClassTeacher{}
	}
	return items, nil
}

// Delete removes a class teacher assignment.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID string) error {
	const query = `DELETE FROM cbc_class_teachers WHERE id = $1 AND tenant_id = $2`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("classteachers.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("classteachers.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// CountPrimaryForClass returns the number of primary teachers assigned to a class.
func (r *PgRepository) CountPrimaryForClass(ctx context.Context, classID, tenantID string) (int, error) {
	const query = `
		SELECT COUNT(*) FROM cbc_class_teachers
		WHERE class_id = $1 AND tenant_id = $2 AND teacher_role = 'PRIMARY_CLASS_TEACHER'
	`
	var count int
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, classID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("classteachers.Repository.CountPrimaryForClass: %w", err)
	}
	return count, nil
}

// ExistsForSubject checks if a teacher is already assigned to a subject in a class.
func (r *PgRepository) ExistsForSubject(ctx context.Context, classID, userID, learningAreaID, tenantID string) (bool, error) {
	const query = `
		SELECT COUNT(*) FROM cbc_class_teachers
		WHERE class_id = $1 AND user_id = $2 AND learning_area_id = $3 AND tenant_id = $4
	`
	var count int
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, classID, userID, learningAreaID, tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("classteachers.Repository.ExistsForSubject: %w", err)
	}
	return count > 0, nil
}

package curriculum

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles learning area database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// Create inserts a new cbc_learning_area and returns its ID.
func (r *PgRepository) Create(ctx context.Context, params CreateLearningAreaParams) (string, error) {
	const query = `
		INSERT INTO cbc_learning_areas (tenant_id, school_id, name, code, education_level)
		VALUES ($1, $2, $3, $4, $5::cbc_education_level)
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.SchoolID,
		params.Name,
		params.Code,
		params.EducationLevel,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("curriculum.Repository.Create: %w", err)
	}
	return id, nil
}

// GetByID retrieves a single learning area by ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*LearningArea, error) {
	const query = `
		SELECT id, tenant_id, school_id, name, code, education_level::text
		FROM cbc_learning_areas
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var la LearningArea
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).
		Scan(&la.ID, &la.TenantID, &la.SchoolID, &la.Name, &la.Code, &la.EducationLevel)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("curriculum.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("curriculum.Repository.GetByID: %w", err)
	}
	return &la, nil
}

// List returns all learning areas for the given tenant and school,
// optionally filtered by education_level.
func (r *PgRepository) List(ctx context.Context, tenantID, schoolID string, educationLevel *string) ([]LearningArea, error) {
	query := `
		SELECT id, tenant_id, school_id, name, code, education_level::text
		FROM cbc_learning_areas
		WHERE tenant_id = $1 AND school_id = $2
	`
	args := []interface{}{tenantID, schoolID}

	if educationLevel != nil && *educationLevel != "" {
		args = append(args, *educationLevel)
		query += fmt.Sprintf(" AND education_level = $%d::cbc_education_level", len(args))
	}

	query += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("curriculum.Repository.List: %w", err)
	}
	defer rows.Close()

	var areas []LearningArea
	for rows.Next() {
		var la LearningArea
		if err := rows.Scan(&la.ID, &la.TenantID, &la.SchoolID, &la.Name, &la.Code, &la.EducationLevel); err != nil {
			return nil, fmt.Errorf("curriculum.Repository.List: scan: %w", err)
		}
		areas = append(areas, la)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("curriculum.Repository.List: rows: %w", err)
	}

	if areas == nil {
		areas = []LearningArea{}
	}

	return areas, nil
}

// Update modifies learning area fields. Only non-nil fields are applied.
func (r *PgRepository) Update(ctx context.Context, params UpdateLearningAreaParams) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *params.Name)
		argIdx++
	}
	if params.Code != nil {
		setClauses = append(setClauses, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *params.Code)
		argIdx++
	}
	if params.EducationLevel != nil {
		setClauses = append(setClauses, fmt.Sprintf("education_level = $%d::cbc_education_level", argIdx))
		args = append(args, *params.EducationLevel)
		argIdx++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("curriculum.Repository.Update: %w", ErrInvalidInput)
	}

	args = append(args, params.ID, params.TenantID, params.SchoolID)
	query := fmt.Sprintf(`
		UPDATE cbc_learning_areas
		SET %s
		WHERE id = $%d AND tenant_id = $%d AND school_id = $%d
	`, joinClauses(setClauses, ", "), argIdx, argIdx+1, argIdx+2)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("curriculum.Repository.Update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.Update: %w", ErrNotFound)
	}

	return nil
}

// Delete removes a learning area by ID, scoped to tenant + school.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		DELETE FROM cbc_learning_areas
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("curriculum.Repository.Delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("curriculum.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// joinClauses joins strings with a separator. Helper for dynamic SET clauses.
func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	result := clauses[0]
	for _, c := range clauses[1:] {
		result += sep + c
	}
	return result
}

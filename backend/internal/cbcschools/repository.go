package cbcschools

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles school database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// Create inserts a new cbc_school and returns its ID.
func (r *PgRepository) Create(ctx context.Context, tenantID string, name string) (string, error) {
	const query = `
		INSERT INTO cbc_schools (tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, '', '', 'Public')
		RETURNING id
	`
	var id string
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("cbcschools.Repository.Create: %w", err)
	}
	return id, nil
}

// GetByID retrieves a school by its ID.
func (r *PgRepository) GetByID(ctx context.Context, id string) (*School, error) {
	const query = `
		SELECT id, tenant_id, name, created_at
		FROM cbc_schools
		WHERE id = $1
	`
	var s School
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.TenantID, &s.Name, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcschools.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcschools.Repository.GetByID: %w", err)
	}
	return &s, nil
}

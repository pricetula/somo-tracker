package cbcstreams

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles stream database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// List returns all streams for the given tenant and school.
func (r *PgRepository) List(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
	const query = `
		SELECT id, name, created_at, updated_at
		FROM cbc_streams
		WHERE tenant_id = $1 AND school_id = $2
		ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Repository.List: query: %w", err)
	}
	defer rows.Close()

	var streams []Stream
	for rows.Next() {
		var s Stream
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("cbcstreams.Repository.List: scan: %w", err)
		}
		streams = append(streams, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cbcstreams.Repository.List: rows: %w", err)
	}

	if streams == nil {
		streams = []Stream{}
	}

	return streams, nil
}

// compile-time interface check
var _ Repository = (*PgRepository)(nil)

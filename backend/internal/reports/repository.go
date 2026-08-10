package reports

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository defines the contract for reports data access.
type Repository interface {
	// Ping verifies the database connection is alive.
	Ping(ctx context.Context) error
}

// PgRepository implements Repository using PostgreSQL.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) Repository {
	return &PgRepository{pool: pools.PG}
}

// Ping checks the database connection.
func (r *PgRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

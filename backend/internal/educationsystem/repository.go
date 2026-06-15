package educationsystem

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository provides data access for education systems.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pools *database.Pools) *Repository {
	return &Repository{pool: pools.PG}
}

// ListAll returns every education system ordered by name.
func (r *Repository) ListAll(ctx context.Context) ([]EducationSystem, error) {
	const query = `SELECT id, name, country_code FROM education_systems ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list education systems: %w", err)
	}
	defer rows.Close()

	var systems []EducationSystem
	for rows.Next() {
		var s EducationSystem
		if err := rows.Scan(&s.ID, &s.Name, &s.CountryCode); err != nil {
			return nil, fmt.Errorf("scan education system: %w", err)
		}
		systems = append(systems, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if systems == nil {
		systems = []EducationSystem{}
	}
	return systems, nil
}

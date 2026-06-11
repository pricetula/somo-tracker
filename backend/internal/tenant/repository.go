package tenant

import "somotracker/backend/internal/database"

// Repository defines the interface for tenant data access.
// Placeholder — to be expanded when schema is finalised.
type Repository any

// SqlcRepository is the concrete PostgreSQL-backed implementation of Repository.
type SqlcRepository struct {
	pools *database.Pools
}

// NewRepository creates a new SqlcRepository.
func NewRepository(pools *database.Pools) *SqlcRepository {
	return &SqlcRepository{pools: pools}
}

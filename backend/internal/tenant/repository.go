package tenant

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// SqlcRepository is the concrete PostgreSQL-backed implementation of Repository.
type SqlcRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new SqlcRepository.
func NewRepository(pools *database.Pools) *SqlcRepository {
	return &SqlcRepository{pool: pools.PG}
}

// ExistsByName checks if a tenant exists with the given name.
func (r *SqlcRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM tenants WHERE name = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check tenant by name: %w", err)
	}
	return exists, nil
}

// ExistsBySlug checks if a tenant exists with the given slug.
func (r *SqlcRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check tenant by slug: %w", err)
	}
	return exists, nil
}

// Create inserts a new tenant and returns it.
func (r *SqlcRepository) Create(ctx context.Context, name, slug string) (*Tenant, error) {
	const query = `
		INSERT INTO tenants (name, slug, stytch_org_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
		RETURNING id, name, slug, created_at
	`
	// Use the slug itself as a placeholder stytch_org_id for admin-created tenants.
	// Real Stytch-backed tenants get their org ID from Stytch during registration.
	stytchOrgID := "admin_" + slug

	var t Tenant
	err := r.pool.QueryRow(ctx, query, name, slug, stytchOrgID).Scan(
		&t.ID, &t.Name, &t.Slug, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}
	return &t, nil
}

// GetByID retrieves a tenant by ID.
func (r *SqlcRepository) GetByID(ctx context.Context, id string) (*Tenant, error) {
	const query = `SELECT id, name, slug, created_at FROM tenants WHERE id = $1`
	var t Tenant
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return &t, nil
}

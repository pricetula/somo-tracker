package activeschool

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles member_active_school database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// Upsert inserts or updates the active school for a user.
// Uses the upsert pattern documented in the migration:
//
//	INSERT INTO member_active_school (user_id, tenant_id, school_id, switched_at)
//	VALUES ($1, $2, $3, NOW())
//	ON CONFLICT (user_id) DO UPDATE
//	  SET school_id   = EXCLUDED.school_id,
//	      tenant_id   = EXCLUDED.tenant_id,
//	      switched_at = NOW();
func (r *PgRepository) Upsert(ctx context.Context, tenantID, userID, schoolID string) error {
	const query = `
		INSERT INTO member_active_school (user_id, tenant_id, school_id, switched_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE
			SET school_id   = EXCLUDED.school_id,
			    tenant_id   = EXCLUDED.tenant_id,
			    switched_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("activeschool.Repository.Upsert: %w", err)
	}
	return nil
}

// GetActiveSchoolID returns the active school ID for a user in a tenant.
func (r *PgRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	const query = `
		SELECT school_id
		FROM member_active_school
		WHERE user_id = $1 AND tenant_id = $2
	`
	var schoolID string
	err := r.pool.QueryRow(ctx, query, userID, tenantID).Scan(&schoolID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("activeschool.Repository.GetActiveSchoolID: %w", ErrNotFound)
		}
		return "", fmt.Errorf("activeschool.Repository.GetActiveSchoolID: %w", err)
	}
	return schoolID, nil
}

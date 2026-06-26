package cbcstreams

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

// List returns all streams for the given tenant and school, ordered by name.
func (r *PgRepository) List(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
	const query = `
		SELECT id, name, created_at, updated_at
		FROM cbc_streams
		WHERE tenant_id = $1 AND school_id = $2
		ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Repository.List: %w", err)
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

// GetByID retrieves a single stream by ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
	const query = `
		SELECT id, name, created_at, updated_at
		FROM cbc_streams
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	var s Stream
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcstreams.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcstreams.Repository.GetByID: %w", err)
	}
	return &s, nil
}

// Create inserts a new stream and returns it.
func (r *PgRepository) Create(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
	const query = `
		INSERT INTO cbc_streams (tenant_id, school_id, name)
		VALUES ($1, $2, $3)
		RETURNING id, name, created_at, updated_at
	`

	var s Stream
	err := r.pool.QueryRow(ctx, query, tenantID, schoolID, name).
		Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Repository.Create: %w", err)
	}
	return &s, nil
}

// Update updates a stream's name and returns the updated stream.
// Only updates if the stream belongs to the given tenant + school.
func (r *PgRepository) Update(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
	const query = `
		UPDATE cbc_streams
		SET name = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND school_id = $4
		RETURNING id, name, created_at, updated_at
	`

	var s Stream
	err := r.pool.QueryRow(ctx, query, name, id, tenantID, schoolID).
		Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcstreams.Repository.Update: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcstreams.Repository.Update: %w", err)
	}
	return &s, nil
}

// Delete removes a stream by ID, scoped to tenant + school.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	const query = `
		DELETE FROM cbc_streams
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`

	result, err := r.pool.Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("cbcstreams.Repository.Delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("cbcstreams.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// HasReferencingClasses checks whether any cbc_classes row references this stream.
func (r *PgRepository) HasReferencingClasses(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM cbc_classes
			WHERE stream_id = $1 AND tenant_id = $2 AND school_id = $3
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, id, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcstreams.Repository.HasReferencingClasses: %w", err)
	}
	return exists, nil
}

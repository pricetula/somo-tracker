package cbcstreams

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		SELECT id, name, color, created_at, updated_at
		FROM cbc_streams
		WHERE tenant_id = $1 AND school_id = $2
		ORDER BY name ASC
	`

	rows, err := database.FromContext(ctx, r.pool).Query(ctx, query, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Repository.List: query: %w", err)
	}
	defer rows.Close()

	var streams []Stream
	for rows.Next() {
		var s Stream
		if err := rows.Scan(&s.ID, &s.Name, &s.Color, &s.CreatedAt, &s.UpdatedAt); err != nil {
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

// Create inserts a new stream and returns it.
func (r *PgRepository) Create(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error) {
	const query = `
		INSERT INTO cbc_streams (tenant_id, school_id, name, color)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, color, created_at, updated_at
	`
	var s Stream
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID, schoolID, name, color).Scan(&s.ID, &s.Name, &s.Color, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("cbcstreams.Repository.Create: %w", ErrAlreadyExists)
		}
		return nil, fmt.Errorf("cbcstreams.Repository.Create: %w", err)
	}
	return &s, nil
}

// GetByID retrieves a stream by ID, scoped to tenant + school.
func (r *PgRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
	const query = `
		SELECT id, name, color, created_at, updated_at
		FROM cbc_streams
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	var s Stream
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(&s.ID, &s.Name, &s.Color, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcstreams.Repository.GetByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("cbcstreams.Repository.GetByID: %w", err)
	}
	return &s, nil
}

// Update modifies a stream's name and returns the updated record.
func (r *PgRepository) Update(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error) {
	const query = `
		UPDATE cbc_streams
		SET name = $1, color = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4 AND school_id = $5
		RETURNING id, name, color, created_at, updated_at
	`
	var s Stream
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, name, color, id, tenantID, schoolID).Scan(&s.ID, &s.Name, &s.Color, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cbcstreams.Repository.Update: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("cbcstreams.Repository.Update: %w", ErrAlreadyExists)
		}
		return nil, fmt.Errorf("cbcstreams.Repository.Update: %w", err)
	}
	return &s, nil
}

// HasActiveEnrollments checks whether any class referencing this stream has
// active student enrollments (status = 'ACTIVE') in the current term.
func (r *PgRepository) HasActiveEnrollments(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM cbc_classes c
			JOIN cbc_student_enrollments e ON e.class_id = c.id
			WHERE c.stream_id = $1
			  AND c.tenant_id = $2
			  AND c.school_id = $3
			  AND e.status = 'ACTIVE'
		)
	`
	var exists bool
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, id, tenantID, schoolID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("cbcstreams.Repository.HasActiveEnrollments: %w", err)
	}
	return exists, nil
}

// Delete removes a stream by ID. Returns ErrStreamHasActiveEnrollments when
// the stream has classes with active student enrollments — the transaction is
// rolled back and the database is left untouched.
func (r *PgRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	// Check for active enrollments first — block deletion if found.
	hasActive, err := r.HasActiveEnrollments(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("cbcstreams.Repository.Delete: %w", err)
	}
	if hasActive {
		return fmt.Errorf("cbcstreams.Repository.Delete: %w", ErrStreamHasActiveEnrollments)
	}

	const query = `
		DELETE FROM cbc_streams
		WHERE id = $1 AND tenant_id = $2 AND school_id = $3
	`
	tag, err := database.FromContext(ctx, r.pool).Exec(ctx, query, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("cbcstreams.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cbcstreams.Repository.Delete: %w", ErrNotFound)
	}
	return nil
}

// isUniqueViolation checks if an error is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// compile-time interface check
var _ Repository = (*PgRepository)(nil)

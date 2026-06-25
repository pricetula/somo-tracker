package invitations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles invitation database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ListInvitations returns paginated invitations with optional filters.
func (r *PgRepository) ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
	// Build count query
	countQuery := `SELECT COUNT(*) FROM invitations WHERE tenant_id = $1 AND school_id = $2`
	dataQuery := `SELECT id, school_id, tenant_id, email, role::text, status::text, full_name, expires_at, created_at FROM invitations WHERE tenant_id = $1 AND school_id = $2`

	args := []interface{}{tenantID, schoolID}
	argIdx := 3

	if !filter.Expired {
		countQuery += ` AND (status != 'expired' OR (status = 'pending' AND expires_at > NOW()))`
		dataQuery += ` AND (status != 'expired' OR (status = 'pending' AND expires_at > NOW()))`
	}

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		countQuery += fmt.Sprintf(` AND (full_name ILIKE $%d)`, argIdx)
		dataQuery += fmt.Sprintf(` AND (full_name ILIKE $%d)`, argIdx)
		args = append(args, pattern)
		argIdx++
	}

	if filter.Email != "" {
		pattern := "%" + filter.Email + "%"
		countQuery += fmt.Sprintf(` AND email ILIKE $%d`, argIdx)
		dataQuery += fmt.Sprintf(` AND email ILIKE $%d`, argIdx)
		args = append(args, pattern)
		argIdx++
	}

	if filter.Status != "" {
		countQuery += fmt.Sprintf(` AND status::text = $%d`, argIdx)
		dataQuery += fmt.Sprintf(` AND status::text = $%d`, argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Role != "" {
		countQuery += fmt.Sprintf(` AND role::text = $%d`, argIdx)
		dataQuery += fmt.Sprintf(` AND role::text = $%d`, argIdx)
		args = append(args, filter.Role)
		argIdx++
	}

	// Count total
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invitations: %w", err)
	}

	// Fetch data
	dataQuery += ` ORDER BY created_at DESC`
	dataQuery += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	var invitations []Invitation
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(&inv.ID, &inv.SchoolID, &inv.TenantID, &inv.Email, &inv.Role, &inv.Status,
			&inv.FullName, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if invitations == nil {
		invitations = []Invitation{}
	}

	return invitations, total, nil
}

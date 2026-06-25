package members

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// PgRepository handles member and invitation database operations.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new PgRepository.
func NewRepository(pools *database.Pools) *PgRepository {
	return &PgRepository{pool: pools.PG}
}

// ListByRole returns paginated members (users with active memberships) for a given role.
func (r *PgRepository) ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = $3 AND m.is_active = true
	`
	args := []interface{}{tenantID, schoolID, role}
	if search != "" {
		pattern := "%" + search + "%"
		countQuery += ` AND (u.full_name ILIKE $4 OR u.email ILIKE $5)`
		args = append(args, pattern, pattern)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count members: %w", err)
	}

	// Fetch data
	dataQuery := `
		SELECT u.id, u.email, u.full_name, m.role::text, m.is_active, m.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = $3 AND m.is_active = true
	`
	dataArgs := []interface{}{tenantID, schoolID, role}
	if search != "" {
		pattern := "%" + search + "%"
		dataQuery += ` AND (u.full_name ILIKE $4 OR u.email ILIKE $5)`
		dataArgs = append(dataArgs, pattern, pattern)
	}

	dataQuery += ` ORDER BY u.full_name LIMIT $` + fmt.Sprintf("%d", len(dataArgs)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(dataArgs)+2)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.FullName, &m.Role, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if members == nil {
		members = []Member{}
	}

	return members, total, nil
}

// GetActiveSchoolID returns the active school ID for a user in a tenant.
func (r *PgRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	const query = `
		SELECT school_id FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		ORDER BY
			CASE role
				WHEN 'SCHOOL_ADMIN'::user_role THEN 1
				WHEN 'TEACHER'::user_role THEN 2
				WHEN 'NURSE'::user_role THEN 3
				WHEN 'FINANCE'::user_role THEN 4
			END
		LIMIT 1
	`

	var schoolID string
	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(&schoolID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("no active membership found")
		}
		return "", fmt.Errorf("get active school: %w", err)
	}
	return schoolID, nil
}

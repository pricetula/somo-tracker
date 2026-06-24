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

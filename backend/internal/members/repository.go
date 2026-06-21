package members

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// Repository handles member and invitation database operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pools *database.Pools) *Repository {
	return &Repository{pool: pools.PG}
}

// ListByRole returns paginated members (users with active memberships) for a given role.
func (r *Repository) ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
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
		countQuery += ` AND (u.first_name ILIKE $4 OR u.last_name ILIKE $5 OR u.email ILIKE $6)`
		args = append(args, pattern, pattern, pattern)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count members: %w", err)
	}

	// Fetch data
	dataQuery := `
		SELECT u.id, u.email, u.first_name, u.last_name, m.role::text, m.is_active, m.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = $1 AND m.school_id = $2 AND m.role::text = $3 AND m.is_active = true
	`
	dataArgs := []interface{}{tenantID, schoolID, role}
	if search != "" {
		pattern := "%" + search + "%"
		dataQuery += ` AND (u.first_name ILIKE $4 OR u.last_name ILIKE $5 OR u.email ILIKE $6)`
		dataArgs = append(dataArgs, pattern, pattern, pattern)
	}

	dataQuery += ` ORDER BY u.first_name, u.last_name LIMIT $` + fmt.Sprintf("%d", len(dataArgs)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(dataArgs)+2)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.FirstName, &m.LastName, &m.Role, &m.IsActive, &m.CreatedAt); err != nil {
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
func (r *Repository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
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
func (r *Repository) ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
	// Build count query
	countQuery := `SELECT COUNT(*) FROM invitations WHERE tenant_id = $1 AND school_id = $2`
	dataQuery := `SELECT id, school_id, tenant_id, email, role::text, status::text, first_name, last_name, expires_at, created_at FROM invitations WHERE tenant_id = $1 AND school_id = $2`

	args := []interface{}{tenantID, schoolID}
	argIdx := 3

	if !filter.Expired {
		countQuery += ` AND (status != 'expired' OR (status = 'pending' AND expires_at > NOW()))`
		dataQuery += ` AND (status != 'expired' OR (status = 'pending' AND expires_at > NOW()))`
	}

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		countQuery += fmt.Sprintf(` AND (first_name ILIKE $%d OR last_name ILIKE $%d)`, argIdx, argIdx+1)
		dataQuery += fmt.Sprintf(` AND (first_name ILIKE $%d OR last_name ILIKE $%d)`, argIdx, argIdx+1)
		args = append(args, pattern, pattern)
		argIdx += 2
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
			&inv.FirstName, &inv.LastName, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
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

// GetPendingInviteByEmail checks if a pending invite exists for this email in the school.

// GetTenantStytchOrgID returns the Stytch org ID for a tenant.
func (r *Repository) GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	const query = `SELECT stytch_org_id FROM tenants WHERE id = $1`

	var orgID string
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&orgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("tenant not found")
		}
		return "", fmt.Errorf("get tenant stytch org: %w", err)
	}
	return orgID, nil
}

// GetMemberByEmail returns the first active member with the given email in a school.
// Returns nil if no active membership exists.
func (r *Repository) GetMemberByEmail(ctx context.Context, schoolID, email string) (*Member, error) {
	const query = `
		SELECT u.id, u.email, u.first_name, u.last_name, m.role::text, m.is_active, m.created_at
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.school_id = $1 AND u.email = $2 AND m.is_active = true
		LIMIT 1
	`

	var m Member
	err := r.pool.QueryRow(ctx, query, schoolID, email).Scan(&m.ID, &m.Email, &m.FirstName, &m.LastName, &m.Role, &m.IsActive, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get member by email: %w", err)
	}
	return &m, nil
}

// GetPendingInviteByEmail checks if a pending invite exists for this email in the school.
func (r *Repository) GetPendingInviteByEmail(ctx context.Context, schoolID, email string) (*Invitation, error) {
	const query = `
		SELECT id, school_id, tenant_id, email, role::text, status::text, first_name, last_name, expires_at, created_at
		FROM invitations
		WHERE school_id = $1 AND email = $2 AND status = 'pending' AND expires_at > NOW()
		LIMIT 1
	`

	var inv Invitation
	err := r.pool.QueryRow(ctx, query, schoolID, email).Scan(
		&inv.ID, &inv.SchoolID, &inv.TenantID, &inv.Email, &inv.Role, &inv.Status,
		&inv.FirstName, &inv.LastName, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get pending invite: %w", err)
	}
	return &inv, nil
}

// CreateInvitation inserts a new invitation record.
func (r *Repository) CreateInvitation(ctx context.Context, inv *Invitation, invitedBy string) error {
	const query = `
		INSERT INTO invitations (id, school_id, tenant_id, email, role, status, invited_by, token, expires_at, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5::user_role, 'pending', $6, $7, $8, $9, $10)
	`

	initedByArg := interface{}(nil)
	if invitedBy != "" {
		initedByArg = invitedBy
	}

	_, err := r.pool.Exec(ctx, query,
		inv.ID, inv.SchoolID, inv.TenantID, inv.Email, inv.Role,
		initedByArg, // invited_by
		inv.ID,      // token is the invitation ID for simplicity
		inv.ExpiresAt,
		inv.FirstName, inv.LastName,
	)
	if err != nil {
		return fmt.Errorf("insert invitation: %w", err)
	}
	return nil
}

// SetInvitationStytchMemberID stores the Stytch member ID on an invitation.
func (r *Repository) SetInvitationStytchMemberID(ctx context.Context, id, stytchMemberID string) error {
	const query = `UPDATE invitations SET stytch_member_id = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, stytchMemberID, id)
	if err != nil {
		return fmt.Errorf("set stytch member id: %w", err)
	}
	return nil
}

// invitationTTL is how long an invitation remains valid.
const invitationTTL = 7 * 24 * time.Hour

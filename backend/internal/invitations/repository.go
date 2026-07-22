package invitations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/middleware"
)

// uuidToPtr converts a uuid.UUID to a *uuid.UUID, returning nil for uuid.Nil.
// This is used for nullable UUID columns where Go zero (uuid.Nil) should map to SQL NULL.
func uuidToPtr(u uuid.UUID) *uuid.UUID {
	if u == uuid.Nil {
		return nil
	}
	return &u
}

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

// CountInvitations returns the total count of invitations for a given role.
func (r *PgRepository) CountInvitations(ctx context.Context, tenantID, schoolID string, role string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM invitations
		WHERE tenant_id = $1 AND school_id = $2 AND role::text = $3
		  AND (status != 'expired' OR (status = 'pending' AND expires_at > NOW()))
	`

	var total int
	if err := r.pool.QueryRow(ctx, query, tenantID, schoolID, role).Scan(&total); err != nil {
		return 0, fmt.Errorf("count invitations: %w", err)
	}

	return total, nil
}

// ============================================================================
// Bulk Invitation Repository Methods
// ============================================================================

// CheckExistingEmails returns two subsets of the input emails:
// 1. those that already exist in the users table for this tenant
// 2. those that already have a pending invitation for this school
func (r *PgRepository) CheckExistingEmails(ctx context.Context, tenantID, schoolID string, emails []string) ([]string, []string, error) {
	if len(emails) == 0 {
		return []string{}, []string{}, nil
	}

	// Check users table
	userQuery := `
		SELECT COALESCE(array_agg(DISTINCT email), ARRAY[]::text[])
		FROM users
		WHERE tenant_id = $1 AND email = ANY($2)
	`
	var existingUsers []string
	if err := r.pool.QueryRow(ctx, userQuery, tenantID, emails).Scan(&existingUsers); err != nil {
		return nil, nil, fmt.Errorf("check existing emails in users: %w", err)
	}

	// Check invitations table for pending invites
	inviteQuery := `
		SELECT COALESCE(array_agg(DISTINCT email), ARRAY[]::text[])
		FROM invitations
		WHERE school_id = $1 AND email = ANY($2) AND status = 'pending'
	`
	var existingInvites []string
	if err := r.pool.QueryRow(ctx, inviteQuery, schoolID, emails).Scan(&existingInvites); err != nil {
		return nil, nil, fmt.Errorf("check existing emails in invitations: %w", err)
	}

	return existingUsers, existingInvites, nil
}

// InsertInvitation inserts a single invitation record within a transaction.
func (r *PgRepository) InsertInvitation(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error {
	query := `
		INSERT INTO invitations (tenant_id, school_id, email, role, status, invited_by,
		                        token, expires_at, full_name, stytch_member_id, import_job_id)
		VALUES ($1, $2, $3, $4::user_role, $5, $6,
		        gen_random_uuid()::text, $7, $8, $9, $10)
	`
	_, err := tx.Exec(ctx, query,
		params.TenantID,
		params.SchoolID,
		params.Email,
		params.Role,
		params.Status,
		uuidToPtr(params.InvitedBy), // nil → SQL NULL
		params.ExpiresAt,
		params.FullName,
		params.StytchMemberID,
		uuidToPtr(params.ImportJobID), // nil → SQL NULL
	)
	if err != nil {
		return fmt.Errorf("insert invitation: %w", err)
	}
	return nil
}

// RevokeInvitation sets an invitation's status to 'revoked'.
func (r *PgRepository) RevokeInvitation(ctx context.Context, id, schoolID string) error {
	query := `UPDATE invitations SET status = 'revoked', updated_at = NOW() WHERE id = $1::uuid AND school_id = $2 AND status = 'pending'`
	tag, err := r.pool.Exec(ctx, query, id, schoolID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke invitation: %w", middleware.ErrNotFound)
	}
	return nil
}

// GetStytchOrgID retrieves the Stytch organization ID for a tenant.
func (r *PgRepository) GetStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	query := `SELECT stytch_org_id FROM tenants WHERE id = $1`
	var orgID string
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&orgID)
	if err != nil {
		return "", fmt.Errorf("get stytch org id: %w", err)
	}
	return orgID, nil
}

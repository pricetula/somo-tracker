package invitations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
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
// CountInvitations returns the total count of invitations for a given role.
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
	if err := database.FromContext(ctx, r.pool).QueryRow(ctx, userQuery, tenantID, emails).Scan(&existingUsers); err != nil {
		return nil, nil, fmt.Errorf("check existing emails in users: %w", err)
	}

	// Check invitations table for pending invites
	inviteQuery := `
		SELECT COALESCE(array_agg(DISTINCT email), ARRAY[]::text[])
		FROM invitations
		WHERE school_id = $1 AND email = ANY($2) AND status = 'pending'
	`
	var existingInvites []string
	if err := database.FromContext(ctx, r.pool).QueryRow(ctx, inviteQuery, schoolID, emails).Scan(&existingInvites); err != nil {
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
// GetStytchOrgID retrieves the Stytch organization ID for a tenant.
func (r *PgRepository) GetStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	query := `SELECT stytch_org_id FROM tenants WHERE id = $1`
	var orgID string
	err := database.FromContext(ctx, r.pool).QueryRow(ctx, query, tenantID).Scan(&orgID)
	if err != nil {
		return "", fmt.Errorf("get stytch org id: %w", err)
	}
	return orgID, nil
}

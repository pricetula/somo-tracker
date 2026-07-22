package invitations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Service contains business logic for the invitations domain.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListInvitations returns paginated invitations with optional filters.
func (s *Service) ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.ListInvitations(ctx, tenantID, schoolID, filter)
}

// CountInvitations returns the total number of non-expired invitations for a given role.
// RevokeInvitation revokes a pending invitation by ID.
func (s *Service) RevokeInvitation(ctx context.Context, id, schoolID string) error {
	return s.repo.RevokeInvitation(ctx, id, schoolID)
}

func (s *Service) CountInvitations(ctx context.Context, tenantID, schoolID string, role string) (int, error) {
	return s.repo.CountInvitations(ctx, tenantID, schoolID, role)
}

// ============================================================================
// Bulk Invitation Service Methods
// ============================================================================

// CheckExistingEmails checks which of the provided emails already exist
// in the users table for this tenant, or have a pending invitation for this school.
func (s *Service) CheckExistingEmails(ctx context.Context, tenantID, schoolID string, emails []string) ([]string, []string, error) {
	return s.repo.CheckExistingEmails(ctx, tenantID, schoolID, emails)
}

// InsertInvitation creates a new invitation record within a transaction.
func (s *Service) InsertInvitation(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error {
	return s.repo.InsertInvitation(ctx, tx, params)
}

// GetStytchOrgID returns the Stytch org ID for the given tenant.
func (s *Service) GetStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	return s.repo.GetStytchOrgID(ctx, tenantID)
}

// StytchInviteSender is the minimal subset of auth.IdentityProvider needed
// by the StaffInviteImporter for sending Stytch invitations.
type StytchInviteSender interface {
	CreateMember(ctx context.Context, orgID, email, name string) (memberID string, err error)
	SendDiscoveryEmail(ctx context.Context, email string) error
	// SendDiscoveryEmailWithRedirect sends a discovery magic link with a custom
	// redirect URL. Used by invite flows so the callback goes to the invite
	// acceptance endpoint instead of the default login callback.
	SendDiscoveryEmailWithRedirect(ctx context.Context, email, redirectURL string) error
	GetMemberByEmail(ctx context.Context, orgID, email string) (memberID string, err error)
	// InviteMemberByEmail sends a Stytch invite email to join an organization.
	// This is the proper invitation email (not a login/discovery email).
	// Creates the member in Stytch and sends the invite in one call.
	InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (memberID string, err error)
}

// validateBulkInviteRole checks that the role string is a valid user_role.
// Returns an error with a user-facing message if invalid.
func validateBulkInviteRole(role string) error {
	validRoles := map[string]bool{
		"SCHOOL_ADMIN": true,
		"TEACHER":      true,
		"NURSE":        true,
		"FINANCE":      true,
	}
	if !validRoles[role] {
		return fmt.Errorf("%w: invalid role %q — must be one of SCHOOL_ADMIN, TEACHER, NURSE, FINANCE", ErrInvalidInput, role)
	}
	return nil
}

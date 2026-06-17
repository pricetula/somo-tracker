package members

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
)

// Service contains business logic for the members domain.
type Service struct {
	repo   *Repository
	idp    auth.IdentityProvider
	cfg    config.Config
	logger *zap.Logger
}

// NewService creates a new Service.
func NewService(repo *Repository, idp auth.IdentityProvider, cfg config.Config, logger *zap.Logger) *Service {
	return &Service{repo: repo, idp: idp, cfg: cfg, logger: logger}
}

// ListMembers returns paginated members filtered by role.
func (s *Service) ListMembers(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByRole(ctx, tenantID, schoolID, role, offset, limit, search)
}

// BulkInvite sends invitations to multiple people to join the school with a given role.
func (s *Service) BulkInvite(ctx context.Context, tenantID, schoolID string, req BulkInviteRequest) (*BulkInviteResponse, error) {
	if len(req.Invites) == 0 {
		return nil, fmt.Errorf("at least one invite is required")
	}

	// Resolve Stytch org ID
	stytchOrgID, err := s.repo.GetTenantStytchOrgID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant stytch org: %w", err)
	}

	resp := &BulkInviteResponse{}

	for _, item := range req.Invites {
		if item.Email == "" {
			resp.Failed++
			resp.Errors = append(resp.Errors, InviteErrorItem{
				Email: item.Email,
				Error: "email is required",
			})
			continue
		}

		// Check if user is already a member
		existing, err := s.repo.GetMemberByEmail(ctx, schoolID, item.Email)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, InviteErrorItem{
				Email: item.Email,
				Error: "internal error checking existing membership",
			})
			continue
		}
		if existing != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, InviteErrorItem{
				Email: item.Email,
				Error: "user is already a member of this school",
			})
			continue
		}

		// Check for existing pending invite
		pending, err := s.repo.GetPendingInviteByEmail(ctx, schoolID, item.Email)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, InviteErrorItem{
				Email: item.Email,
				Error: "internal error checking existing invitation",
			})
			continue
		}
		if pending != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, InviteErrorItem{
				Email: item.Email,
				Error: "pending invitation already exists for this email",
			})
			continue
		}

		// Create invitation record
		now := time.Now().UTC()
		firstName := item.FirstName
		lastName := item.LastName

		inv := &Invitation{
			ID:        newID(),
			SchoolID:  schoolID,
			TenantID:  tenantID,
			Email:     item.Email,
			Role:      req.Role,
			Status:    "pending",
			FirstName: strPtr(firstName),
			LastName:  strPtr(lastName),
			ExpiresAt: now.Add(invitationTTL),
			CreatedAt: now,
		}

		if err := s.repo.CreateInvitation(ctx, inv); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, InviteErrorItem{
				Email: item.Email,
				Error: "failed to create invitation",
			})
			continue
		}

		// Send Stytch invite email
		fullName := item.FirstName
		if item.LastName != "" {
			if fullName != "" {
				fullName += " "
			}
			fullName += item.LastName
		}

		memberID, err := s.idp.InviteMemberByEmail(ctx, stytchOrgID, item.Email, fullName, s.cfg.FrontendURL+"/login")
		if err != nil {
			s.logger.Warn("stytch invite failed, but invitation persisted",
				zap.String("email", item.Email),
				zap.Error(err),
			)
			resp.Sent++
			// The invitation is already persisted — it can be resent later
			continue
		}

		// Store the Stytch member ID
		if err := s.repo.SetInvitationStytchMemberID(ctx, inv.ID, memberID); err != nil {
			s.logger.Warn("failed to persist stytch member id",
				zap.String("invitation_id", inv.ID),
				zap.Error(err),
			)
		}

		resp.Sent++
	}

	return resp, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────

var idCounter int64

func newID() string {
	idCounter++
	return fmt.Sprintf("inv_%d_%d", time.Now().UnixNano(), idCounter)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

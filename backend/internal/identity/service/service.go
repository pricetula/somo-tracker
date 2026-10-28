package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SessionContext holds validated session data for request context.
type SessionContext struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	Role     string
}

// Service defines business logic for identity domain (BE-002).
type Service struct {
	repo Repository
}

// Repository is domain repository interface, defined by consumer (BE-005).
type Repository interface {
	// TODO: define methods matching store queries
	// GetPendingOnboardingByEmail, UpsertPendingOnboarding, UpdatePendingOnboardingStep
	// CreateAppSession, GetAppSessionByTokenHash, RevokeAppSession
	// CreateOnboardingTicket, GetOnboardingTicketByHash, MarkOnboardingTicketUsed
}

// New creates service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// RequestMagicLink initiates Stytch Discovery magic link flow.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	// 1. Rate limit by email/IP
	// 2. Generate PKCE code_verifier, store hashed in PKCECode with expiry
	// 3. Compute code_challenge S256
	// 4. Call Stytch B2B Discovery magic_links/email with email, login_magic_link_url, pkce_code_challenge
	// 5. Return generic success
	return nil
}

// AuthenticateToken verifies one-time Stytch token, upserts pending onboarding and creates ticket.
func (s *Service) AuthenticateToken(ctx context.Context, token, pkceCodeVerifier string) (string, error) {
	// 1. Call Stytch b2b/discovery/authenticate with token + pkce_code_verifier
	// 2. Get verified email, discovered_organizations, intermediate_session_token
	// 3. Normalize email, upsert PendingOnboarding status=pending
	// 4. Create OnboardingTicket with ticket_hash + encrypted IST, expires ≤ IST TTL
	// 5. Return state: onboarding_required | organization_selection_required | authenticated
	return "", nil
}

// ProvisionTenant creates Stytch org/member and local tenant/user/membership.
func (s *Service) ProvisionTenant(ctx context.Context, pendingID uuid.UUID, schoolName string, idempotencyKey string) error {
	// 1. Idempotency check
	// 2. Exchange IST for Stytch session via discovery/exchange
	// 3. Create Stytch organization and member
	// 4. Create local Tenant, User, Membership in transaction
	// 5. Update PendingOnboarding with stytch_organization_id, stytch_member_id, tenant_id
	return nil
}

// CreateSession creates opaque app session after tenant provisioning.
func (s *Service) CreateSession(ctx context.Context, userID, tenantID uuid.UUID, stytchMemberSessionID string, expiresAt time.Time) (string, error) {
	// 1. Generate 256-bit random token
	// 2. SHA-256 hash, store in AppSession with expires_at = Stytch expires_at
	// 3. Return raw token for cookie issuance
	return "", nil
}

// ValidateSession validates opaque token hash.
func (s *Service) ValidateSession(ctx context.Context, tokenHash []byte) (*SessionContext, error) {
	// 1. Lookup AppSession by hash, check revoked_at IS NULL and expires_at > now
	// 2. Return SessionContext with user_id, tenant_id
	return nil, nil
}

// Logout revokes session and Stytch member session.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	// 1. Revoke AppSession row
	// 2. Call Stytch b2b/members/{id}/sessions/revoke if stytch_member_session_id present
	return nil
}

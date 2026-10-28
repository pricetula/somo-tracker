package service

import (
	"context"

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
	// TODO: rate limit, call Stytch client, return generic success
	return nil
}

// AuthenticateToken verifies one-time Stytch token, upserts pending onboarding and creates ticket.
func (s *Service) AuthenticateToken(ctx context.Context, token string) (string, error) {
	// TODO: verify with Stytch, upsert PendingOnboarding, create OnboardingTicket
	return "", nil
}

// CreateSession creates opaque app session after tenant provisioning.
func (s *Service) CreateSession(ctx context.Context, userID, tenantID uuid.UUID) (string, error) {
	// TODO: generate random token >=256 bits, hash, store AppSession, return raw token
	return "", nil
}

// ValidateSession validates opaque token hash.
func (s *Service) ValidateSession(ctx context.Context, tokenHash []byte) (*SessionContext, error) {
	// TODO: lookup AppSession, check expiry/revocation, return user/tenant
	return nil, nil
}

// Logout revokes session and Stytch member session.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	// TODO: revoke AppSession, call Stytch revoke
	return nil
}

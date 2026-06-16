package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/slug"

	"github.com/stytchauth/stytch-go/v16/stytch/b2b/b2bstytchapi"
	intermediatesessions "github.com/stytchauth/stytch-go/v16/stytch/b2b/discovery/intermediatesessions"
	magiclinksdiscovery "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/discovery"
	emaildiscovery "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/email/discovery"
	"github.com/stytchauth/stytch-go/v16/stytch/b2b/organizations"
	"github.com/stytchauth/stytch-go/v16/stytch/stytcherror"
)

// ============================================================================
// StytchAdapter — concrete implementation of IdentityProvider.
// This is the ONLY file allowed to import the Stytch SDK (requirement 1).
// ============================================================================

// StytchAdapter wraps the Stytch B2B API client and implements IdentityProvider.
type StytchAdapter struct {
	api    *b2bstytchapi.API
	logger *zap.Logger
	cfg    config.Config
}

// NewStytchAdapter creates a new StytchAdapter and initializes the Stytch B2B client.
func NewStytchAdapter(cfg config.Config, logger *zap.Logger) (*StytchAdapter, error) {
	opts := []b2bstytchapi.Option{
		b2bstytchapi.WithSkipJWKSInitialization(),
	}
	if cfg.StytchBaseURL != "" {
		opts = append(opts, b2bstytchapi.WithBaseURI(cfg.StytchBaseURL))
		logger.Info("stytch adapter: using custom base URL",
			zap.String("base_url", cfg.StytchBaseURL),
		)
	}
	api, err := b2bstytchapi.NewClient(cfg.StytchProjectID, cfg.StytchSecret, opts...)
	if err != nil {
		return nil, fmt.Errorf("stytch client init: %w", err)
	}

	logger.Info("Stytch B2B client initialized",
		zap.String("project_id", cfg.StytchProjectID),
	)

	return &StytchAdapter{
		api:    api,
		logger: logger,
		cfg:    cfg,
	}, nil
}

// SendDiscoveryEmail dispatches a discovery magic link via Stytch.
func (s *StytchAdapter) SendDiscoveryEmail(ctx context.Context, email string) error {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch SendDiscoveryEmail completed",
			zap.String("email", email),
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &emaildiscovery.SendParams{
		EmailAddress:        email,
		DiscoveryRedirectURL: s.cfg.StytchRedirectURL,
	}

	_, err := s.api.MagicLinks.Email.Discovery.Send(ctx, params)
	if err != nil {
		s.logger.Error("Stytch SendDiscoveryEmail failed",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("%w: stytch send discovery email: %v", ErrInternal, err)
	}

	return nil
}

// AuthenticateDiscoveryToken validates a magic-link token and returns the IST.
func (s *StytchAdapter) AuthenticateDiscoveryToken(ctx context.Context, token string) (string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch AuthenticateDiscoveryToken completed",
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &magiclinksdiscovery.AuthenticateParams{
		DiscoveryMagicLinksToken: token,
	}

	resp, err := s.api.MagicLinks.Discovery.Authenticate(ctx, params)
	if err != nil {
		s.logger.Error("Stytch AuthenticateDiscoveryToken failed",
			zap.Error(err),
		)

		if isExpiredTokenError(err) {
			return "", fmt.Errorf("%w: stytch token expired", ErrExpiredToken)
		}
		return "", fmt.Errorf("%w: stytch authenticate: %v", ErrInternal, err)
	}

	if resp.IntermediateSessionToken == "" {
		return "", fmt.Errorf("%w: stytch response missing intermediate_session_token", ErrInternal)
	}

	return resp.IntermediateSessionToken, nil
}

// CreateOrganization provisions a new organization in Stytch.
func (s *StytchAdapter) CreateOrganization(ctx context.Context, name string) (string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch CreateOrganization completed",
			zap.String("org_name", name),
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &organizations.CreateParams{
		OrganizationName: name,
		OrganizationSlug: slug.Generate(name),
	}

	resp, err := s.api.Organizations.Create(ctx, params)
	if err != nil {
		s.logger.Error("Stytch CreateOrganization failed",
			zap.String("org_name", name),
			zap.Error(err),
		)
		return "", fmt.Errorf("%w: stytch create org: %v", ErrInternal, err)
	}

	orgID := resp.Organization.OrganizationID
	if orgID == "" {
		return "", fmt.Errorf("%w: stytch response missing organization_id", ErrInternal)
	}

	s.logger.Info("Stytch organization created",
		zap.String("org_name", name),
		zap.String("stytch_org_id", orgID),
	)

	return orgID, nil
}

// ExchangeIntermediateSession exchanges an IST for a full session within an org.
func (s *StytchAdapter) ExchangeIntermediateSession(ctx context.Context, ist, orgID string) (ExchangeResult, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch ExchangeIntermediateSession completed",
			zap.String("org_id", orgID),
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &intermediatesessions.ExchangeParams{
		IntermediateSessionToken: ist,
		OrganizationID:           orgID,
	}

	resp, err := s.api.Discovery.IntermediateSessions.Exchange(ctx, params)
	if err != nil {
		s.logger.Error("Stytch ExchangeIntermediateSession failed",
			zap.String("org_id", orgID),
			zap.Error(err),
		)

		// Map specific Stytch error types to domain errors for proper handling
		if isJITProvisioningError(err) {
			return ExchangeResult{}, fmt.Errorf("%w: stytch exchange ist: %v", ErrJITProvisioningNotAllowed, err)
		}
		if isMemberNotFoundError(err) {
			return ExchangeResult{}, fmt.Errorf("%w: stytch exchange ist: %v", ErrMemberNotFound, err)
		}
		if isOrgNotFoundError(err) {
			return ExchangeResult{}, fmt.Errorf("%w: stytch exchange ist: %v", ErrOrgNotFound, err)
		}

		return ExchangeResult{}, fmt.Errorf("%w: stytch exchange ist: %v", ErrInternal, err)
	}

	result := ExchangeResult{
		MemberAuthenticated: resp.MemberAuthenticated,
		StytchSessionToken:  resp.SessionToken,
		MemberID:            resp.Member.MemberID,
		OrganizationID:      orgID,
	}

	s.logger.Info("IST exchange completed",
		zap.String("member_id", result.MemberID),
		zap.String("org_id", result.OrganizationID),
		zap.Bool("mfa_authenticated", result.MemberAuthenticated),
	)

	return result, nil
}

// isExpiredTokenError checks if the error is a Stytch expired magic link token error.
func isExpiredTokenError(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.ErrorType == "magic_link_token_expired"
	}
	return false
}

// Compile-time interface check.
// isJITProvisioningError checks if the error indicates that email JIT provisioning
// is not allowed for the target organization. This occurs when a member tries to
// authenticate against an org that doesn't allow just-in-time member provisioning.
func isJITProvisioningError(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.ErrorType == "email_jit_provisioning_not_allowed"
	}
	return false
}

// isMemberNotFoundError checks if the error indicates the member doesn't exist
// in the target organization. This can happen during IST exchange when the
// member was not JIT provisioned and has no existing membership.
func isMemberNotFoundError(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.ErrorType == "member_not_found"
	}
	return false
}

// isOrgNotFoundError checks if the error indicates the organization was not
// found in Stytch. This can happen during IST exchange if the org ID is
// invalid or the org was deleted.
func isOrgNotFoundError(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.ErrorType == "organization_not_found"
	}
	return false
}

var _ IdentityProvider = (*StytchAdapter)(nil)

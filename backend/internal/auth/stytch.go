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
	b2bmagiclinks "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks"
	magiclinksdiscovery "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/discovery"
	stytchemail "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/email"
	emaildiscovery "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/email/discovery"
	"github.com/stytchauth/stytch-go/v16/stytch/b2b/organizations"
	members "github.com/stytchauth/stytch-go/v16/stytch/b2b/organizations/members"
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
	return s.sendDiscoveryEmail(ctx, email, s.cfg.StytchRedirectURL)
}

// SendDiscoveryEmailWithRedirect dispatches a discovery magic link to the given
// email with a custom redirect URL. Used by invite flows where the callback
// endpoint differs from the default login callback.
func (s *StytchAdapter) SendDiscoveryEmailWithRedirect(ctx context.Context, email, redirectURL string) error {
	if redirectURL == "" {
		redirectURL = s.cfg.StytchRedirectURL
	}
	return s.sendDiscoveryEmail(ctx, email, redirectURL)
}

// sendDiscoveryEmail is the shared implementation that sends a Stytch discovery
// magic link with the given redirect URL.
func (s *StytchAdapter) sendDiscoveryEmail(ctx context.Context, email, redirectURL string) error {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch sendDiscoveryEmail completed",
			zap.String("email", email),
			zap.String("redirect_url", redirectURL),
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &emaildiscovery.SendParams{
		EmailAddress:         email,
		DiscoveryRedirectURL: redirectURL,
	}

	_, err := s.api.MagicLinks.Email.Discovery.Send(ctx, params)
	if err != nil {
		s.logger.Error("Stytch sendDiscoveryEmail failed",
			zap.String("email", email),
			zap.String("redirect_url", redirectURL),
			zap.Error(err),
		)
		return fmt.Errorf("%w: stytch send discovery email: %v", ErrInternal, err)
	}

	return nil
}

// AuthenticateDiscoveryToken validates a magic-link token and returns the IST,
// email, and any discovered organizations the user already belongs to.
func (s *StytchAdapter) AuthenticateDiscoveryToken(ctx context.Context, token string) (string, string, []DiscoveredOrg, error) {
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
			return "", "", nil, fmt.Errorf("%w: stytch token expired", ErrExpiredToken)
		}
		return "", "", nil, fmt.Errorf("%w: stytch authenticate: %v", ErrInternal, err)
	}

	if resp.IntermediateSessionToken == "" {
		return "", "", nil, fmt.Errorf("%w: stytch response missing intermediate_session_token", ErrInternal)
	}
	if resp.EmailAddress == "" {
		return "", "", nil, fmt.Errorf("%w: stytch response missing email_address", ErrInternal)
	}

	// Map Stytch discovered organizations to our domain type
	discoveredOrgs := make([]DiscoveredOrg, 0, len(resp.DiscoveredOrganizations))
	for _, do := range resp.DiscoveredOrganizations {
		org := DiscoveredOrg{
			MemberAuthenticated: do.MemberAuthenticated,
		}
		if do.Organization != nil {
			org.OrganizationID = do.Organization.OrganizationID
			org.OrganizationName = do.Organization.OrganizationName
		}
		if do.Membership != nil && do.Membership.Member != nil {
			org.MemberID = do.Membership.Member.MemberID
			org.MemberName = do.Membership.Member.Name
		}
		discoveredOrgs = append(discoveredOrgs, org)
	}

	s.logger.Info("Stytch AuthenticateDiscoveryToken completed",
		zap.String("email", resp.EmailAddress),
		zap.Int("discovered_orgs", len(discoveredOrgs)),
		zap.Duration("latency", time.Since(start)),
	)

	return resp.IntermediateSessionToken, resp.EmailAddress, discoveredOrgs, nil
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

// AuthenticateInviteToken validates a magic-link token sent via an invite email.
// This is functionally identical to AuthenticateDiscoveryToken but exposed as
// a separate method for invite-flow clarity.
func (s *StytchAdapter) AuthenticateInviteToken(ctx context.Context, token string) (string, string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch AuthenticateInviteToken completed",
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &magiclinksdiscovery.AuthenticateParams{
		DiscoveryMagicLinksToken: token,
	}

	resp, err := s.api.MagicLinks.Discovery.Authenticate(ctx, params)
	if err != nil {
		s.logger.Error("Stytch AuthenticateInviteToken failed",
			zap.Error(err),
		)

		if isExpiredTokenError(err) {
			return "", "", fmt.Errorf("%w: stytch invite token expired", ErrExpiredToken)
		}
		return "", "", fmt.Errorf("%w: stytch authenticate invite: %v", ErrInternal, err)
	}

	if resp.IntermediateSessionToken == "" {
		return "", "", fmt.Errorf("%w: stytch response missing intermediate_session_token", ErrInternal)
	}
	if resp.EmailAddress == "" {
		return "", "", fmt.Errorf("%w: stytch response missing email_address", ErrInternal)
	}

	return resp.IntermediateSessionToken, resp.EmailAddress, nil
}

// AuthenticateMagicLink authenticates an org-scoped magic link token using the
// correct Stytch B2B endpoint: POST /v1/b2b/magic_links/authenticate.
//
// This is the RIGHT endpoint for invite tokens (from InviteMemberByEmail) and
// login/signup tokens. It is NOT the discovery authenticate endpoint
// (POST /v1/b2b/magic_links/discovery/authenticate), which only handles
// cross-org discovery flows and returns 404 for org-scoped invite tokens.
//
// If the member is fully authenticated (no MFA required), StytchSessionToken
// is set in the result. If MFA is required, IntermediateSessionToken is set
// and MemberAuthenticated is false — the caller should complete MFA and then
// exchange the IST via ExchangeIntermediateSession.
func (s *StytchAdapter) AuthenticateMagicLink(ctx context.Context, token string) (*MagicLinkAuthResult, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch AuthenticateMagicLink completed",
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &b2bmagiclinks.AuthenticateParams{
		MagicLinksToken: token,
	}

	resp, err := s.api.MagicLinks.Authenticate(ctx, params)
	if err != nil {
		s.logger.Error("Stytch AuthenticateMagicLink failed",
			zap.Error(err),
		)

		// Expired invite magic links must surface as 401 expired_token, not a
		// generic 500 — mirrors the discovery endpoint's handling (B4).
		if isExpiredTokenError(err) {
			return nil, fmt.Errorf("%w: stytch invite token expired", ErrExpiredToken)
		}
		return nil, fmt.Errorf("%w: stytch authenticate magic link: %v", ErrInternal, err)
	}

	if resp.MemberID == "" {
		return nil, fmt.Errorf("%w: stytch response missing member_id", ErrInternal)
	}

	// Extract email from the member object in the response.
	email := ""
	if resp.Member.EmailAddress != "" {
		email = resp.Member.EmailAddress
	}

	// Determine whether we got a full session or need MFA
	stytchSessionToken := ""
	intermediateSessionToken := ""
	if resp.MemberAuthenticated {
		stytchSessionToken = resp.SessionToken
		// An authenticated member MUST carry a session token; storing an empty
		// one would poison the sessions table and every downstream Stytch call
		// (B5).
		if stytchSessionToken == "" {
			return nil, fmt.Errorf("%w: stytch response missing session_token despite member_authenticated", ErrInternal)
		}
	} else {
		intermediateSessionToken = resp.IntermediateSessionToken
	}

	result := &MagicLinkAuthResult{
		MemberID:                 resp.MemberID,
		OrganizationID:           resp.OrganizationID,
		Email:                    email,
		StytchSessionToken:       stytchSessionToken,
		IntermediateSessionToken: intermediateSessionToken,
		MemberAuthenticated:      resp.MemberAuthenticated,
	}

	s.logger.Info("Stytch AuthenticateMagicLink completed",
		zap.String("member_id", result.MemberID),
		zap.String("org_id", result.OrganizationID),
		zap.String("email", result.Email),
		zap.Bool("member_authenticated", result.MemberAuthenticated),
	)

	return result, nil
}

// ExchangeInviteSession exchanges an IST for a full Stytch session within a
// specific organization. Enforces MemberAuthenticated == true and returns the
// Stytch session token directly.
func (s *StytchAdapter) ExchangeInviteSession(ctx context.Context, ist, orgID string) (string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch ExchangeInviteSession completed",
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
		s.logger.Error("Stytch ExchangeInviteSession failed",
			zap.String("org_id", orgID),
			zap.Error(err),
		)

		if isJITProvisioningError(err) {
			return "", fmt.Errorf("%w: stytch exchange invite: %v", ErrJITProvisioningNotAllowed, err)
		}
		if isMemberNotFoundError(err) {
			return "", fmt.Errorf("%w: stytch exchange invite: %v", ErrMemberNotFound, err)
		}
		if isOrgNotFoundError(err) {
			return "", fmt.Errorf("%w: stytch exchange invite: %v", ErrOrgNotFound, err)
		}

		return "", fmt.Errorf("%w: stytch exchange invite: %v", ErrInternal, err)
	}

	if !resp.MemberAuthenticated {
		s.logger.Warn("Stytch ExchangeInviteSession: MFA not satisfied",
			zap.String("org_id", orgID),
		)
		return "", ErrMFARequired
	}

	if resp.SessionToken == "" {
		return "", fmt.Errorf("%w: stytch response missing session_token", ErrInternal)
	}

	s.logger.Info("IST exchange completed for invite flow",
		zap.String("org_id", orgID),
	)

	return resp.SessionToken, nil
}

// sanitizeStytchError extracts a user-facing message from a Stytch error.
// The raw Stytch error contains request IDs, status codes, and debug URLs
// that are meaningless to end users. This helper returns just the human-readable
// message (the ErrorMessage field) when the error is a Stytch error, or a
// generic fallback otherwise.
func sanitizeStytchError(err error) string {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		// Stytch ErrorMessage already contains a human-readable description.
		// If it's empty, fall back to the error type code.
		msg := string(stytchErr.ErrorMessage)
		if msg == "" {
			msg = string(stytchErr.ErrorType)
		}
		return msg
	}
	// Non-Stytch errors: return a generic message to avoid leaking internals.
	return "authentication provider error"
}

// isExpiredTokenError checks if the error is a Stytch expired magic link token error.
func isExpiredTokenError(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.ErrorType == "magic_link_token_expired"
	}
	return false
}

// CreateMember provisions a new member in an existing Stytch organization.
func (s *StytchAdapter) CreateMember(ctx context.Context, orgID, email, name string) (string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch CreateMember completed",
			zap.String("org_id", orgID),
			zap.String("email", email),
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &members.CreateParams{
		OrganizationID: orgID,
		EmailAddress:   email,
		Name:           name,
	}

	resp, err := s.api.Organizations.Members.Create(ctx, params)
	if err != nil {
		s.logger.Error("Stytch CreateMember failed",
			zap.String("org_id", orgID),
			zap.String("email", email),
			zap.Error(err),
		)

		// A4: when the email is already a member of this org (e.g. the user was
		// previously registered and our DB row was lost, or they were provisioned
		// from the Stytch console), recover the existing member ID instead of
		// failing the whole registration with a 500.
		if isDuplicateMemberError(err) {
			s.logger.Info("Stytch CreateMember: member already exists — recovering existing member ID",
				zap.String("org_id", orgID),
				zap.String("email", email),
			)
			existingID, getErr := s.GetMemberByEmail(ctx, orgID, email)
			if getErr != nil {
				return "", fmt.Errorf("%w: stytch duplicate member + recovery lookup failed: %v", ErrInternal, getErr)
			}
			return existingID, nil
		}

		return "", fmt.Errorf("%w: stytch create member: %s", ErrInternal, sanitizeStytchError(err))
	}

	memberID := resp.Member.MemberID
	if memberID == "" {
		return "", fmt.Errorf("%w: stytch response missing member_id", ErrInternal)
	}

	s.logger.Info("Stytch member created",
		zap.String("member_id", memberID),
		zap.String("org_id", orgID),
		zap.String("email", email),
	)

	return memberID, nil
}

// isDuplicateMemberError checks if the error indicates the member already
// exists in the organization (Stytch B2B error type "duplicate_member_email").
func isDuplicateMemberError(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.ErrorType == "duplicate_member_email"
	}
	return false
}

// InviteMemberByEmail sends a Stytch invite email to join an organization.
// InviteMemberByEmail sends a Stytch invite email to join an organization.
func (s *StytchAdapter) InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
	start := time.Now()

	params := &stytchemail.InviteParams{
		OrganizationID:    orgID,
		EmailAddress:      email,
		Name:              name,
		InviteRedirectURL: redirectURL,
	}

	resp, err := s.api.MagicLinks.Email.Invite(ctx, params)
	if err != nil {
		s.logger.Error("Stytch InviteMemberByEmail failed",
			zap.String("org_id", orgID),
			zap.String("email", email),
			zap.Error(err),
		)
		return "", fmt.Errorf("%w: stytch invite member: %s", ErrInternal, sanitizeStytchError(err))
	}

	memberID := resp.MemberID
	if memberID == "" {
		return "", fmt.Errorf("%w: stytch response missing member_id", ErrInternal)
	}

	s.logger.Info("Stytch invite sent",
		zap.String("member_id", memberID),
		zap.String("org_id", orgID),
		zap.String("email", email),
		zap.Duration("latency", time.Since(start)),
	)

	return memberID, nil
}

// GetMemberByEmail retrieves a Stytch member's ID by looking up their email
// in the given organization. This is used when an invite attempt returns
// duplicate_member_email (i.e., the member is already active) and we still
// need the member ID for our local invitation record.
func (s *StytchAdapter) GetMemberByEmail(ctx context.Context, orgID, email string) (string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info("Stytch GetMemberByEmail completed",
			zap.String("org_id", orgID),
			zap.String("email", email),
			zap.Duration("latency", time.Since(start)),
		)
	}()

	params := &members.GetParams{
		OrganizationID: orgID,
		EmailAddress:   email,
	}

	resp, err := s.api.Organizations.Members.Get(ctx, params)
	if err != nil {
		s.logger.Error("Stytch GetMemberByEmail failed",
			zap.String("org_id", orgID),
			zap.String("email", email),
			zap.Error(err),
		)
		return "", fmt.Errorf("%w: stytch get member by email: %s", ErrInternal, sanitizeStytchError(err))
	}

	if resp.MemberID == "" {
		return "", fmt.Errorf("%w: stytch response missing member_id", ErrInternal)
	}

	s.logger.Info("Stytch member found by email",
		zap.String("member_id", resp.MemberID),
		zap.String("email", email),
	)

	return resp.MemberID, nil
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

package stytch

import "context"

// Client is the Stytch B2B adapter interface.
type Client interface {
	// Discovery magic link
	SendDiscoveryMagicLink(ctx context.Context, email string, loginURL string, pkceCodeChallenge string) error

	// Authenticate discovery token
	AuthenticateDiscovery(ctx context.Context, token string, pkceCodeVerifier string) (*DiscoveryAuthResponse, error)

	// Exchange IST for session
	ExchangeDiscoverySession(ctx context.Context, intermediateSessionToken string) (*DiscoveryExchangeResponse, error)

	// Provision organization and member
	CreateOrganizationAndMember(ctx context.Context, name string, email string) (*OrgMemberResponse, error)

	// Revoke member session
	RevokeMemberSession(ctx context.Context, memberID string, sessionID string) error
}

// DiscoveryAuthResponse mirrors Stytch b2b/discovery/authenticate response.
type DiscoveryAuthResponse struct {
	Email           string
	IntermediateSessionToken string
	ExpiresAt       string
	DiscoveredOrganizations []DiscoveredOrg
}

type DiscoveredOrg struct {
	OrganizationID string
	MemberID       string
}

// DiscoveryExchangeResponse mirrors exchange response.
type DiscoveryExchangeResponse struct {
	SessionToken string
	ExpiresAt    string
}

// OrgMemberResponse mirrors org/member creation.
type OrgMemberResponse struct {
	OrganizationID string
	MemberID       string
}

package stytch

import (
	"context"
)

// SDKClient wraps official Stytch Go SDK.
// Implementation is pending SDK version pin.
type SDKClient struct {
	// client *stytch.Client
	// config *config.Config
}

func NewSDKClient() *SDKClient {
	return &SDKClient{}
}

func (c *SDKClient) SendDiscoveryMagicLink(ctx context.Context, email string, loginURL string, pkceCodeChallenge string) error {
	// TODO: call stytch.Client.B2B.Discovery.MagicLinks.Email.Send
	// with email, login_magic_link_url, pkce_code_challenge
	return nil
}

func (c *SDKClient) AuthenticateDiscovery(ctx context.Context, token string, pkceCodeVerifier string) (*DiscoveryAuthResponse, error) {
	// TODO: call stytch.Client.B2B.Discovery.Authenticate
	// return email, intermediate_session_token, discovered_organizations
	return &DiscoveryAuthResponse{}, nil
}

func (c *SDKClient) ExchangeDiscoverySession(ctx context.Context, intermediateSessionToken string) (*DiscoveryExchangeResponse, error) {
	// TODO: call stytch.Client.B2B.Discovery.Exchange
	return &DiscoveryExchangeResponse{}, nil
}

func (c *SDKClient) CreateOrganizationAndMember(ctx context.Context, name string, email string) (*OrgMemberResponse, error) {
	// TODO: create org via stytch.Client.B2B.Organizations.Create
	// then create member via stytch.Client.B2B.Members.Create
	return &OrgMemberResponse{}, nil
}

func (c *SDKClient) RevokeMemberSession(ctx context.Context, memberID string, sessionID string) error {
	// TODO: call stytch.Client.B2B.Members.Sessions.Revoke
	return nil
}

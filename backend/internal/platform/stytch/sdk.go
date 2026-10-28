package stytch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SDKClient wraps Stytch B2B API via direct HTTP calls.
// Replace with official SDK once pinned.
type SDKClient struct {
	ProjectID  string
	Secret     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewSDKClient(projectID, secret, baseURL string) *SDKClient {
	return &SDKClient{
		ProjectID:  projectID,
		Secret:     secret,
		BaseURL:    baseURL,
		HTTPClient: http.DefaultClient,
	}
}

func (c *SDKClient) SendDiscoveryMagicLink(ctx context.Context, email string, loginURL string, pkceCodeChallenge string) error {
	payload := map[string]any{
		"email_address": email,
		"login_magic_link_url": loginURL,
		"pkce_code_challenge": pkceCodeChallenge,
		"pkce_code_challenge_method": "S256",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/b2b/discovery/magic_links/email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stytch-Project-Id", c.ProjectID)
	req.Header.Set("Stytch-Secret", c.Secret)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stytch error %d", resp.StatusCode)
	}
	return nil
}

func (c *SDKClient) AuthenticateDiscovery(ctx context.Context, token string, pkceCodeVerifier string) (*DiscoveryAuthResponse, error) {
	payload := map[string]any{
		"token": token,
		"pkce_code_verifier": pkceCodeVerifier,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/b2b/discovery/authenticate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stytch-Project-Id", c.ProjectID)
	req.Header.Set("Stytch-Secret", c.Secret)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		Email string `json:"email_address"`
		IntermediateSessionToken string `json:"intermediate_session_token"`
		ExpiresAt string `json:"expires_at"`
		DiscoveredOrganizations []struct{ OrganizationID string `json:"organization_id"`; MemberID string `json:"member_id"` } `json:"discovered_organizations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	discovered := make([]DiscoveredOrg, len(out.DiscoveredOrganizations))
	for i, d := range out.DiscoveredOrganizations {
		discovered[i] = DiscoveredOrg{OrganizationID: d.OrganizationID, MemberID: d.MemberID}
	}
	return &DiscoveryAuthResponse{
		Email: out.Email,
		IntermediateSessionToken: out.IntermediateSessionToken,
		ExpiresAt: out.ExpiresAt,
		DiscoveredOrganizations: discovered,
	}, nil
}

func (c *SDKClient) ExchangeDiscoverySession(ctx context.Context, intermediateSessionToken string) (*DiscoveryExchangeResponse, error) {
	// Stub for now
	return &DiscoveryExchangeResponse{}, nil
}

func (c *SDKClient) CreateOrganizationAndMember(ctx context.Context, name string, email string) (*OrgMemberResponse, error) {
	// Stub for now
	return &OrgMemberResponse{}, nil
}

func (c *SDKClient) RevokeMemberSession(ctx context.Context, memberID string, sessionID string) error {
	// Stub for now
	return nil
}

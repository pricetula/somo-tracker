package stytch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	stytchb2bapi "github.com/stytchauth/stytch-go/v16/stytch/b2b/b2bstytchapi"
	magicEmailDisc "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/email/discovery"
	magicDisc "github.com/stytchauth/stytch-go/v16/stytch/b2b/magiclinks/discovery"
	"github.com/stytchauth/stytch-go/v16/stytch/b2b/discovery/intermediatesessions"
	"github.com/stytchauth/stytch-go/v16/stytch/b2b/organizations"
	"github.com/stytchauth/stytch-go/v16/stytch/b2b/organizations/members"
	"github.com/stytchauth/stytch-go/v16/stytch/b2b/sessions"
)

// SDKClient wraps the official Stytch Go SDK and implements the stytch.Client interface
type SDKClient struct {
	api *stytchb2bapi.API
}

// NewSDKClient creates a new Stytch SDK client using official SDK
func NewSDKClient(projectID, secret string) *SDKClient {
	api, err := stytchb2bapi.NewClient(projectID, secret)
	if err != nil {
		return &SDKClient{api: nil}
	}
	return &SDKClient{api: api}
}

// NewSDKClientWithEnv creates a new Stytch SDK client
func NewSDKClientWithEnv(projectID, secret, env string) *SDKClient {
	api, err := stytchb2bapi.NewClient(projectID, secret)
	if err != nil {
		return &SDKClient{api: nil}
	}
	return &SDKClient{api: api}
}

func (c *SDKClient) SendDiscoveryMagicLink(ctx context.Context, email string, loginURL string, pkceCodeChallenge string) error {
	if c.api == nil {
		return context.DeadlineExceeded
	}
	_, err := c.api.MagicLinks.Email.Discovery.Send(ctx, &magicEmailDisc.SendParams{
		EmailAddress:         email,
		DiscoveryRedirectURL: loginURL,
		PkceCodeChallenge:    pkceCodeChallenge,
	})
	return err
}

func (c *SDKClient) AuthenticateDiscovery(ctx context.Context, token string, pkceCodeVerifier string) (*DiscoveryAuthResponse, error) {
	if c.api == nil {
		return nil, context.DeadlineExceeded
	}
	resp, err := c.api.MagicLinks.Discovery.Authenticate(ctx, &magicDisc.AuthenticateParams{
		DiscoveryMagicLinksToken: token,
		PkceCodeVerifier:         pkceCodeVerifier,
	})
	if err != nil {
		return nil, err
	}
	discovered := make([]DiscoveredOrg, 0)
	if resp.DiscoveredOrganizations != nil {
		for _, org := range resp.DiscoveredOrganizations {
			orgID := ""
			memID := ""
			if org.Organization != nil {
				orgID = org.Organization.OrganizationID
			}
			if org.Membership != nil && org.Membership.Member != nil {
				memID = org.Membership.Member.MemberID
			}
			discovered = append(discovered, DiscoveredOrg{
				OrganizationID: orgID,
				MemberID:       memID,
			})
		}
	}
	return &DiscoveryAuthResponse{
		Email:                      resp.EmailAddress,
		IntermediateSessionToken:   resp.IntermediateSessionToken,
		ExpiresAt:                  "",
		DiscoveredOrganizations:    discovered,
	}, nil
}

func (c *SDKClient) ExchangeDiscoverySession(ctx context.Context, intermediateSessionToken string) (*DiscoveryExchangeResponse, error) {
	if c.api == nil {
		return nil, context.DeadlineExceeded
	}
	resp, err := c.api.Discovery.IntermediateSessions.Exchange(ctx, &intermediatesessions.ExchangeParams{
		IntermediateSessionToken: intermediateSessionToken,
	})
	if err != nil {
		return nil, err
	}
	return &DiscoveryExchangeResponse{
		SessionToken: resp.SessionToken,
		ExpiresAt:    "",
	}, nil
}

func (c *SDKClient) CreateOrganizationAndMember(ctx context.Context, name string, email string) (*OrgMemberResponse, error) {
	if c.api == nil {
		return nil, context.DeadlineExceeded
	}
	slug := generateOrganizationSlug(name)
	orgResp, err := c.api.Organizations.Create(ctx, &organizations.CreateParams{
		OrganizationName: name,
		OrganizationSlug: slug,
	})
	if err != nil {
		return nil, err
	}
	memberResp, err := c.api.Organizations.Members.Create(ctx, &members.CreateParams{
		OrganizationID: orgResp.Organization.OrganizationID,
		EmailAddress:   email,
	})
	if err != nil {
		return nil, err
	}
	return &OrgMemberResponse{
		OrganizationID: orgResp.Organization.OrganizationID,
		MemberID:       memberResp.MemberID,
	}, nil
}

func (c *SDKClient) RevokeMemberSession(ctx context.Context, memberID string, sessionID string) error {
	if c.api == nil {
		return context.DeadlineExceeded
	}
	_, err := c.api.Sessions.Revoke(ctx, &sessions.RevokeParams{
		MemberID:        memberID,
		MemberSessionID: sessionID,
	})
	return err
}

// generateOrganizationSlug creates a Stytch-compatible organization slug from a name.
// Slugs must be 2-128 chars, alphanumeric plus - . _ ~, and contain at least one alphanumeric.
func generateOrganizationSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Replace disallowed characters with hyphen
	re := regexp.MustCompile(`[^a-z0-9\-._~]+`)
	s = re.ReplaceAllString(s, "-")
	// Collapse consecutive hyphens
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	// Trim leading/trailing reserved characters
	s = strings.Trim(s, "-._~")
	// Ensure minimum length
	if len(s) < 2 {
		s = "org"
	}
	// Ensure at least one alphanumeric character
	if !regexp.MustCompile(`[a-z0-9]`).MatchString(s) {
		s = "org-" + s
	}
	// Truncate base to leave room for random suffix
	baseMax := 120
	if len(s) > baseMax {
		s = s[:baseMax]
		s = strings.TrimRight(s, "-._~")
	}
	// Append random suffix to ensure uniqueness
	suffix, _ := randomHex(3)
	s = s + "-" + suffix
	// Final trim to max 128 chars
	if len(s) > 128 {
		s = s[:128]
		s = strings.TrimRight(s, "-._~")
	}
	return s
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	b2bintermediatesessions "github.com/stytchauth/stytch-go/v18/stytch/b2b/discovery/intermediatesessions"
	b2bdiscoveryorg "github.com/stytchauth/stytch-go/v18/stytch/b2b/discovery/organizations"
	b2bdiscovery "github.com/stytchauth/stytch-go/v18/stytch/b2b/magiclinks/discovery"
)

type mockErr struct{}

func (mockErr) Error() string { return "mock error" }

type mockStytchClient struct {
	newUser bool
}

func (m *mockStytchClient) SendMagicLink(ctx context.Context, email string) error { return nil }
func (m *mockStytchClient) AuthenticateDiscovery(ctx context.Context, token string) (*b2bdiscovery.AuthenticateResponse, error) {
	return &b2bdiscovery.AuthenticateResponse{IntermediateSessionToken: "ist-test", EmailAddress: "alice@example.com"}, nil
}
func (m *mockStytchClient) CreateDiscoveryOrganization(ctx context.Context, ist, name, slug string) (*b2bdiscoveryorg.CreateResponse, error) {
	return &b2bdiscoveryorg.CreateResponse{MemberID: "m", MemberAuthenticated: true, SessionToken: "s"}, nil
}
func (m *mockStytchClient) ExchangeWithOrg(ctx context.Context, intermediateToken, orgID string) (*b2bintermediatesessions.ExchangeResponse, error) {
	return &b2bintermediatesessions.ExchangeResponse{MemberID: "m", MemberAuthenticated: true, SessionToken: "s"}, nil
}

func TestMockStytchClient_BothBranches(t *testing.T) {
	newUser := &mockStytchClient{newUser: true}
	existing := &mockStytchClient{newUser: false}
	assert.NotNil(t, newUser)
	require.True(t, newUser.newUser)
	assert.NotNil(t, existing)
	require.False(t, existing.newUser)
}

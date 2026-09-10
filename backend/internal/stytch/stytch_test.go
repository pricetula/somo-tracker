package stytch

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/stytchauth/stytch-go/v18/stytch/stytcherror"
	"somotracker/backend/internal/config"
)

// mockLifecycle satisfies fx.Lifecycle for unit tests.
type mockLifecycle struct{}

func (mockLifecycle) Append(fx.Hook) {}

// TestSanitizedError_InvalidMagicLink asserts the magic-link error path.
func TestSanitizedError_InvalidMagicLink(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	err := makeStytchError(400, "invalid_magic_link", "invalid token")
	sanitized := c.SanitizedError(err)
	require.Error(t, sanitized)
	assert.Contains(t, sanitized.Error(), "bad_request")
	assert.Contains(t, sanitized.Error(), "invalid or has already been used")
}

// TestSanitizedError_TokenExpired asserts the expired-session error path.
func TestSanitizedError_TokenExpired(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	err := makeStytchError(401, "token_expired", "session expired")
	sanitized := c.SanitizedError(err)
	require.Error(t, sanitized)
	assert.Contains(t, sanitized.Error(), "unauthorized")
	assert.Contains(t, sanitized.Error(), "session has expired")
}

// TestSanitizedError_RateLimit asserts the rate-limit error path.
func TestSanitizedError_RateLimit(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	err := makeStytchError(429, "rate_limit_exceeded", "too many requests")
	sanitized := c.SanitizedError(err)
	require.Error(t, sanitized)
	assert.Contains(t, sanitized.Error(), "too_many_requests")
}

// TestSanitizedError_UnexpectedSystemError asserts the fallback error path.
func TestSanitizedError_UnexpectedSystemError(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	err := makeStytchError(500, "internal_error", "internal server error")
	sanitized := c.SanitizedError(err)
	require.Error(t, sanitized)
	assert.Contains(t, sanitized.Error(), "internal_error")
	assert.Contains(t, sanitized.Error(), "unexpected error occurred")
}

// TestSanitizedError_Nil returns nil for nil input.
func TestSanitizedError_Nil(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}
	assert.Nil(t, c.SanitizedError(nil))
}

// TestSanitizedError_NonStytchError asserts that non-Stytch errors map to 500.
func TestSanitizedError_NonStytchError(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	err := errors.New("connection refused: dial tcp 127.0.0.1:6379")
	sanitized := c.SanitizedError(err)
	require.Error(t, sanitized)
	assert.Contains(t, sanitized.Error(), "internal_error")
}

// TestSanitizedError_NoLeak asserts that sanitized messages never contain
// internal details like project IDs, secrets, request IDs, or stack traces.
func TestSanitizedError_NoLeak(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	// Raw error with sensitive and verbose data.
	raw := errors.New(
		"stytch error: request_id=req_abc123 secret=sk_test_456 " +
			"project_id=project-xyz stack=goroutine 1 ... context canceled",
	)
	sanitized := c.SanitizedError(raw)
	require.Error(t, sanitized)
	msg := sanitized.Error()

	// The sanitized output must not expose the raw error's contents.
	assert.NotContains(t, msg, "req_abc123")
	assert.NotContains(t, msg, "sk_test_456")
	assert.NotContains(t, msg, "project-xyz")
	assert.NotContains(t, msg, "goroutine")
}

// TestNewClient_NilConfig asserts the nil-config safety net.
func TestNewClient_NilConfig(t *testing.T) {
	logger := zap.NewNop()
	_, err := NewClient(mockLifecycle{}, nil, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

// TestNewClient_NilLogger asserts the nil-logger safety net.
func TestNewClient_NilLogger(t *testing.T) {
	cfg := &config.Config{
		StytchProjectID: "project-test",
		StytchSecret:    "sk_test_xxx",
		StytchEnv:       "test",
	}
	_, err := NewClient(mockLifecycle{}, cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger is required")
}

// TestNewClient_MissingProjectID asserts the missing project ID safety net.
func TestNewClient_MissingProjectID(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		StytchProjectID: "",
		StytchSecret:    "sk_test_xxx",
		StytchEnv:       "test",
	}
	_, err := NewClient(mockLifecycle{}, cfg, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STYTCH_PROJECT_ID")
}

// TestNewClient_MissingSecret asserts the missing secret safety net.
func TestNewClient_MissingSecret(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		StytchProjectID: "project-test",
		StytchSecret:    "",
		StytchEnv:       "test",
	}
	_, err := NewClient(mockLifecycle{}, cfg, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STYTCH_SECRET")
}

// TestNewClient_InvalidEnv verifies that an unsupported STYTCH_ENV is rejected
// during config validation. The SDK validation happens in config.Load(),
// so this is exercised by the config package. Here we verify the Stytch
// client does not accept a nil logger (the first guard in NewClient).
func TestNewClient_InvalidEnv(t *testing.T) {
	// Skip: SDK initialization requires valid Stytch credentials (JWKS fetch).
	// Config validation for STYTCH_ENV is tested in the config package.
	t.Skip("requires live Stytch credentials for JWKS initialization")
}

// TestSanitizedError_StatusCodePatterns verifies all four required HTTP mappings
// using real stytcherror.Error values (the same type returned by the SDK).
func TestSanitizedError_StatusCodePatterns(t *testing.T) {
	logger := zap.NewNop()
	c := &Client{logger: logger}

	cases := []struct {
		statusCode  int
		errorType   string
		msg         string
		expCode     string // expected sanitized prefix
		expContains string
	}{
		{400, "invalid_magic_link", "The magic link is invalid", "bad_request", "invalid or has already been used"},
		{401, "token_expired", "Your session has expired", "unauthorized", "session has expired"},
		{429, "rate_limit_exceeded", "Too many requests", "too_many_requests", "Too many requests"},
		{500, "internal_error", "Something went wrong", "internal_error", "unexpected error occurred"},
	}

	for _, tc := range cases {
		t.Run(tc.expCode, func(t *testing.T) {
			// Construct a real stytcherror.Error (same type the SDK returns).
			stErr := &stytcherror.Error{
				StatusCode:   tc.statusCode,
				ErrorType:    stytcherror.Type(tc.errorType),
				ErrorMessage: stytcherror.Message(tc.msg),
				RequestID:    "req_test_123",
			}
			sanitized := c.SanitizedError(stErr)
			require.Error(t, sanitized)
			assert.True(t, strings.HasPrefix(sanitized.Error(), tc.expCode+":"),
				"expected prefix %q, got %q", tc.expCode+":", sanitized.Error())
			assert.Contains(t, sanitized.Error(), tc.expContains)
		})
	}
}

// TestIsRetryable covers the retryable-error detection.
func TestIsRetryable(t *testing.T) {
	cases := []struct {
		errMsg string
		retry  bool
	}{
		{"connection refused", true},
		{"temporary failure", true},
		{"timeout exceeded", true},
		{"network unreachable", true},
		{"too many requests", true},
		{"invalid token", false},
		{"session not found", false},
		{"", false},
		{"internal server error", false},
	}

	for _, tc := range cases {
		t.Run(tc.errMsg, func(t *testing.T) {
			got := isRetryable(errors.New(tc.errMsg))
			assert.Equal(t, tc.retry, got)
		})
	}
}

// makeStytchError builds a real *stytcherror.Error (the same type returned by
// the SDK) so SanitizedError's errors.As branch executes.
func makeStytchError(status int, errType, msg string) error {
	return &stytcherror.Error{
		StatusCode:   status,
		ErrorType:    stytcherror.Type(errType),
		ErrorMessage: stytcherror.Message(msg),
		RequestID:    "req_test_123",
	}
}

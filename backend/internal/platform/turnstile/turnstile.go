// Package turnstile verifies Cloudflare Turnstile tokens.
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// siteverifyURL is Cloudflare's Turnstile siteverify endpoint.
const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// defaultTimeout is the timeout for siteverify requests.
const defaultTimeout = 5 * time.Second

// ErrUpstream is returned when the siteverify call cannot reach Cloudflare
// or times out. Callers should treat this as a 502 upstream failure.
var ErrUpstream = errors.New("turnstile upstream unavailable")

// ErrVerificationFailed is returned when Cloudflare reports success: false.
// Callers should treat this as a 403 failed verification.
var ErrVerificationFailed = errors.New("turnstile verification failed")

// Result is the parsed JSON response from Cloudflare's siteverify endpoint.
type Result struct {
	Success     bool      `json:"success"`
	ChallengeTS string    `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

// Verifier verifies Turnstile tokens with Cloudflare.
type Verifier interface {
	// Verify sends the token to Cloudflare for verification.
	// Returns ErrUpstream if the siteverify call fails or times out.
	// Returns ErrVerificationFailed if Cloudflare reports success: false
	// or if the hostname does not match the expected host.
	Verify(ctx context.Context, token, remoteIP string) (*Result, error)
}

// httpClient is the subset of *http.Client used by the verifier.
// It allows tests to inject a mock.
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Config holds Turnstile verification configuration.
type Config struct {
	// SecretKey is the Turnstile secret key (server-side only).
	SecretKey string
	// ExpectedHost is the expected hostname in the verification response.
	// If empty, the hostname check is skipped.
	ExpectedHost string
	// HTTPClient is the HTTP client used for siteverify calls.
	// If nil, a default client with a 5-second timeout is used.
	HTTPClient httpClient
	// Timeout is the request timeout for siteverify calls.
	// If zero, defaultTimeout is used.
	Timeout time.Duration
}

// verifier implements Verifier using Cloudflare's siteverify endpoint.
type verifier struct {
	cfg Config
}

// New creates a Turnstile Verifier.
func New(cfg Config) Verifier {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	return &verifier{cfg: cfg}
}

// Verify sends the token to Cloudflare's siteverify endpoint.
func (v *verifier) Verify(ctx context.Context, token, remoteIP string) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, v.cfg.Timeout)
	defer cancel()

	data := url.Values{}
	data.Set("secret", v.cfg.SecretKey)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", siteverifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, ErrUpstream
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.cfg.HTTPClient.Do(req)
	if err != nil {
		// Context deadline exceeded or network error -> upstream failure
		return nil, ErrUpstream
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrUpstream
	}

	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ErrUpstream
	}

	if !result.Success {
		return &result, ErrVerificationFailed
	}

	if v.cfg.ExpectedHost != "" && result.Hostname != v.cfg.ExpectedHost {
		return &result, ErrVerificationFailed
	}

	return &result, nil
}

// Ensure verifier implements Verifier.
var _ Verifier = (*verifier)(nil)

package turnstile

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient is an injectable HTTP client for testing.
type mockHTTPClient struct {
	doFunc func(*http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func newTestVerifier(secretKey, expectedHost string, timeout time.Duration, doFunc func(*http.Request) (*http.Response, error)) *verifier {
	return &verifier{
		cfg: Config{
			SecretKey:    secretKey,
			ExpectedHost: expectedHost,
			HTTPClient:   &mockHTTPClient{doFunc: doFunc},
			Timeout:      timeout,
		},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func readBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer func() { _ = body.Close() }()
	b, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(b)
}

func TestVerify_Success(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "POST", req.Method)
		require.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		body := readBody(t, req.Body)
		assert.Contains(t, body, "secret=secret")
		assert.Contains(t, body, "response=test-token")
		assert.Contains(t, body, "remoteip=1.2.3.4")
		return jsonResponse(200, `{"success":true,"challenge_ts":"2024-01-01T00:00:00Z","hostname":"example.com"}`), nil
	})

	result, err := v.Verify(ctx, "test-token", "1.2.3.4")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "example.com", result.Hostname)
}

func TestVerify_FailedVerification(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"success":false,"error-codes":["invalid-input-response"]}`), nil
	})

	result, err := v.Verify(ctx, "bad-token", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVerificationFailed)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorCodes, "invalid-input-response")
}

func TestVerify_Timeout(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", time.Millisecond, func(req *http.Request) (*http.Response, error) {
		// Block on the request's context (which has a 1ms timeout)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	_, err := v.Verify(ctx, "token", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestVerify_NetworkError(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})

	_, err := v.Verify(ctx, "token", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestVerify_MalformedResponse(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `not json`), nil
	})

	_, err := v.Verify(ctx, "token", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestVerify_HostnameMismatch(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "expected.com", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"success":true,"hostname":"evil.com"}`), nil
	})

	result, err := v.Verify(ctx, "token", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVerificationFailed)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "evil.com", result.Hostname)
}

func TestVerify_HostnameMatch(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "example.com", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"success":true,"hostname":"example.com"}`), nil
	})

	result, err := v.Verify(ctx, "token", "")
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestVerify_NoExpectedHostSkipsCheck(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"success":true,"hostname":"anything.com"}`), nil
	})

	result, err := v.Verify(ctx, "token", "")
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestVerify_EmptyRemoteIPNotSent(t *testing.T) {
	ctx := context.Background()

	v := newTestVerifier("secret", "", 5*time.Second, func(req *http.Request) (*http.Response, error) {
		body := readBody(t, req.Body)
		assert.NotContains(t, body, "remoteip")
		return jsonResponse(200, `{"success":true}`), nil
	})

	_, err := v.Verify(ctx, "token", "")
	require.NoError(t, err)
}

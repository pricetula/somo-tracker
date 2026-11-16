package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"somotracker/internal/platform/config"
	"somotracker/internal/platform/ratelimit"
)

func newRateLimitApp(t *testing.T, ipPerMin int) *fiber.App {
	t.Helper()
	cfg := &config.Config{
		RateLimitMagicLinkIPPerMin:     ipPerMin,
		RateLimitTokenExchangeIPPerMin: ipPerMin,
		RateLimitLogoutIPPerMin:        ipPerMin,
		RateLimitProvisionIPPerMin:     ipPerMin,
		RateLimitPublicIPPerMin:        ipPerMin,
	}
	limiter := ratelimit.New(nil, "") // in-memory fallback
	mw := NewRateLimitMiddleware(limiter, cfg)

	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
		// Honor X-Forwarded-For so per-IP limits are testable. app.Test connects
		// from 127.0.0.1, so trust that as a proxy.
		ProxyHeader:      fiber.HeaderXForwardedFor,
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: []string{"127.0.0.1"}},
	})
	app.Use(mw.PublicRateLimit())
	app.Use(func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func reqWithIP(method, path, ip string) *http.Request {
	req := httptest.NewRequest(method, "http://example.com"+path, nil)
	if ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
	}
	return req
}

func TestPublicRateLimit_AllowsUnderLimit(t *testing.T) {
	app := newRateLimitApp(t, 3)

	ip := "1.2.3.4"
	for i := 0; i < 3; i++ {
		resp, err := app.Test(reqWithIP(http.MethodPost, "/api/v1/auth/request-magic-link", ip), fiber.TestConfig{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "request %d should be allowed", i+1)
	}
}

func TestPublicRateLimit_BlocksOverLimit(t *testing.T) {
	app := newRateLimitApp(t, 2)

	ip := "1.2.3.4"

	// First two allowed
	for i := 0; i < 2; i++ {
		resp, err := app.Test(reqWithIP(http.MethodPost, "/api/v1/auth/request-magic-link", ip), fiber.TestConfig{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Third should be rate limited
	resp, err := app.Test(reqWithIP(http.MethodPost, "/api/v1/auth/request-magic-link", ip), fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "RATE_LIMITED", body["code"])
	require.Equal(t, "Too many requests", body["message"])

	// Retry-After should be set (window = 1 minute)
	require.Equal(t, "60", resp.Header.Get("Retry-After"))
}

func TestPublicRateLimit_EndpointsAreIndependent(t *testing.T) {
	app := newRateLimitApp(t, 1)

	ip := "5.6.7.8"

	// Exhaust magic-link limit
	resp, err := app.Test(reqWithIP(http.MethodPost, "/api/v1/auth/request-magic-link", ip), fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Same IP, different endpoint should still be allowed (separate counters)
	resp, err = app.Test(reqWithIP(http.MethodPost, "/api/v1/auth/authenticate-token", ip), fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// magic-link now exhausted
	resp, err = app.Test(reqWithIP(http.MethodPost, "/api/v1/auth/request-magic-link", ip), fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestPublicRateLimit_DefaultPathUsesPublicLimit(t *testing.T) {
	app := newRateLimitApp(t, 1)

	ip := "9.9.9.9"

	// A non-auth public route consumes the generic "public" counter
	resp, err := app.Test(reqWithIP(http.MethodGet, "/api/v1/reference/countries", ip), fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = app.Test(reqWithIP(http.MethodGet, "/api/v1/reference/countries", ip), fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, "60", resp.Header.Get("Retry-After"))
}

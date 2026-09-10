// Package cors provides CORS middleware tests for the Somotracker API.
package cors

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/config"
)

func newTestApp(t *testing.T, cfg *config.Config) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(Middleware(cfg))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func TestCORSMiddleware_AllowsConfiguredOrigin(t *testing.T) {
	cfg := &config.Config{
		AllowedOrigins: "http://localhost:3000,http://app.somo.io",
	}
	app := newTestApp(t, cfg)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000",
		resp.Header.Get("Access-Control-Allow-Origin"),
		"allowed origin must be echoed back")
}

func TestCORSMiddleware_RejectsUnconfiguredOrigin(t *testing.T) {
	cfg := &config.Config{
		AllowedOrigins: "http://localhost:3000",
	}
	app := newTestApp(t, cfg)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://attacker.example.com")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// The request still succeeds (the origin is just not echoed in CORS
	// headers), but Allow-Origin must NOT match the attacker.
	allow := resp.Header.Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "http://attacker.example.com", allow,
		"unconfigured origin must not be allowed")
}

func TestCORSMiddleware_PreflightAllowedMethods(t *testing.T) {
	cfg := &config.Config{
		AllowedOrigins: "http://localhost:3000",
	}
	app := newTestApp(t, cfg)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	assert.True(t, strings.Contains(allowMethods, "POST"),
		"POST must be allowed, got %q", allowMethods)
	assert.True(t, strings.Contains(allowMethods, "GET"),
		"GET must be allowed, got %q", allowMethods)
	assert.True(t, strings.Contains(allowMethods, "DELETE"),
		"DELETE must be allowed, got %q", allowMethods)
	assert.True(t, strings.Contains(allowMethods, "OPTIONS"),
		"OPTIONS must be allowed, got %q", allowMethods)
}

func TestCORSMiddleware_AllowsCredentials(t *testing.T) {
	cfg := &config.Config{
		AllowedOrigins: "http://localhost:3000",
	}
	app := newTestApp(t, cfg)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"),
		"credentials must be allowed to support session cookies")
}

func TestCORSMiddleware_ExposesXRequestIDHeader(t *testing.T) {
	cfg := &config.Config{
		AllowedOrigins: "http://localhost:3000",
	}
	app := newTestApp(t, cfg)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "X-Request-ID")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	assert.True(t, strings.Contains(strings.ToLower(allowHeaders), "x-request-id"),
		"X-Request-ID must be allowed, got %q", allowHeaders)
}

func TestAllowedOriginsList_ParsesCommaSeparated(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "http://a.example", []string{"http://a.example"}},
		{"multiple", "http://a,http://b,http://c",
			[]string{"http://a", "http://b", "http://c"}},
		{"with_spaces", "http://a, http://b , http://c",
			[]string{"http://a", "http://b", "http://c"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{AllowedOrigins: tc.input}
			assert.Equal(t, tc.expected, cfg.AllowedOriginsList())
		})
	}
}

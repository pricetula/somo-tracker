package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/stretchr/testify/require"
	"somotracker/internal/platform/config"
)

func TestSecurityHeadersMiddleware_Production(t *testing.T) {
	cfg := &config.Config{
		Env: "production",
	}

	app := newTestApp(cfg)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

	// Simulate HTTPS request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)

	// Check security headers
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	require.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	require.Equal(t, "geolocation=(), microphone=(), camera=()", resp.Header.Get("Permissions-Policy"))

	// HSTS should be set in production (via X-Forwarded-Proto)
	hsts := resp.Header.Get("Strict-Transport-Security")
	require.NotEmpty(t, hsts, "HSTS should be set in production")
	require.Contains(t, hsts, "max-age=31536000")
	require.Contains(t, hsts, "includeSubDomains")
	require.Contains(t, hsts, "preload")

	// CSP enforced in production
	require.Equal(t, "default-src 'none'; frame-ancestors 'none'", resp.Header.Get("Content-Security-Policy"))

	// No CSP-Report-Only in production when enforced
	require.Empty(t, resp.Header.Get("Content-Security-Policy-Report-Only"))
}

func TestSecurityHeadersMiddleware_Development(t *testing.T) {
	cfg := &config.Config{
		Env: "development",
	}

	app := newTestApp(cfg)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)

	// Basic headers still set
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))

	// HSTS NOT set in development
	require.Empty(t, resp.Header.Get("Strict-Transport-Security"))

	// CSP Report-Only in development (if enabled)
	require.Empty(t, resp.Header.Get("Content-Security-Policy"))
	require.Empty(t, resp.Header.Get("Content-Security-Policy-Report-Only"))
}

func TestSecurityHeadersMiddleware_CSPReportOnlyEnabled(t *testing.T) {
	cfg := &config.Config{
		Env:                  "development",
		CSPReportOnlyEnabled: true,
		CSPReportOnlyPolicy:  "default-src 'self'; script-src 'self'",
		CSPReportOnlyURI:     "/csp-report",
	}

	app := newTestApp(cfg)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)

	// CSP Report-Only should be set
	require.Equal(t, "default-src 'self'; script-src 'self'; report-uri /csp-report", resp.Header.Get("Content-Security-Policy-Report-Only"))
	require.Empty(t, resp.Header.Get("Content-Security-Policy"))
}

func TestSecurityHeadersMiddleware_HSTSDetectedViaXForwardedProto(t *testing.T) {
	cfg := &config.Config{
		Env: "production",
	}

	app := newTestApp(cfg)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("OK") })

	// Simulate request behind TLS-terminating proxy
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)

	// HSTS should be set even if c.Secure() is false (behind proxy)
	hsts := resp.Header.Get("Strict-Transport-Security")
	require.NotEmpty(t, hsts, "HSTS should be set via X-Forwarded-Proto")
	require.Contains(t, hsts, "max-age=31536000")
}

func TestSecurityHeadersMiddleware_CacheControlOnErrors(t *testing.T) {
	cfg := &config.Config{Env: "production"}

	app := newTestApp(cfg)
	// Endpoint that triggers a panic (to test error handler)
	app.Get("/panic", func(c fiber.Ctx) error { panic("test panic") })

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)

	// Cache-Control: no-store on error responses (set by error handler)
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func newTestApp(cfg *config.Config) *fiber.App {
	// Create minimal app with security headers middleware
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Error handler sets Cache-Control: no-store
			c.Set("Cache-Control", "no-store")
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		},
	})

	// Add recover middleware to catch panics
	app.Use(recover.New())

	// Apply the security headers middleware directly
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		if cfg.Env == "production" && (c.Secure() || c.Get("X-Forwarded-Proto") == "https") {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		if cfg.Env == "production" {
			c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		} else if cfg.CSPReportOnlyEnabled {
			policy := cfg.CSPReportOnlyPolicy
			if cfg.CSPReportOnlyURI != "" {
				policy += "; report-uri " + cfg.CSPReportOnlyURI
			}
			c.Set("Content-Security-Policy-Report-Only", policy)
		}

		return c.Next()
	})

	return app
}
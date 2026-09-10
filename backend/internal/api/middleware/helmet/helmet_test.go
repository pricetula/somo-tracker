// Package helmet provides security header middleware tests.
package helmet

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func TestHelmetMiddleware_SetsSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"),
		"X-Content-Type-Options must be nosniff")
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"),
		"X-Frame-Options must be DENY")
	assert.Equal(t, "default-src 'none'", resp.Header.Get("Content-Security-Policy"),
		"Content-Security-Policy must be default-src 'none'")
}

func TestHelmetMiddleware_PreservesRequestID(t *testing.T) {
	// Ensure helmet does not override X-Request-ID set by upstream middleware.
	app := fiber.New()
	// Simulate request ID middleware by setting header manually before helmet.
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Request-ID", "test-request-id-123")
		return c.Next()
	})
	app.Use(Middleware())
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "test-request-id-123", resp.Header.Get("X-Request-ID"),
		"Helmet middleware must not overwrite X-Request-ID")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
}

func TestHelmetMiddleware_SetsHeadersOnError(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/error", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Headers should still be present even on error responses.
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "default-src 'none'", resp.Header.Get("Content-Security-Policy"))
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"somotracker/backend/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFGuard_GlobalIntegration(t *testing.T) {
	cfg := config.Config{
		AllowedOrigins: "http://localhost:3000,https://app.somotracker.com",
	}

	app := fiber.New()
	app.Use(NewCSRFGuard(cfg))

	// Simulated mutating and safe endpoints
	app.Get("/api/v1/students/list", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Post("/api/v1/students", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusCreated) })
	app.Patch("/api/v1/students/:id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Delete("/api/v1/students/:id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	// Pre-session endpoints that must bypass CSRF
	app.Post("/api/auth/discover", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Post("/api/auth/verify", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Post("/api/auth/register", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	// Helper to execute request
	do := func(method, path string, origin, cookieToken, headerToken string) *http.Response {
		req := httptest.NewRequest(method, path, nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if cookieToken != "" {
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookieToken})
		}
		if headerToken != "" {
			req.Header.Set("X-CSRF-Token", headerToken)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		return resp
	}

	t.Run("Mutating POST missing CSRF token -> 403", func(t *testing.T) {
		resp := do(http.MethodPost, "/api/v1/students", "http://localhost:3000", "", "")
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Mutating PATCH mismatched token -> 403", func(t *testing.T) {
		resp := do(http.MethodPatch, "/api/v1/students/123", "http://localhost:3000", "cookie-abc", "header-xyz")
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Mutating DELETE with valid matching token -> 200", func(t *testing.T) {
		token := "valid-csrf-123"
		resp := do(http.MethodDelete, "/api/v1/students/123", "http://localhost:3000", token, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Safe GET request passes without CSRF tokens", func(t *testing.T) {
		resp := do(http.MethodGet, "/api/v1/students/list", "", "", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Safe HEAD request passes without CSRF tokens", func(t *testing.T) {
		resp := do(http.MethodHead, "/api/v1/students/list", "", "", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Safe OPTIONS request passes without CSRF tokens", func(t *testing.T) {
		resp := do(http.MethodOptions, "/api/v1/students/list", "", "", "")
		// CSRF guard allows OPTIONS; route may not exist, so we only assert it is not blocked by CSRF (403)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Pre-session POST /api/auth/discover bypasses CSRF", func(t *testing.T) {
		resp := do(http.MethodPost, "/api/auth/discover", "", "", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Pre-session POST /api/auth/verify bypasses CSRF", func(t *testing.T) {
		resp := do(http.MethodPost, "/api/auth/verify", "", "", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Pre-session POST /api/auth/register bypasses CSRF", func(t *testing.T) {
		resp := do(http.MethodPost, "/api/auth/register", "", "", "")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Mutating POST with invalid Origin -> 403", func(t *testing.T) {
		token := "valid-token"
		resp := do(http.MethodPost, "/api/v1/students", "https://evil.com", token, token)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Mutating POST with valid Origin and valid tokens -> 201", func(t *testing.T) {
		token := "valid-token"
		resp := do(http.MethodPost, "/api/v1/students", "https://app.somotracker.com", token, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestCSRFGuard_ErrorResponseShape(t *testing.T) {
	cfg := config.Config{
		AllowedOrigins: "http://localhost:3000",
	}

	app := fiber.New()
	app.Use(NewCSRFGuard(cfg))
	app.Post("/api/v1/data", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	// Missing tokens
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Response should be JSON with code, message, request_id
	// (withRequestID is used inside guard)
	// We only assert status here; full shape is covered by middleware error contract tests.
}

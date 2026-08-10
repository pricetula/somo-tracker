package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(newSecurityHeaders())

	// Register dummy endpoints
	app.Get("/api/v1/students", func(c *fiber.Ctx) error {
		return c.SendString("student data")
	})
	app.Post("/api/v1/students", func(c *fiber.Ctx) error {
		return c.SendString("created")
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/apipublic", func(c *fiber.Ctx) error {
		return c.SendString("public")
	})
	app.Get("/web/dashboard", func(c *fiber.Ctx) error {
		return c.SendString("dashboard")
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
		expectCSP      bool
	}{
		{
			name:           "GET /api/* gets baseline headers and CSP",
			method:         http.MethodGet,
			path:           "/api/v1/students",
			expectedStatus: http.StatusOK,
			expectedBody:   "student data",
			expectCSP:      true,
		},
		{
			name:           "POST /api/* gets baseline headers and CSP",
			method:         http.MethodPost,
			path:           "/api/v1/students",
			expectedStatus: http.StatusOK,
			expectedBody:   "created",
			expectCSP:      true,
		},
		{
			name:           "Exact /health route gets baseline headers and CSP",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
			expectCSP:      true,
		},
		{
			name:           "Route /apipublic without trailing slash does not get API CSP",
			method:         http.MethodGet,
			path:           "/apipublic",
			expectedStatus: http.StatusOK,
			expectedBody:   "public",
			expectCSP:      false,
		},
		{
			name:           "Non-API web route gets baseline headers but no CSP",
			method:         http.MethodGet,
			path:           "/web/dashboard",
			expectedStatus: http.StatusOK,
			expectedBody:   "dashboard",
			expectCSP:      false,
		},
		{
			name:           "404 Not Found routes still receive baseline security headers",
			method:         http.MethodGet,
			path:           "/api/v1/nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Cannot GET /api/v1/nonexistent",
			expectCSP:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Read and verify body to confirm c.Next() executed
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(body))

			// Verify global security headers (applied to ALL routes)
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))

			// Verify scoped Content-Security-Policy
			if tt.expectCSP {
				assert.Equal(t, "default-src 'self'", resp.Header.Get("Content-Security-Policy"))
			} else {
				assert.Empty(t, resp.Header.Get("Content-Security-Policy"))
			}

			// Inside TestSecurityHeaders assertions:
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
			assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
			assert.Equal(t, "0", resp.Header.Get("X-XSS-Protection"))
			assert.Equal(t, "camera=(), microphone=(), geolocation=(), payment=()", resp.Header.Get("Permissions-Policy"))
			assert.Equal(t, "same-site", resp.Header.Get("Cross-Origin-Resource-Policy"))
		})
	}
}

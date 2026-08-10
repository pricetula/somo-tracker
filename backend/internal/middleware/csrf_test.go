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

func TestCSRFGuard(t *testing.T) {
	cfg := config.Config{
		AllowedOrigins: "http://localhost:3000,https://app.somotracker.com",
	}

	app := fiber.New()
	app.Use(newCSRFGuard(cfg))

	app.Get("/api/v1/data", func(c *fiber.Ctx) error { return c.SendString("read ok") })
	app.Post("/api/v1/data", func(c *fiber.Ctx) error { return c.SendString("write ok") })
	app.Post("/api/auth/verify", func(c *fiber.Ctx) error { return c.SendString("auth ok") })

	tests := []struct {
		name           string
		method         string
		path           string
		origin         string
		cookieToken    string
		headerToken    string
		expectedStatus int
	}{
		{
			name:           "Safe GET request bypasses CSRF check",
			method:         http.MethodGet,
			path:           "/api/v1/data",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Public auth endpoint bypasses CSRF check on POST",
			method:         http.MethodPost,
			path:           "/api/auth/verify",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST fails with unauthorized Origin header",
			method:         http.MethodPost,
			path:           "/api/v1/data",
			origin:         "https://malicious-site.com",
			cookieToken:    "valid-token",
			headerToken:    "valid-token",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST passes with valid Origin header",
			method:         http.MethodPost,
			path:           "/api/v1/data",
			origin:         "http://localhost:3000",
			cookieToken:    "valid-token",
			headerToken:    "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST fails when CSRF tokens are completely missing",
			method:         http.MethodPost,
			path:           "/api/v1/data",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST fails on token mismatch",
			method:         http.MethodPost,
			path:           "/api/v1/data",
			cookieToken:    "token-aaa",
			headerToken:    "token-bbb",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST passes with matching CSRF cookie and header",
			method:         http.MethodPost,
			path:           "/api/v1/data",
			cookieToken:    "valid-csrf-token",
			headerToken:    "valid-csrf-token",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "csrf_token", Value: tt.cookieToken})
			}
			if tt.headerToken != "" {
				req.Header.Set("X-CSRF-Token", tt.headerToken)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

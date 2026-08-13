package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestID_GeneratesWhenAbsent(t *testing.T) {
	app := fiber.New()
	app.Use(NewRequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	assert.NotEmpty(t, resp.Header.Get(RequestIDHeader))
}

func TestNewRequestID_PropagatesIncoming(t *testing.T) {
	app := fiber.New()
	app.Use(NewRequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "edge-proxy-id-42")
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	assert.Equal(t, "edge-proxy-id-42", resp.Header.Get(RequestIDHeader))
}

func TestNewRequestID_RejectsMalformedIncoming(t *testing.T) {
	app := fiber.New()
	app.Use(NewRequestID())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "bad id!") // chars outside [A-Za-z0-9._-]
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	id := resp.Header.Get(RequestIDHeader)
	assert.NotEqual(t, "bad id!", id)
	assert.NotEmpty(t, id)
}

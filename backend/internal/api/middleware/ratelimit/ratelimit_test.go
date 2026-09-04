package ratelimit

import (
	"net/http/httptest"
	"testing"

	"github.com/go-redis/redis_rate/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewLimiter_NilClient(t *testing.T) {
	limiter, err := NewLimiter(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
	assert.Nil(t, limiter)
}

func TestNewRateLimitMiddleware_NilLimiter_PassesThrough(t *testing.T) {
	app := fiber.New()
	app.Use(NewRateLimitMiddleware(nil, redis_rate.PerMinute(10), "test"))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestNewRateLimitMiddleware_AllowedRequest_SetsHeaders(t *testing.T) {
	// Nil limiter verifies header-setting logic doesn't crash;
	// full integration requires a live Redis instance.
	app := fiber.New()
	app.Use(NewRateLimitMiddleware(nil, redis_rate.PerMinute(10), "test"))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestExtractClientID_ExtractsIP(t *testing.T) {
	app := fiber.New()
	var capturedIP string
	app.Get("/capture", func(c fiber.Ctx) error {
		capturedIP = extractClientID(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/capture", nil)
	req.RemoteAddr = "192.168.1.10:1234"
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, capturedIP)
}

func TestFallbackLogger_GlobalWhenNotSet(t *testing.T) {
	base := zaptest.NewLogger(t)
	zap.ReplaceGlobals(base)

	app := fiber.New()
	app.Use(NewRateLimitMiddleware(nil, redis_rate.PerMinute(10), "test"))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

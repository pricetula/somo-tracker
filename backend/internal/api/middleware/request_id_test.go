package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func newTestApp(t testing.TB, baseLogger *zap.Logger) *fiber.App {
	app := fiber.New()
	app.Use(RequestIDMiddleware(baseLogger))
	app.Get("/ping", func(c fiber.Ctx) error {
		// Read the values back out of Fiber locals + context.
		rid := GetRequestID(c)
		lg := GetLogger(c)
		if rid == "" {
			t.Errorf("request id must be present in locals")
		}
		if lg == nil {
			t.Errorf("logger must be present in locals")
		}
		// Also assert they are accessible via c.Context() (standard ctx).
		if rid != RequestID(c.Context()) {
			t.Errorf("request id mismatch via context")
		}
		if lg != Logger(c.Context()) {
			t.Errorf("logger mismatch via context")
		}
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func TestRequestIDMiddleware_GeneratesUUIDWhenMissing(t *testing.T) {
	base := zaptest.NewLogger(t)
	app := newTestApp(t, base)

	req := httptest.NewRequest("GET", "/ping", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	headerID := resp.Header.Get("X-Request-ID")
	assert.NotEmpty(t, headerID, "response must include X-Request-ID header")

	// Header value should be a valid UUID.
	_, parseErr := uuid.Parse(headerID)
	assert.NoError(t, parseErr, "generated request id should be a valid UUID")
}

func TestRequestIDMiddleware_ReusesIncomingHeader(t *testing.T) {
	base := zaptest.NewLogger(t)
	app := newTestApp(t, base)

	const incoming = "my-trace-123"
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-Request-ID", incoming)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, incoming, resp.Header.Get("X-Request-ID"),
		"middleware must echo the inbound X-Request-ID")
}

func TestRequestIDMiddleware_StoresInContext(t *testing.T) {
	base := zaptest.NewLogger(t)

	// We use a sentinel handler to capture the request-scoped logger.
	var capturedLogger *zap.Logger
	var capturedCtx context.Context

	app := fiber.New()
	app.Use(RequestIDMiddleware(base))
	app.Get("/capture", func(c fiber.Ctx) error {
		capturedLogger = GetLogger(c)
		capturedCtx = c.Context()
		return c.SendStatus(fiber.StatusOK)
	})

	const incoming = "ctx-trace-456"
	req := httptest.NewRequest("GET", "/capture", nil)
	req.Header.Set("X-Request-ID", incoming)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.NotNil(t, capturedLogger, "logger must be injected into context")
	assert.Equal(t, incoming, RequestID(capturedCtx),
		"request id must propagate via c.Context()")
	assert.NotSame(t, base, capturedLogger,
		"request-scoped logger must be a derived instance, not the base")
}

func TestNewRequestIDHandler_ResolvesLogger(t *testing.T) {
	base := zaptest.NewLogger(t)
	h := NewRequestIDHandler(base)
	require.NotNil(t, h, "NewRequestIDHandler must return a non-nil handler")

	app := fiber.New()
	app.Use(h)
	app.Get("/ok", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/ok", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

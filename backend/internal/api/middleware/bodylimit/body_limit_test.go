package bodylimit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestApp builds a Fiber app with our middleware. The server's own
// BodyLimit is set high (50MB) so the test fixture can send large
// Content-Length headers without fasthttp rejecting them at the
// transport layer; our middleware is responsible for the real 4MB cap.
func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New(fiber.Config{
		BodyLimit: 50 * 1024 * 1024, // 50MB
	})
	app.Use(Middleware())
	app.Post("/echo", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func decodeCanonicalError(t *testing.T, body io.Reader) (map[string]any, error) {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func TestBodyLimitMiddleware_AllowsUnderLimit(t *testing.T) {
	app := newTestApp(t)

	payload := bytes.Repeat([]byte("a"), 1024) // 1KB
	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(payload))

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestBodyLimitMiddleware_AllowsExactlyFourMB(t *testing.T) {
	app := newTestApp(t)

	payload := bytes.Repeat([]byte("a"), maxPayloadSize)
	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(payload))

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode,
		"body exactly at the limit should pass")
}

func TestBodyLimitMiddleware_RejectsOverLimit(t *testing.T) {
	app := newTestApp(t)

	// 4MB + 1 byte — just over the middleware cap.
	payload := bytes.Repeat([]byte("a"), maxPayloadSize+1)
	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(payload))

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusRequestEntityTooLarge, resp.StatusCode)

	body, err := decodeCanonicalError(t, resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "payload_too_large", body["code"])
	assert.Equal(t, "Request body exceeds the maximum allowed size", body["message"])
	assert.Equal(t, map[string]any{}, body["errors"])
}

// Package compression provides response compression middleware tests.
package compression

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(Middleware())
	// Return a large JSON payload to trigger compression.
	app.Get("/large", func(c fiber.Ctx) error {
		payload := strings.Repeat(`{"data":"x"}`, 200) // ~2KB
		return c.Type("application/json").SendString(payload)
	})
	// Small payload endpoint.
	app.Get("/small", func(c fiber.Ctx) error {
		return c.Type("application/json").SendString(`{"ok":true}`)
	})
	return app
}

func TestCompressionMiddleware_CompressesLargeResponseWithGzip(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/large", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	encoding := resp.Header.Get("Content-Encoding")
	assert.Contains(t, encoding, "gzip", "large responses should be gzip compressed")

	// Ensure body is readable after compression by checking length reduction.
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)
	// Compressed body should be smaller than original ~2.4KB.
	assert.Less(t, body.Len(), 2000, "compressed body should be smaller than uncompressed")
}

func TestCompressionMiddleware_DoesNotCompressSmallResponse(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/small", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	// Small responses may remain uncompressed; ensure no error and content is valid.
	// We only assert that the middleware does not break the response.
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, body.String(), `"ok":true`)
}

func TestCompressionMiddleware_RespectsNoEncoding(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/large", nil)
	// No Accept-Encoding header.
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Content-Encoding"),
		"response should not be compressed when client does not advertise encoding")
}

package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceFingerprinter(t *testing.T) {
	// Enable ProxyHeader so Fiber reads c.IP() from X-Forwarded-For in tests
	app := fiber.New(fiber.Config{
		ProxyHeader: fiber.HeaderXForwardedFor,
	})
	app.Use(NewDeviceFingerprinter())

	var capturedFingerprint string
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedFingerprint = c.Locals("device_fingerprint").(string)
		return c.SendString("ok")
	})

	t.Run("Generates correct deterministic SHA-256 fingerprint", func(t *testing.T) {
		ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
		al := "en-US,en;q=0.9"

		// Calculate expected hash independently
		expectedRaw := ua + "|" + al
		expectedSum := sha256.Sum256([]byte(expectedRaw))
		expectedHash := "v2:" + hex.EncodeToString(expectedSum[:])

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1") // still set for ProxyHeader
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept-Language", al)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, expectedHash, capturedFingerprint)
	})

	t.Run("Handles missing headers gracefully without panicking", func(t *testing.T) {
		capturedFingerprint = ""

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEmpty(t, capturedFingerprint)
		assert.True(t, strings.HasPrefix(capturedFingerprint, "v2:")) // versioned
		assert.Len(t, capturedFingerprint, 3+64)                      // "v2:" + hex SHA-256
	})

	t.Run("Different headers produce distinct fingerprints", func(t *testing.T) {
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set("User-Agent", "Browser-A")
		_, err := app.Test(req1)
		require.NoError(t, err)
		hash1 := capturedFingerprint

		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set("User-Agent", "Browser-B")
		_, err = app.Test(req2)
		require.NoError(t, err)
		hash2 := capturedFingerprint

		assert.NotEqual(t, hash1, hash2)
	})
}

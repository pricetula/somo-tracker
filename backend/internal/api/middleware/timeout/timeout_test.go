package timeout

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutMiddleware_AllowsFastHandler(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/fast", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/fast", nil)
	req.Header.Set("X-Request-ID", "test-fast-001")

	// Give Test a generous timeout so we don't time out the test itself.
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTimeoutMiddleware_RejectsSlowHandler(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/slow", func(c fiber.Ctx) error {
		// Deliberately sleep longer than the 15-second timeout.
		time.Sleep(20 * time.Second)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	req.Header.Set("X-Request-ID", "test-slow-001")

	// Give the test itself a long deadline so the timeout fires in the app.
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusGatewayTimeout, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(bodyBytes)
	assert.Contains(t, bodyStr, `"code":"gateway_timeout"`)
	assert.Contains(t, bodyStr, `"message":"Gateway timeout"`)
}

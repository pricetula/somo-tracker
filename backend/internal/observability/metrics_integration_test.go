package observability

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/api/middleware"
)

func TestMetricsEndpoint_DoesNotBreakRequestIDMiddleware(t *testing.T) {
	// Regression: /metrics must not interfere with request-id middleware.
	app := fiber.New()
	logger, _ := zap.NewDevelopment()
	app.Use(middleware.NewRequestIDHandler(logger))
	RegisterMetrics(app)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("X-Request-ID", "regression-test-123")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "regression-test-123", resp.Header.Get("X-Request-ID"), "metrics route must preserve request-id middleware")
}

func TestMetricsEndpoint_AvailableAndContainsStandardMetrics(t *testing.T) {
	app := fiber.New()
	RegisterMetrics(app)

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 200, resp.StatusCode, "/metrics must return 200 OK")

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)

	assert.Contains(t, body, "go_goroutines", "metrics output must contain standard Go runtime metric")
	assert.Contains(t, body, "process_cpu_seconds_total", "metrics output must contain process metric")
}

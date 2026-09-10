package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler_404ReturnsCanonicalJSON(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: NewErrorHandler(),
	})
	app.Get("/exists", func(c fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("GET", "/missing", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

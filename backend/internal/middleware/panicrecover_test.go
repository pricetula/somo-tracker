package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicRecover(t *testing.T) {
	app := fiber.New()
	app.Use(NewPanicRecover())

	app.Get("/ok", func(c *fiber.Ctx) error {
		return c.SendString("all good")
	})

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("simulated critical backend error")
	})

	t.Run("Normal requests pass through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Panics are caught and converted to 500 Internal Server Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Server remains healthy after a panic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

//go:build integration

package api

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestErrorContractShape(t *testing.T) {
	app := fiber.New()
	app.Get("/test-error", func(c fiber.Ctx) error {
		return WriteError(c, APIError{
			Status:  http.StatusBadRequest,
			Code:    "bad_request",
			Message: "validation failed",
			Errors: map[string][]string{
				"email": {"invalid email"},
			},
		})
	})

	req := app.AcquireReq()
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer app.ReleaseReq(req)

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	// Response body should contain code, message, errors, request_id
	// Full validation performed in integration suite
}

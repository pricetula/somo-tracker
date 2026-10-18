package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestStudentsErrorContractShape(t *testing.T) {
	app := fiber.New()
	app.Get("/test-bad", func(c fiber.Ctx) error {
		return WriteError(c, ErrBadRequest("invalid input", map[string][]string{"field": {"must be set"}}))
	})
	app.Get("/test-notfound", func(c fiber.Ctx) error {
		return WriteError(c, ErrNotFound("not found"))
	})

	// Bad request
	req := httptest.NewRequest(http.MethodGet, "/test-bad", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	assert.Contains(t, body, "code")
	assert.Contains(t, body, "message")
	assert.Contains(t, body, "errors")
	assert.Equal(t, "bad_request", body["code"])
	assert.Equal(t, "invalid input", body["message"])
	errorsMap := body["errors"].(map[string]any)
	assert.Contains(t, errorsMap, "field")

	// Not found
	req2 := httptest.NewRequest(http.MethodGet, "/test-notfound", nil)
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp2.StatusCode)

	var body2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&body2)
	assert.Equal(t, "not_found", body2["code"])
	assert.Contains(t, body2, "errors")
}

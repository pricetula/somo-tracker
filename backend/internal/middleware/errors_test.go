package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/xerrors"
)

// do executes a request against a Fiber app whose handler returns
// HTTPError(c, err) and decodes the canonical response body.
func do(t *testing.T, err error, setup ...func(c *fiber.Ctx)) (int, map[string]any) {
	t.Helper()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		for _, fn := range setup {
			fn(c)
		}
		return HTTPError(c, err)
	})
	resp, rerr := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, rerr)
	t.Cleanup(func() { _ = resp.Body.Close() })
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return resp.StatusCode, body
}

func TestHTTPError_DomainSentinelMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", ErrNotFound, http.StatusNotFound, "not_found"},
		{"already exists", ErrAlreadyExists, http.StatusConflict, "already_exists"},
		{"invalid input", ErrInvalidInput, http.StatusBadRequest, "invalid_input"},
		{"unauthorized", ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
		{"forbidden", ErrForbidden, http.StatusForbidden, "forbidden"},
		{"conflict", ErrConflict, http.StatusConflict, "conflict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := do(t, tt.err)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantCode, body["code"])
			assert.NotEmpty(t, body["message"])
		})
	}
}

func TestHTTPError_WrappedDomainError(t *testing.T) {
	status, body := do(t, fmt.Errorf("members.Service.GetMember: %w", ErrNotFound))
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "not_found", body["code"])
	assert.Contains(t, body["message"], "members.Service.GetMember")
}

func TestHTTPError_FieldErrors(t *testing.T) {
	fe := &FieldError{
		Err:    xerrors.InvalidInput("email is already in use"),
		Fields: map[string][]string{"email": {"email is already in use"}},
	}
	status, body := do(t, fe)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_input", body["code"])
	errs, ok := body["errors"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, errs, "email")
}

func TestHTTPError_InternalDomainErrorIsGeneric(t *testing.T) {
	internal := &xerrors.DomainError{
		Code:    "db_connection_failed",
		Status:  http.StatusInternalServerError,
		Message: "postgres connection refused: secret password",
	}
	status, body := do(t, internal)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "db_connection_failed", body["code"])
	assert.Equal(t, "an unexpected error occurred", body["message"])
}

func TestHTTPError_UnclassifiedError(t *testing.T) {
	status, body := do(t, errors.New("something leaked through"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "internal_error", body["code"])
	assert.Equal(t, "an unexpected error occurred", body["message"])
}

func TestHTTPError_ContextErrors(t *testing.T) {
	status, body := do(t, context.Canceled)
	assert.Equal(t, 499, status)
	assert.Equal(t, "request_canceled", body["code"])

	status, body = do(t, context.DeadlineExceeded)
	assert.Equal(t, http.StatusGatewayTimeout, status)
	assert.Equal(t, "timeout", body["code"])
}

func TestHTTPError_Nil(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return HTTPError(c, nil)
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	// A nil error must not write a response body; the caller's own response
	// (default 200 for a GET handler that returns nil) is what the client sees.
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHTTPError_RequestIDThreadedIntoBody(t *testing.T) {
	status, body := do(t, ErrUnauthorized, func(c *fiber.Ctx) {
		c.Locals(RequestIDKey, "req-123")
	})
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, "req-123", body["request_id"])
}

func TestHTTPError_NoRequestIDNoField(t *testing.T) {
	_, body := do(t, ErrUnauthorized)
	_, present := body["request_id"]
	assert.False(t, present)
}

func TestStatusForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"domain not found", ErrNotFound, http.StatusNotFound},
		{"wrapped domain", fmt.Errorf("wrap: %w", ErrForbidden), http.StatusForbidden},
		{"fiber 404", fiber.ErrNotFound, http.StatusNotFound},
		{"fiber 405", fiber.ErrMethodNotAllowed, http.StatusMethodNotAllowed},
		{"canceled", context.Canceled, 499},
		{"deadline", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"unknown", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, statusForError(tt.err))
		})
	}
}

func TestHTTPError_FiberErrorsReturnJSON(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"fiber 404", fiber.ErrNotFound, http.StatusNotFound, "not_found"},
		{"fiber 405", fiber.ErrMethodNotAllowed, http.StatusMethodNotAllowed, "method_not_allowed"},
		{"fiber 400", fiber.NewError(fiber.StatusBadRequest, "bad request"), fiber.StatusBadRequest, "invalid_input"},
		{"fiber 401", fiber.NewError(fiber.StatusUnauthorized, "unauthorized"), fiber.StatusUnauthorized, "unauthorized"},
		{"fiber 403", fiber.NewError(fiber.StatusForbidden, "forbidden"), fiber.StatusForbidden, "forbidden"},
		{"fiber 409", fiber.NewError(fiber.StatusConflict, "conflict"), fiber.StatusConflict, "conflict"},
		{"fiber 422", fiber.NewError(fiber.StatusUnprocessableEntity, "unprocessable"), fiber.StatusUnprocessableEntity, "unprocessable_entity"},
		{"fiber 413", fiber.NewError(fiber.StatusRequestEntityTooLarge, "too large"), fiber.StatusRequestEntityTooLarge, "request_too_large"},
		{"fiber 429", fiber.NewError(fiber.StatusTooManyRequests, "rate limited"), fiber.StatusTooManyRequests, "rate_limited"},
		{"fiber 500", fiber.NewError(fiber.StatusInternalServerError, "internal"), fiber.StatusInternalServerError, "internal_error"},
		{"fiber 503", fiber.NewError(fiber.StatusServiceUnavailable, "unavailable"), fiber.StatusServiceUnavailable, "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := do(t, tt.err)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantCode, body["code"])
			assert.NotEmpty(t, body["message"])
			// Fiber errors should NOT have the generic "an unexpected error occurred" message
			assert.NotEqual(t, "an unexpected error occurred", body["message"])
		})
	}
}

func TestHTTPError_FallbackToInternal(t *testing.T) {
	// Test various unclassified errors all map to internal_error
	tests := []struct {
		name string
		err  error
	}{
		{"plain error", errors.New("something leaked through")},
		{"wrapped plain error", fmt.Errorf("service.DoThing: %w", errors.New("plain"))},
		{"nil pointer panic recovery", fmt.Errorf("panic recovered: %w", errors.New("nil pointer"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := do(t, tt.err, func(c *fiber.Ctx) {
				c.Locals(RequestIDKey, "req-test")
			})
			assert.Equal(t, http.StatusInternalServerError, status)
			assert.Equal(t, "internal_error", body["code"])
			assert.Equal(t, "an unexpected error occurred", body["message"])
			assert.Equal(t, "req-test", body["request_id"])
		})
	}
}

func TestHTTPError_RequestIDInFiberErrorResponses(t *testing.T) {
	status, body := do(t, fiber.ErrNotFound, func(c *fiber.Ctx) {
		c.Locals(RequestIDKey, "req-404")
	})
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "req-404", body["request_id"])
}

func TestHTTPError_WrappedInternalDomainError_MessageSanitized(t *testing.T) {
	// Wrapped internal error should still have message sanitized
	internal := &xerrors.DomainError{
		Code:    "db_connection_failed",
		Status:  http.StatusInternalServerError,
		Message: "postgres connection refused: secret password",
	}
	wrapped := fmt.Errorf("service.Query: %w", internal)

	status, body := do(t, wrapped)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "db_connection_failed", body["code"])
	assert.Equal(t, "an unexpected error occurred", body["message"])
	// The original message with sensitive info should not appear in response
	assert.NotContains(t, body["message"], "secret password")
}

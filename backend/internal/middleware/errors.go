// =============================================================================
// Canonical Error Response Contract
//
// Every non-2xx HTTP response from the backend MUST return this exact JSON body:
//
//	{
//	  "code":    "snake_case_error_code",
//	  "message": "human readable message",
//	  "errors":  { "field_name": ["Specific field validation message"] }
//	}
//
// code is always a snake_case string the frontend can switch on
// (e.g. "member_not_found", "invalid_member_input", "unauthorized").
// message is a safe, human-readable string. For 500 errors it must be a
// generic string — never an internal detail.
// errors is an optional object populated exclusively on 400 Bad Request /
// validation failures, mapping field keys to an array of specific error
// string messages.
//
// Frontend counterpart: src/lib/api/client.ts
// =============================================================================

package middleware

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// sentinel domain error references — each module declares its own package-level
// sentinels. This file uses errors.Is() to match them in the error chain.
var (
	// ErrNotFound is the canonical not-found sentinel (404).
	// Matched by errors.Is against any module's ErrNotFound.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is the canonical conflict-for-duplicate sentinel (409).
	ErrAlreadyExists = errors.New("already exists")
	// ErrInvalidInput is the canonical validation-failure sentinel (400).
	ErrInvalidInput = errors.New("invalid input")
	// ErrUnauthorized is the canonical unauthenticated sentinel (401).
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is the canonical forbidden sentinel (403).
	ErrForbidden = errors.New("forbidden")
	// ErrConflict is the canonical optimistic-lock / concurrent-mod sentinel (409).
	ErrConflict = errors.New("conflict")
)

// HTTPError is the single place where domain errors are mapped to HTTP status
// codes and JSON response bodies. All handlers must call this function instead
// of duplicating errors.Is / switch logic inline.
//
// It uses errors.Is() to unwrap the full error chain and match sentinels.
// For 500 errors the internal error is logged with slog.ErrorContext and a
// generic message is returned to the client.
//
// Parameters:
//   - c: the Fiber request context (used for logging method + path).
//   - err: the error to map. Must be non-nil.
//
// Returns an error suitable for Fiber's error handler chain.
func HTTPError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	// Build the standard response.
	type errorResponse struct {
		Code    string              `json:"code"`
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors,omitempty"`
	}

	var code string
	var message string
	var status int
	var fieldErrors map[string][]string

	switch {
	case errors.Is(err, ErrNotFound):
		status = fiber.StatusNotFound
		code = "not_found"
		message = "the requested resource was not found"

	case errors.Is(err, ErrAlreadyExists):
		status = fiber.StatusConflict
		code = "already_exists"
		message = "the resource already exists"

	case errors.Is(err, ErrInvalidInput):
		status = fiber.StatusBadRequest
		code = "invalid_input"
		message = err.Error() // surface validation details

		// Check if the error carries field-level validation metadata via
		// the FieldErrors interface.
		var fe interface{ FieldErrors() map[string][]string }
		if errors.As(err, &fe) {
			fieldErrors = fe.FieldErrors()
		}

	case errors.Is(err, ErrUnauthorized):
		status = fiber.StatusUnauthorized
		code = "unauthorized"
		message = "authentication required"

	case errors.Is(err, ErrForbidden):
		status = fiber.StatusForbidden
		code = "forbidden"
		message = "insufficient permissions"

	case errors.Is(err, ErrConflict):
		status = fiber.StatusConflict
		code = "conflict"
		message = "the resource was modified by another request"

	case errors.Is(err, context.Canceled):
		status = 499
		code = "request_canceled"
		message = "the request was canceled"

	case errors.Is(err, context.DeadlineExceeded):
		status = fiber.StatusGatewayTimeout
		code = "timeout"
		message = "the request timed out"

	default:
		// Log the full internal error with context
		status = fiber.StatusInternalServerError
		code = "internal_error"
		message = "an unexpected error occurred"

		handlerFields := []slog.Attr{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("error", err.Error()),
		}
		slog.LogAttrs(context.Background(), slog.LevelError, "HTTPError: internal error", handlerFields...)
	}

	return c.Status(status).JSON(errorResponse{
		Code:    code,
		Message: message,
		Errors:  fieldErrors,
	})
}

// FieldError is a convenience type that carries both a sentinel error and
// field-level validation metadata. Use it in service-layer validation:
//
//	return &middleware.FieldError{
//	    Err:    members.ErrInvalidInput,
//	    Fields: map[string][]string{"email": {"email is already in use"}},
//	}
type FieldError struct {
	Err    error
	Fields map[string][]string
}

func (e *FieldError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "validation error"
}

func (e *FieldError) Unwrap() error { return e.Err }

// FieldErrors implements the field-error extraction interface used by HTTPError.
func (e *FieldError) FieldErrors() map[string][]string { return e.Fields }

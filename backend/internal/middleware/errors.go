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
// errors is an optional field populated exclusively on validation or
// semantic failures (e.g. 400 Bad Request, 422 Unprocessable Entity).
// It can be a map of field→messages or any other structured data
// (e.g. a list of conflicting resources).
//
// Frontend counterpart: src/lib/api/client.ts
// =============================================================================

package middleware

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"somotracker/backend/internal/xerrors"
)

// Sentinel errors for use by middleware (RequireAuth, RequireRole).
// Domain packages define their own sentinels; they don't import middleware.
var (
	ErrNotFound      = xerrors.ErrNotFound
	ErrAlreadyExists = xerrors.ErrAlreadyExists
	ErrInvalidInput  = xerrors.ErrInvalidInput
	ErrUnauthorized  = xerrors.ErrUnauthorized
	ErrForbidden     = xerrors.ErrForbidden
	ErrConflict      = xerrors.ErrConflict
)

// HTTPError is the single place where errors are mapped to HTTP status
// codes and JSON response bodies. All handlers must call this function
// instead of duplicating error-mapping logic inline.
//
// It uses errors.As() to find the nearest *xerrors.DomainError in the
// chain, extracting status, code, message, and optional metadata.
// For 500 errors (unrecognized errors), the internal error is logged
// with slog.ErrorContext and a generic message is returned to the client.
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

	// Try to extract a *xerrors.DomainError from the chain.
	var de *xerrors.DomainError
	if errors.As(err, &de) {
		// Build the response body.
		// Use err.Error() for the message — it includes context added by
		// callers via fmt.Errorf("context: %w", domainErr). For 500 errors
		// we replace it with a generic message to avoid leaking internals.
		resp := fiber.Map{
			"code":    de.Code,
			"message": err.Error(),
		}

		// Check for field-level validation errors (middleware.FieldError or
		// any error implementing FieldErrors() interface).
		type hasFieldErrors interface{ FieldErrors() map[string][]string }
		if fe, ok := err.(hasFieldErrors); ok && len(fe.FieldErrors()) > 0 {
			resp["errors"] = fe.FieldErrors()
		} else if len(de.Fields) > 0 {
			resp["errors"] = de.Fields
		}

		// Check for extra metadata (implemented by custom error types
		// that embed *xerrors.DomainError).
		var details any
		if hd, ok := err.(xerrors.HasDetails); ok {
			details = hd.ErrorDetails()
		}
		if details != nil {
			resp["errors"] = details
		}

		// Log internal/unexpected errors only — the rest are client errors.
		if de.Status == fiber.StatusInternalServerError {
			resp["message"] = "an unexpected error occurred"
			slog.LogAttrs(context.Background(), slog.LevelError,
				"HTTPError: internal error",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("code", de.Code),
				slog.String("error", err.Error()),
			)
		}

		return c.Status(de.Status).JSON(resp)
	}

	// Special cases that aren't DomainErrors.
	switch {
	case errors.Is(err, context.Canceled):
		return c.Status(499).JSON(fiber.Map{
			"code":    "request_canceled",
			"message": "the request was canceled",
		})
	case errors.Is(err, context.DeadlineExceeded):
		return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
			"code":    "timeout",
			"message": "the request timed out",
		})
	default:
		// Log unknown errors — something leaked through without a domain wrapper.
		slog.LogAttrs(context.Background(), slog.LevelError,
			"HTTPError: unclassified error — wrap with xerrors.DomainError",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("error", err.Error()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "an unexpected error occurred",
		})
	}
}

// FieldError is a convenience type for carrying field-level validation
// metadata alongside a domain sentinel. Keep the underlying error as a
// *xerrors.DomainError so that HTTPError can extract the status code.
//
// Usage in service-layer validation:
//
//	return &middleware.FieldError{
//	    Err:    xerrors.InvalidInput("email is already in use"),
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

// FieldErrors returns the field-level validation errors for HTTP response.
func (e *FieldError) FieldErrors() map[string][]string { return e.Fields }

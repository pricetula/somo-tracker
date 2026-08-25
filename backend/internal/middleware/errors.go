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
	"net/http"

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

	// ErrDeviceFingerprintMismatch is returned when a request presents a
	// device fingerprint different from the one recorded when the session was
	// created. Mapped to 401 so the client re-authenticates — a stolen cookie
	// cannot be replayed from a different device (C5).
	ErrDeviceFingerprintMismatch = &xerrors.DomainError{
		Code:    string(xerrors.CodeDeviceFingerprintMismatch),
		Status:  http.StatusUnauthorized,
		Message: "session is bound to a different device; re-authenticate to continue",
	}
)

// HTTPError is the single place where errors are mapped to HTTP status
// codes and JSON response bodies. All handlers must call this function
// instead of duplicating error-mapping logic inline.
//
// It uses errors.As() to find the nearest *xerrors.DomainError in the
// chain, extracting status, code, message, and optional metadata.
// For 500 errors (unrecognized errors), the internal error is logged
// with the request logger (zap) and a generic message is returned to
// the client.
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
		// callers via xerrors.Wrap/Wrapf. For 500 errors we replace it
		// with a generic message to avoid leaking internals.
		resp := withRequestID(c, fiber.Map{
			"code":    de.Code,
			"message": err.Error(),
		})

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
			fields := []interface{}{
				"method", c.Method(),
				"path", c.Path(),
				"code", de.Code,
				"error", err.Error(),
			}
			if rid := GetRequestID(c); rid != "" {
				fields = append(fields, "request_id", rid)
			}
			loggerFrom(c).Errorw("HTTPError: internal error", fields...)
		}

		return c.Status(de.Status).JSON(resp)
	}

	// Special cases that aren't DomainErrors but have standard mappings.
	switch {
	case errors.Is(err, context.Canceled):
		return c.Status(499).JSON(withRequestID(c, fiber.Map{
			"code":    string(xerrors.CodeRequestCanceled),
			"message": "the request was canceled",
		}))
	case errors.Is(err, context.DeadlineExceeded):
		return c.Status(fiber.StatusGatewayTimeout).JSON(withRequestID(c, fiber.Map{
			"code":    string(xerrors.CodeTimeout),
			"message": "the request timed out",
		}))
	}

	// Handle Fiber's built-in HTTP errors (404, 405, etc.) by mapping them
	// to the canonical JSON format instead of returning HTML/text.
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code := fiberErrToCode(fe.Code)
		return c.Status(fe.Code).JSON(withRequestID(c, fiber.Map{
			"code":    code,
			"message": fe.Message,
		}))
	}

	// Unknown error — something leaked through without a domain wrapper.
	// Log it and return a generic 500.
	fields := []interface{}{
		"method", c.Method(),
		"path", c.Path(),
		"error", err.Error(),
	}
	if rid := GetRequestID(c); rid != "" {
		fields = append(fields, "request_id", rid)
	}
	loggerFrom(c).Errorw("HTTPError: unclassified error — wrap with xerrors.DomainError", fields...)
	return c.Status(fiber.StatusInternalServerError).JSON(withRequestID(c, fiber.Map{
		"code":    string(xerrors.CodeInternalError),
		"message": "an unexpected error occurred",
	}))
}

// fiberErrToCode maps Fiber HTTP status codes to our canonical error codes.
func fiberErrToCode(status int) string {
	switch status {
	case fiber.StatusNotFound:
		return string(xerrors.CodeNotFound)
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusBadRequest:
		return string(xerrors.CodeInvalidInput)
	case fiber.StatusUnauthorized:
		return string(xerrors.CodeUnauthorized)
	case fiber.StatusForbidden:
		return string(xerrors.CodeForbidden)
	case fiber.StatusConflict:
		return string(xerrors.CodeConflict)
	case fiber.StatusUnprocessableEntity:
		return string(xerrors.CodeUnprocessableEntity)
	case fiber.StatusRequestEntityTooLarge:
		return "request_too_large"
	case fiber.StatusTooManyRequests:
		return "rate_limited"
	default:
		if status >= 500 {
			return string(xerrors.CodeInternalError)
		}
		return "http_error"
	}
}

// statusForError resolves the HTTP status HTTPError would assign to err
// without writing a response. It mirrors the mapping in HTTPError and is used
// by the access-log middleware to record an accurate status when the handler
// chain returned an error that the global error handler hasn't serialized yet.
func statusForError(err error) int {
	var de *xerrors.DomainError
	if errors.As(err, &de) {
		return de.Status
	}
	switch {
	case errors.Is(err, fiber.ErrNotFound):
		return fiber.StatusNotFound
	case errors.Is(err, fiber.ErrMethodNotAllowed):
		return fiber.StatusMethodNotAllowed
	case errors.Is(err, context.Canceled):
		return 499
	case errors.Is(err, context.DeadlineExceeded):
		return fiber.StatusGatewayTimeout
	default:
		return fiber.StatusInternalServerError
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

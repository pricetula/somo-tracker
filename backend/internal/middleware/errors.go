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
// generic
// — never an internal detail.
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

	"somotracker/backend/internal/telemetry"
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

func handleSpecialErrors(c *fiber.Ctx, err error) error {
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

	// Unknown / unwrapped errors (DB failures, etc.): log and return canonical 500
	loggerFrom(c).Errorw("unhandled error in HTTPError", "error", err.Error(), "method", c.Method(), "path", c.Path())
	return c.Status(fiber.StatusInternalServerError).JSON(withRequestID(c, fiber.Map{
		"code":    string(xerrors.CodeInternalError),
		"message": "an unexpected error occurred",
	}))
}

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
		return handleDomainError(c, err, de)
	}

	// Special cases that aren't DomainErrors but have standard mappings.
	return handleSpecialErrors(c, err)
}

// handleDomainError processes a DomainError and sends to telemetry.
func handleDomainError(c *fiber.Ctx, err error, de *xerrors.DomainError) error {
	// Build telemetry request with context
	req := buildTelemetryRequest(c)

	// Add error metadata to telemetry request
	if len(de.Meta) > 0 {
		if req.Context == nil {
			req.Context = make(map[string]any)
		}
		for k, v := range de.Meta {
			req.Context["error_meta_"+k] = v
		}
	}

	// Add source location if available
	if de.Source != nil {
		if req.Context == nil {
			req.Context = make(map[string]any)
		}
		req.Context["error_source"] = map[string]any{
			"package":  de.Source.Package,
			"function": de.Source.Function,
			"file":     de.Source.File,
			"line":     de.Source.Line,
		}
	}

	// Select and apply policy
	policy := selectPolicy(de)
	_ = policy.Apply(c.Context(), req, de)

	// Build the response body
	resp := buildResponseBody(c, err)

	// Sanitize 500 messages per policy
	if de.Status == fiber.StatusInternalServerError {
		resp["message"] = "an unexpected error occurred"
	}

	return c.Status(statusForError(err)).JSON(resp)
}

// buildTelemetryRequest creates a telemetry request from Fiber context.
func buildTelemetryRequest(c *fiber.Ctx) *xerrors.TelemetryRequest {
	req := &xerrors.TelemetryRequest{
		Method:  c.Method(),
		Path:    c.Path(),
		Query:   make(map[string][]string),
		Context: make(map[string]any),
	}

	// Extract request ID via resolver
	if telemetry.RequestIDResolver != nil {
		if rid := telemetry.RequestIDResolver(c); rid != "" {
			req.RequestID = rid
		}
	}

	// Extract session context via resolver
	if telemetry.SessionResolver != nil {
		if sess := telemetry.SessionResolver(c); sess != nil {
			req.UserID = sess.UserID
			req.TenantID = sess.TenantID
			if req.Context == nil {
				req.Context = make(map[string]any)
			}
			req.Context["role"] = sess.Role
			req.Context["stytch_member_id"] = sess.StytchMemberID
		}
	}

	// Extract key headers
	req.Headers = make(map[string]string)
	for _, h := range []string{"user-agent", "referer", "x-forwarded-for"} {
		if v := c.Get(h); v != "" {
			req.Headers[h] = v
		}
	}

	return req
}

// selectPolicy returns the appropriate error policy based on the domain error.
func selectPolicy(de *xerrors.DomainError) *telemetry.ErrorPolicy {
	switch {
	case de.Status >= 500:
		return telemetry.InternalErrorPolicy()
	case de.Status == http.StatusUnauthorized || de.Status == http.StatusForbidden:
		return telemetry.AuthPolicy()
	case de.Status >= 400:
		return telemetry.ValidationPolicy()
	default:
		return telemetry.DefaultPolicy()
	}
}

// buildResponseBody constructs the JSON response body.
func buildResponseBody(c *fiber.Ctx, err error) fiber.Map {
	var code string
	var message string

	var de *xerrors.DomainError
	if errors.As(err, &de) {
		code = de.Code
		message = err.Error()

	} else {
		// Handle special cases
		switch {
		case errors.Is(err, context.Canceled):
			code = string(xerrors.CodeRequestCanceled)
			message = "the request was canceled"

		case errors.Is(err, context.DeadlineExceeded):
			code = string(xerrors.CodeTimeout)
			message = "the request timed out"

		default:
			// Fiber errors or unknown errors
			var fe *fiber.Error
			if errors.As(err, &fe) {
				code = fiberErrToCode(fe.Code)
				message = fe.Message

			} else {
				code = string(xerrors.CodeInternalError)
				message = "an unexpected error occurred"

			}
		}
	}

	resp := fiber.Map{
		"code":    code,
		"message": message,
	}

	// Check for field-level validation errors
	type hasFieldErrors interface{ FieldErrors() map[string][]string }
	if fe, ok := err.(hasFieldErrors); ok && len(fe.FieldErrors()) > 0 {
		resp["errors"] = fe.FieldErrors()
	} else if de, ok := err.(*xerrors.DomainError); ok && len(de.Fields) > 0 {
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

	// Add request ID to response
	if rid := GetRequestID(c); rid != "" {
		resp["request_id"] = rid
	}

	return resp
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

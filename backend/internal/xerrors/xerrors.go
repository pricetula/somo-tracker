// Package xerrors provides the structured error types used across all layers.
//
// DomainError is the universal error type that carries:
//   - machine-readable code  (e.g. "member_not_found")
//   - client-safe message    (human-readable)
//   - HTTP status            (mapped at the middleware layer)
//   - optional metadata      (field errors, conflicting resources, etc.)
//
// Every domain package defines its own sentinel errors as *DomainError
// instances. Packages do NOT import middleware or HTTP packages — the
// status code is embedded in the error itself and extracted by the
// centralized HTTPError function in the middleware.
package xerrors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// DomainError is a structured error that crosses layer boundaries.
// It implements error and Unwrap interfaces so that both
// errors.Is/errors.As and the HTTP middleware can inspect it.
//
// Fields:
//   - Code: Machine-readable snake_case error code (e.g. "member_not_found")
//   - Message: Human-readable message safe for client consumption
//   - Status: HTTP status code (not serialized to client)
//   - Fields: Optional field-level validation errors (serialized to "errors")
//   - Meta: Internal metadata for telemetry/observability (NOT serialized to client)
//   - Source: Source code location where error was created (NOT serialized to client)
type DomainError struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Status  int                 `json:"-"`
	Fields  map[string][]string `json:"errors,omitempty"`
	Meta    map[string]any      `json:"-"`
	Source  *ErrorSource        `json:"-"`
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func (e *DomainError) Unwrap() error { return nil }

// ErrorDetails returns any extra metadata for the response body.
// Implemented by custom error types that embed *DomainError.
func (e *DomainError) ErrorDetails() any { return nil }

// HasMeta is an interface for errors that carry internal telemetry metadata.
// Implemented by DomainError to expose Meta without type assertion.
type HasMeta interface {
	ErrorMeta() map[string]any
}

func (e *DomainError) ErrorMeta() map[string]any { return e.Meta }

// ErrorSource captures where an error was created for debugging.
// Not serialized to the client response.
type ErrorSource struct {
	Package  string
	Function string
	File     string
	Line     int
}

// WithMeta attaches internal metadata for telemetry/observability.
// This metadata is NOT sent to the client response but is available
// to telemetry sinks for debugging and monitoring.
//
// Usage:
//
//	err := xerrors.New("db_timeout", 504, "query timed out")
//	err = xerrors.WithMeta(err, map[string]any{
//		"query":     "SELECT * FROM users WHERE id = $1",
//		"duration":  5000,
//		"attempt":   3,
//	})
func WithMeta(de *DomainError, meta map[string]any) *DomainError {
	if de.Meta == nil {
		de.Meta = make(map[string]any)
	}
	for k, v := range meta {
		de.Meta[k] = v
	}
	return de
}

// WithSource captures the source code location.
// Intended to be called via xerrors.CaptureSource() at the error creation site.
func WithSource(de *DomainError, source *ErrorSource) *DomainError {
	de.Source = source
	return de
}

// CaptureSource returns an ErrorSource for the caller's location.
// Use this when creating errors to automatically capture file/function/line.
//
// Usage:
//
//	return xerrors.New("db_error", 500, "query failed").WithSource(xerrors.CaptureSource(1))
func CaptureSource(skip int) *ErrorSource {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return nil
	}
	fn := runtime.FuncForPC(pc)
	var pkg, function string
	if fn != nil {
		// Extract package and function name from full path
		fullName := fn.Name()
		// Format: github.com/user/repo/pkg.Func
		if idx := strings.LastIndex(fullName, "."); idx != -1 {
			pkg = fullName[:idx]
			function = fullName[idx+1:]
		} else {
			function = fullName
		}
	}
	return &ErrorSource{
		Package:  pkg,
		Function: function,
		File:     file,
		Line:     line,
	}
}

// ── Wrapping helpers ──────────────────────────────────────────────────────

// Wrap adds context to an error while preserving the error chain for
// errors.Is/errors.As. If err is nil, returns nil. The returned error
// unwraps to err, so sentinel checks and DomainError extraction continue
// to work.
//
// Usage:
//
//	return xerrors.Wrap(err, "users.Service.CreateUser")
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf is like Wrap but with formatted message.
//
// Usage:
//
//	return xerrors.Wrapf(err, "users.Service.CreateUser: email=%s", email)
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// WrapMeta wraps an error with additional metadata while preserving the chain.
// The metadata is merged with any existing Meta on the underlying DomainError.
//
// Usage:
//
//	return xerrors.WrapMeta(err, "users.Service.CreateUser", map[string]any{
//		"user_id": userID,
//		"email":   email,
//	})
func WrapMeta(err error, msg string, meta map[string]any) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("%s: %w", msg, err)

	// Try to update the underlying DomainError's Meta if present
	var de *DomainError
	if errors.As(err, &de) {
		// Merge metadata with existing
		merged := make(map[string]any)
		if de.Meta != nil {
			for k, v := range de.Meta {
				merged[k] = v
			}
		}
		for k, v := range meta {
			merged[k] = v
		}
		// Update the existing DomainError's Meta directly
		de.Meta = merged
	}

	return wrapped
}

// WrapMetaf is like WrapMeta but with formatted message.
func WrapMetaf(err error, meta map[string]any, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return WrapMeta(err, fmt.Sprintf(format, args...), meta)
}

// ── IsDomainError / AsDomainError ─────────────────────────────────────────

// Is is a convenience wrapper around errors.Is for *DomainError comparands.
func Is(err, target error) bool { return errors.Is(err, target) }

// As extracts the nearest *DomainError from the error chain.
func As(err error) (*DomainError, bool) {
	var de *DomainError
	ok := errors.As(err, &de)
	return de, ok
}

// GetMeta extracts the Meta map from the nearest DomainError in the chain.
// Returns nil if no DomainError with Meta is found.
func GetMeta(err error) map[string]any {
	var de *DomainError
	if errors.As(err, &de) && de.Meta != nil {
		return de.Meta
	}
	return nil
}

// ── Error Codes (constants for compile-time safety) ───────────────────────

// Code is a machine-readable error code. Use these constants instead of
// string literals to avoid typos and enable grepping.
type Code string

const (
	CodeNotFound                  Code = "not_found"
	CodeAlreadyExists             Code = "already_exists"
	CodeInvalidInput              Code = "invalid_input"
	CodeUnauthorized              Code = "unauthorized"
	CodeForbidden                 Code = "forbidden"
	CodeConflict                  Code = "conflict"
	CodeUnprocessableEntity       Code = "unprocessable_entity"
	CodeDeviceFingerprintMismatch Code = "device_fingerprint_mismatch"
	CodeRequestCanceled           Code = "request_canceled"
	CodeTimeout                   Code = "timeout"
	CodeInternalError             Code = "internal_error"
)

// ── Sentinels ─────────────────────────────────────────────────────────────

// These package-level sentinels are used by middleware (auth.go, etc.)
// that must return errors without a domain context. Domain packages should
// define their own sentinels using the constructors below.

var (
	ErrNotFound      = &DomainError{Code: string(CodeNotFound), Status: http.StatusNotFound, Message: "resource not found"}
	ErrAlreadyExists = &DomainError{Code: string(CodeAlreadyExists), Status: http.StatusConflict, Message: "resource already exists"}
	ErrInvalidInput  = &DomainError{Code: string(CodeInvalidInput), Status: http.StatusBadRequest, Message: "invalid input"}
	ErrUnauthorized  = &DomainError{Code: string(CodeUnauthorized), Status: http.StatusUnauthorized, Message: "authentication required"}
	ErrForbidden     = &DomainError{Code: string(CodeForbidden), Status: http.StatusForbidden, Message: "insufficient permissions"}
	ErrConflict      = &DomainError{Code: string(CodeConflict), Status: http.StatusConflict, Message: "resource conflict"}
)

// ErrDeviceFingerprintMismatch is returned when a request presents a
// device fingerprint different from the one recorded when the session was
// created. Mapped to 401 so the client re-authenticates — a stolen cookie
// cannot be replayed from a different device.
var ErrDeviceFingerprintMismatch = &DomainError{
	Code:    string(CodeDeviceFingerprintMismatch),
	Status:  http.StatusUnauthorized,
	Message: "session is bound to a different device; re-authenticate to continue",
}

// ── Sentinel Constructors ─────────────────────────────────────────────────

// New creates a DomainError with an explicit machine-readable code. Use this
// when a sentinel must surface a code distinct from the generic ones produced
// by the shorthand constructors (Unauthorized, Forbidden, ...) — e.g. auth's
// ErrExpiredToken must carry code "expired_token" on the wire so the frontend
// can distinguish it from "session_ref_expired" or "mfa_required".
func New(code string, status int, message string) *DomainError {
	return &DomainError{Code: code, Status: status, Message: message}
}

// NewWithMeta creates a DomainError with initial metadata for telemetry.
func NewWithMeta(code string, status int, message string, meta map[string]any) *DomainError {
	de := &DomainError{Code: code, Status: status, Message: message}
	if meta != nil {
		de.Meta = meta
	}
	return de
}

func NotFound(msg string) *DomainError {
	return &DomainError{Code: string(CodeNotFound), Status: http.StatusNotFound, Message: msg}
}

func AlreadyExists(msg string) *DomainError {
	return &DomainError{Code: string(CodeAlreadyExists), Status: http.StatusConflict, Message: msg}
}

func InvalidInput(msg string) *DomainError {
	return &DomainError{Code: string(CodeInvalidInput), Status: http.StatusBadRequest, Message: msg}
}

func Unauthorized(msg string) *DomainError {
	return &DomainError{Code: string(CodeUnauthorized), Status: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) *DomainError {
	return &DomainError{Code: string(CodeForbidden), Status: http.StatusForbidden, Message: msg}
}

func Conflict(msg string) *DomainError {
	return &DomainError{Code: string(CodeConflict), Status: http.StatusConflict, Message: msg}
}

func UnprocessableEntity(msg string) *DomainError {
	return &DomainError{Code: string(CodeUnprocessableEntity), Status: http.StatusUnprocessableEntity, Message: msg}
}

func RequestTimeout() *DomainError {
	return &DomainError{Code: string(CodeTimeout), Status: http.StatusGatewayTimeout, Message: "the request timed out"}
}

func RequestCanceled() *DomainError {
	return &DomainError{Code: string(CodeRequestCanceled), Status: 499, Message: "the request was canceled"}
}

// WithFields attaches field-level validation metadata.
func WithFields(de *DomainError, fields map[string][]string) *DomainError {
	de.Fields = fields
	return de
}

// ── ErrorDetails interface ────────────────────────────────────────────────

// HasDetails is the interface that custom domain error types can implement
// to attach extra metadata (conflicting resources, dependent IDs, etc.) to
// the response body. The return value is serialized into the "errors" field
// of the canonical JSON response.
type HasDetails interface {
	ErrorDetails() any
}

// ── Telemetry Interface ────────────────────────────────────────────────────

// TelemetrySink defines the interface for error telemetry backends.
// Implementation examples: Sentry, Datadog, New Relic, Prometheus, custom.
type TelemetrySink interface {
	// ProcessError receives error details for telemetry.
	// Must be safe to call from goroutines.
	ProcessError(ctx context.Context, err *DomainError, req *TelemetryRequest)
	// Flush ensures all buffered data is sent.
	// Called during server shutdown.
	Flush(ctx context.Context) error
	// Name returns the sink's identifier for logging/debugging.
	Name() string
}

// TelemetryRequest contains contextual information about the error occurrence.
type TelemetryRequest struct {
	Method    string
	Path      string
	Query     map[string][]string
	Headers   map[string]string
	UserID    string
	TenantID  string
	RequestID string
	// Context is additional structured data from the request
	Context map[string]any
}

// ── pgx Helpers ───────────────────────────────────────────────────────────

// MapPgxError maps common pgx errors to domain sentinels.
// Use this inside repository methods to avoid string-matching.
func MapPgxError(cause error, fmtMsg string) error {
	if cause == nil {
		return nil
	}
	// pgx.ErrNoRows is the standard "not found" from a QueryRow scan.
	if errors.Is(cause, errNoRows()) {
		return fmt.Errorf("%s: %w", fmtMsg, ErrNotFound)
	}
	return fmt.Errorf("%s: %w", fmtMsg, cause)
}

// errNoRows is a lazily-resolved reference to pgx.ErrNoRows that avoids
// a hard import of the pgx package in this shared package.
var errNoRows = func() error { return errors.New("no rows in result set") }

// SetErrNoRowsResolver allows the database package to inject the real
// pgx.ErrNoRows sentinel at init time, so this package doesn't import pgx.
func SetErrNoRowsResolver(fn func() error) {
	errNoRows = fn
}

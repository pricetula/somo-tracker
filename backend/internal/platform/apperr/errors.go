package apperr

import (
	"fmt"
	"sync"
	"time"
)

// Code is stable SCREAMING_SNAKE error code (BE-025).
type Code string

// FieldError represents a single field-level validation error.
type FieldError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error is typed error with code, message, field errors, retry-after, and cause.
type Error struct {
	Code       Code
	Message    string                  // client-safe; ignored for 5xx
	Fields     map[string][]FieldError // optional
	RetryAfter time.Duration           // optional, for 429/503
	Cause      error                   // logged, never sent to the client
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

var (
	codeToStatus = make(map[Code]int)
	statusToCode = make(map[int]Code)
	mu           sync.RWMutex
)

// Register registers an error code with its HTTP status.
// Must be called during initialization (e.g., in init()).
// Panics if code is already registered with a different status.
func Register(code Code, status int) {
	mu.Lock()
	defer mu.Unlock()

	if existing, ok := codeToStatus[code]; ok {
		if existing != status {
			panic(fmt.Sprintf("apperr: code %q already registered with status %d", code, existing))
		}
		return
	}
	codeToStatus[code] = status
	statusToCode[status] = code
}

// Status returns the HTTP status for a code.
// Returns 0 if code is not registered (programming error).
func Status(code Code) int {
	mu.RLock()
	defer mu.RUnlock()
	return codeToStatus[code]
}

// CodeFromStatus returns a code for a status, or "INTERNAL_ERROR" if unknown.
func CodeFromStatus(status int) Code {
	mu.RLock()
	defer mu.RUnlock()
	if c, ok := statusToCode[status]; ok {
		return c
	}
	if status >= 500 {
		return INTERNAL_ERROR
	}
	return CLIENT_ERROR
}

// New creates a new Error with the given code and message.
// The code must be registered (will panic in tests if not).
func New(code Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

// Wrap wraps an existing error with a code and message.
// The cause is preserved for logging but never sent to the client.
func Wrap(code Code, msg string, cause error) *Error {
	return &Error{Code: code, Message: msg, Cause: cause}
}

// Validation creates a VALIDATION_FAILED error with field errors.
func Validation(fields map[string][]FieldError) *Error {
	return &Error{
		Code:    VALIDATION_FAILED,
		Message: "Request validation failed",
		Fields:  fields,
	}
}

// Generic error codes (section 5.1 of contract)
const (
	BAD_REQUEST         Code = "BAD_REQUEST"
	UNAUTHENTICATED     Code = "UNAUTHENTICATED"
	FORBIDDEN           Code = "FORBIDDEN"
	CSRF_FAILED         Code = "CSRF_FAILED"
	NOT_FOUND           Code = "NOT_FOUND"
	METHOD_NOT_ALLOWED  Code = "METHOD_NOT_ALLOWED"
	REQUEST_TIMEOUT     Code = "REQUEST_TIMEOUT"
	CONFLICT            Code = "CONFLICT"
	PAYLOAD_TOO_LARGE   Code = "PAYLOAD_TOO_LARGE"
	UNSUPPORTED_MEDIA   Code = "UNSUPPORTED_MEDIA_TYPE"
	VALIDATION_FAILED   Code = "VALIDATION_FAILED"
	RATE_LIMITED        Code = "RATE_LIMITED"
	INTERNAL_ERROR      Code = "INTERNAL_ERROR"
	BAD_GATEWAY         Code = "BAD_GATEWAY"
	SERVICE_UNAVAILABLE Code = "SERVICE_UNAVAILABLE"
	GATEWAY_TIMEOUT     Code = "GATEWAY_TIMEOUT"
	CLIENT_ERROR        Code = "CLIENT_ERROR"
)

// Domain codes (section 5.2) - register in domain packages via init()
const (
	EMAIL_TAKEN          Code = "EMAIL_TAKEN"
	INVITE_EXPIRED       Code = "INVITE_EXPIRED"
	INVALID_TOKEN        Code = "INVALID_TOKEN"
	MISSING_PKCE         Code = "MISSING_PKCE"
	INVALID_TICKET       Code = "INVALID_TICKET"
	IDEMPOTENCY_CONFLICT Code = "IDEMPOTENCY_CONFLICT"
	PROVISIONING_FAILED  Code = "PROVISIONING_FAILED"
	SESSION_CREATE_FAILED Code = "SESSION_CREATE_FAILED"
	NO_ACTIVE_SESSION    Code = "NO_ACTIVE_SESSION"
	INVALID_REQUEST      Code = "INVALID_REQUEST"
	INVALID_EMAIL        Code = "INVALID_EMAIL"
)

func init() {
	// Domain codes first (so forward map gets them)
	Register(EMAIL_TAKEN, 409)
	Register(INVITE_EXPIRED, 401)
	Register(INVALID_TOKEN, 401)
	Register(MISSING_PKCE, 400)
	Register(INVALID_TICKET, 400)
	Register(IDEMPOTENCY_CONFLICT, 409)
	Register(PROVISIONING_FAILED, 500)
	Register(SESSION_CREATE_FAILED, 500)
	Register(NO_ACTIVE_SESSION, 401)
	Register(INVALID_REQUEST, 400)
	Register(INVALID_EMAIL, 422)

	// Generic codes last (so they win in reverse map as canonical)
	Register(BAD_REQUEST, 400)
	Register(UNAUTHENTICATED, 401)
	Register(CSRF_FAILED, 403)
	Register(FORBIDDEN, 403)
	Register(NOT_FOUND, 404)
	Register(METHOD_NOT_ALLOWED, 405)
	Register(REQUEST_TIMEOUT, 408)
	Register(CONFLICT, 409)
	Register(PAYLOAD_TOO_LARGE, 413)
	Register(UNSUPPORTED_MEDIA, 415)
	Register(VALIDATION_FAILED, 422)
	Register(RATE_LIMITED, 429)
	Register(INTERNAL_ERROR, 500)
	Register(BAD_GATEWAY, 502)
	Register(SERVICE_UNAVAILABLE, 503)
	Register(GATEWAY_TIMEOUT, 504)
}
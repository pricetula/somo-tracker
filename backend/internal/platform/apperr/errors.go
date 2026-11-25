package apperr

import (
	"errors"
	"fmt"
)

// Code is a stable error code returned to clients.
type Code string

const (
	INVALID_REQUEST   Code = "INVALID_REQUEST"
	INVALID_EMAIL     Code = "INVALID_EMAIL"
	INVALID_TOKEN     Code = "INVALID_TOKEN"
	INTERNAL_ERROR    Code = "INTERNAL_ERROR"
	BAD_GATEWAY       Code = "BAD_GATEWAY"
	RATE_LIMITED      Code = "RATE_LIMITED"
	UNAUTHENTICATED   Code = "UNAUTHENTICATED"
	FORBIDDEN         Code = "FORBIDDEN"
	NOT_FOUND         Code = "NOT_FOUND"
	CONFLICT          Code = "CONFLICT"
	VALIDATION_FAILED Code = "VALIDATION_FAILED"
)

// Error is a typed application error with a stable code.
type Error struct {
	Code    Code
	Message string
	Err     error // wrapped cause
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// New creates a new application error.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap wraps an existing error with a code and message.
func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// IsCode checks if an error is an *Error with the given code.
func IsCode(err error, code Code) bool {
	var appErr *Error
	if err == nil {
		return false
	}
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

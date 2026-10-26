package apperr

import "fmt"

// Code is stable SCREAMING_SNAKE error code (BE-025).
type Code string

const (
	BAD_REQUEST        Code = "BAD_REQUEST"
	UNAUTHENTICATED    Code = "UNAUTHENTICATED"
	FORBIDDEN          Code = "FORBIDDEN"
	NOT_FOUND          Code = "NOT_FOUND"
	CONFLICT           Code = "CONFLICT"
	VALIDATION_FAILED  Code = "VALIDATION_FAILED"
	INTERNAL           Code = "INTERNAL"
)

// Error is typed error with code and message.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Err)
	}
	return string(e.Code)
}

func New(code Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

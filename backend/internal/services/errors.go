package services

import (
	"fmt"
)

// ErrForbidden is a service-level error for authorization failures.
// It wraps a message and can be checked by callers.
type ErrForbidden struct {
	Code    string
	Message string
}

func (e ErrForbidden) Error() string {
	return e.Message
}

// NewErrForbidden creates a new ErrForbidden.
func NewErrForbidden(code, message string) ErrForbidden {
	return ErrForbidden{Code: code, Message: message}
}

// IsErrForbidden checks if an error is an ErrForbidden.
func IsErrForbidden(err error) bool {
	_, ok := err.(ErrForbidden)
	return ok
}

// ErrInternal is a service-level error for internal failures.
type ErrInternal struct {
	Message string
}

func (e ErrInternal) Error() string {
	return e.Message
}

func NewErrInternal(message string) ErrInternal {
	return ErrInternal{Message: message}
}

// ErrNotFound is a service-level error for not found.
type ErrNotFound struct {
	Message string
}

func (e ErrNotFound) Error() string {
	return e.Message
}

func NewErrNotFound(message string) ErrNotFound {
	return ErrNotFound{Message: message}
}

// ErrBadRequest is a service-level error for bad requests.
type ErrBadRequest struct {
	Message string
}

func (e ErrBadRequest) Error() string {
	return e.Message
}

func NewErrBadRequest(message string) ErrBadRequest {
	return ErrBadRequest{Message: message}
}

// ServiceError is a generic service error with code.
type ServiceError struct {
	Code    string
	Message string
	Err     error
}

func (e ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e ServiceError) Unwrap() error {
	return e.Err
}

func NewServiceError(code, message string, err error) ServiceError {
	return ServiceError{Code: code, Message: message, Err: err}
}

// BadRequest returns a ServiceError with code "bad_request".
func BadRequest(msg string) error {
	return &ServiceError{Code: "bad_request", Message: msg}
}

// Forbidden returns a ServiceError with code "forbidden".
func Forbidden(msg string) error {
	return &ServiceError{Code: "forbidden", Message: msg}
}

// Internal returns a ServiceError with code "internal_error".
func Internal(msg string) error {
	return &ServiceError{Code: "internal_error", Message: msg}
}

package services

import "fmt"

type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func BadRequest(msg string) error {
	return &ServiceError{Code: "bad_request", Message: msg}
}

func Forbidden(msg string) error {
	return &ServiceError{Code: "forbidden", Message: msg}
}

func Internal(msg string) error {
	return &ServiceError{Code: "internal_error", Message: msg}
}

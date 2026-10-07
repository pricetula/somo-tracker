package api

import (
	"github.com/gofiber/fiber/v3"
	"somotracker/backend/internal/api/middleware"
)

// APIError is the canonical error response shape used across the backend.
type APIError struct {
	Code    string
	Message string
	Errors  map[string][]string
	Status  int
}

// WriteError writes a canonical JSON error response with request_id correlation.
// It sets X-Request-ID header, uses the request-scoped logger for the request, and
// returns a Fiber error so callers can continue the chain.
func WriteError(c fiber.Ctx, e APIError) error {
	reqID := middleware.GetRequestID(c)
	if reqID == "" {
		reqID = c.Get("X-Request-ID")
	}
	// Ensure header is echoed
	c.Set("X-Request-ID", reqID)

	// Keep errors map non-nil
	if e.Errors == nil {
		e.Errors = map[string][]string{}
	}

	return c.Status(e.Status).JSON(fiber.Map{
		"code":       e.Code,
		"message":    e.Message,
		"errors":     e.Errors,
		"request_id": reqID,
	})
}

// Common error constructors

func ErrBadRequest(message string, errors map[string][]string) APIError {
	return APIError{
		Status:  fiber.StatusBadRequest,
		Code:    "bad_request",
		Message: message,
		Errors:  errors,
	}
}

func ErrNotFound(message string) APIError {
	return APIError{
		Status:  fiber.StatusNotFound,
		Code:    "not_found",
		Message: message,
		Errors:  map[string][]string{},
	}
}

func ErrUnauthorized(message string) APIError {
	return APIError{
		Status:  fiber.StatusUnauthorized,
		Code:    "unauthorized",
		Message: message,
		Errors:  map[string][]string{},
	}
}

func ErrForbidden(code, message string) APIError {
	return APIError{
		Status:  fiber.StatusForbidden,
		Code:    code,
		Message: message,
		Errors:  map[string][]string{},
	}
}

func ErrInternal(message string) APIError {
	return APIError{
		Status:  fiber.StatusInternalServerError,
		Code:    "internal_error",
		Message: message,
		Errors:  map[string][]string{},
	}
}

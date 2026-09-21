package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"

	"somotracker/backend/internal/api/middleware"
)

func NewErrorHandler() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, code := codeAndStatus(err)
		reqLogger := middleware.GetLogger(c)
		reqID := middleware.GetRequestID(c)

		reqLogger.Error("request error",
			zap.Error(err),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.String("request_id", reqID),
		)

		// Sanitize internal errors for clients
		message := "Something went wrong"
		if status != fiber.StatusInternalServerError {
			message = err.Error()
		}

		return WriteError(c, APIError{
			Status:  status,
			Code:    code,
			Message: message,
			Errors:  map[string][]string{},
		})
	}
}

func codeAndStatus(err error) (int, string) {
	switch {
	case errors.Is(err, fiber.ErrNotFound):
		return fiber.StatusNotFound, "not_found"
	case errors.Is(err, fiber.ErrInternalServerError):
		return fiber.StatusInternalServerError, "internal_error"
	default:
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return fe.Code, stableCode(fe.Code)
		}
		return fiber.StatusInternalServerError, "internal_error"
	}
}

func stableCode(httpStatus int) string {
	switch httpStatus {
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	default:
		return "internal_error"
	}
}

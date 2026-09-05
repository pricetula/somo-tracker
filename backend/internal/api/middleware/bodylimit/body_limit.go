// Package bodylimit provides a middleware that restricts incoming JSON
// payload sizes globally to 4MB.
package bodylimit

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

const maxPayloadSize = 4 * 1024 * 1024 // 4MB

// Middleware returns a Fiber handler that enforces a 4MB body size limit.
// It checks the declared Content-Length header before passing to the next
// handler, and also intercepts fiber.ErrRequestEntityTooLarge raised by
// downstream body parsing to return the canonical error response.
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Fast-path: reject requests with an explicitly declared oversized body.
		if cl := c.Request().Header.ContentLength(); cl > maxPayloadSize {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"code":    "payload_too_large",
				"message": "Request body exceeds the maximum allowed size",
				"errors":  fiber.Map{},
			})
		}

		err := c.Next()

		// Intercept Fiber's native body-too-large error and return our contract.
		if err != nil && errors.Is(err, fiber.ErrRequestEntityTooLarge) {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"code":    "payload_too_large",
				"message": "Request body exceeds the maximum allowed size",
				"errors":  fiber.Map{},
			})
		}

		return err
	}
}

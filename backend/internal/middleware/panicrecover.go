package middleware

import (
	"github.com/gofiber/fiber/v2"
	fibermiddleware "github.com/gofiber/fiber/v2/middleware/recover"
)

// NewRecover handles panic recovery.
func newPanicRecover() fiber.Handler {
	return fibermiddleware.New()
}

// Package timeout provides a request-timeout middleware that caps handler
// execution at 15 seconds and returns a 504 Gateway Timeout with our canonical
// error format when the deadline is exceeded.
package timeout

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/timeout"
	"go.uber.org/zap"

	"somotracker/backend/internal/api/middleware"
)

const maxDuration = 15 * time.Second

// Middleware returns a Fiber handler that enforces a 15-second execution cap.
// It wraps Fiber's built-in timeout middleware (with Abandon/reclaim logic)
// and uses OnTimeout to return a 504 with our canonical error format.
func Middleware() fiber.Handler {
	onTimeout := func(c fiber.Ctx) error {
		logger := middleware.GetLogger(c)
		logger.Warn("timeout middleware: request cancelled",
			zap.String("request_id", middleware.GetRequestID(c)),
			zap.Duration("timeout", maxDuration),
		)

		return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
			"code":    "gateway_timeout",
			"message": "Gateway timeout",
			"errors":  fiber.Map{},
		})
	}

	return timeout.New(func(c fiber.Ctx) error {
		return c.Next()
	}, timeout.Config{
		Timeout:   maxDuration,
		OnTimeout: onTimeout,
	})
}

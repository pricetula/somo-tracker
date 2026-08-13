package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// loggerKey is the fiber Locals key under which the application logger is
// stored by WithLogger.
const loggerKey = "middleware.logger"

// WithLogger stores the *zap.SugaredLogger on the request context so that
// package-level middleware helpers (HTTPError, access log, panic recovery,
// rate limiter) can log without threading a logger through every handler.
//
// It MUST be registered first in Register so the logger is present before any
// other middleware — and, critically, before the global error handler runs,
// since HTTPError reads it from c.Locals while serializing 500s.
func WithLogger(logger *zap.SugaredLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(loggerKey, logger)
		return c.Next()
	}
}

// loggerFrom returns the request logger stored by WithLogger. It falls back to
// a no-op logger so helpers stay safe when invoked outside a registered Fiber
// app (e.g. in unit tests that call HTTPError directly).
func loggerFrom(c *fiber.Ctx) *zap.SugaredLogger {
	if logger, ok := c.Locals(loggerKey).(*zap.SugaredLogger); ok && logger != nil {
		return logger
	}
	return zap.NewNop().Sugar()
}

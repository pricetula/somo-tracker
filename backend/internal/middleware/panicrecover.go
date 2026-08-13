package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	recovermw "github.com/gofiber/fiber/v2/middleware/recover"
)

// NewPanicRecover recovers panics into errors so the global error handler
// (HTTPError) can return the canonical JSON error body instead of an empty
// 500. Stack traces are logged via zap with request context so panics are
// actually debuggable in production.
func NewPanicRecover() fiber.Handler {
	return recovermw.New(recovermw.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, r any) {
			fields := []interface{}{
				"method", c.Method(),
				"path", c.Path(),
				"panic", fmt.Sprintf("%v", r),
				"stack", string(debug.Stack()),
			}
			if rid := GetRequestID(c); rid != "" {
				fields = append(fields, "request_id", rid)
			}
			loggerFrom(c).Errorw("panic recovered", fields...)
		},
	})
}

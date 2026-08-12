package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// NewAccessLog logs one structured line per API request after the handler
// chain completes: method, path, status, duration, request_id, and — when a
// session was resolved — user_id, tenant_id and role.
//
// It MUST be registered after NewSessionResolver so the session is available;
// consequently, requests rejected earlier in the chain (CORS, CSRF, coarse
// rate limit) are not access-logged. Error-level detail for 5xx responses is
// logged once by HTTPError, so this middleware never duplicates it.
func NewAccessLog() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			// The global error handler serializes the response after the
			// middleware chain unwinds, so the raw status is not set yet.
			status = statusForError(err)
		}

		fields := []interface{}{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration", time.Since(start),
		}
		if rid := GetRequestID(c); rid != "" {
			fields = append(fields, "request_id", rid)
		}
		if sess := GetSession(c); sess != nil {
			fields = append(fields,
				"user_id", sess.UserID,
				"tenant_id", sess.TenantID,
				"role", sess.Role,
			)
		}
		loggerFrom(c).Infow("http request", fields...)

		return err
	}
}

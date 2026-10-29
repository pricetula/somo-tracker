package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/go-playground/validator/v10"
	"somotracker/internal/identity/handler"
	"somotracker/internal/platform/apperr"
)

// Context keys for request-scoped values
type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
)

// requestIDPattern validates inbound X-Request-ID headers.
// Allows alphanumeric, hyphen, underscore; 8-64 chars.
var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// newRequestID generates a request ID: "req_" + 22 chars (16 bytes, base64 RawURL = 128 bits).
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "req_unknown"
	}
	return "req_" + base64.RawURLEncoding.EncodeToString(b)
}

// RequestID middleware generates or validates the request ID.
// Must be the FIRST middleware so router errors and panics carry an ID.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if !requestIDPattern.MatchString(id) {
			id = newRequestID()
		}

		logger := slog.Default().With("request_id", id)
		ctx := context.WithValue(c.Context(), requestIDKey, id)
		ctx = context.WithValue(ctx, loggerKey, logger)
		c.SetContext(ctx)
		c.Locals("request_id", id)
		c.Set("X-Request-ID", id)

		return c.Next()
	}
}

// AccessLog middleware emits exactly one completion line per request.
// Logs: method, route (pattern), status, duration_ms, bytes.
func AccessLog() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// Skip health/ready endpoints to reduce noise
		path := c.Path()
		if path == "/healthz" || path == "/readyz" {
			return err
		}

		duration := time.Since(start).Milliseconds()
		status := c.Response().StatusCode()
		bytes := c.Response().Header.ContentLength()

		attrs := []any{
			"method", c.Method(),
			"route", c.Route().Path,
			"status", status,
			"duration_ms", duration,
			"bytes", bytes,
		}

		// If there's an error, call the error handler to ensure it's logged
		if err != nil {
			if eh := c.App().Config().ErrorHandler; eh != nil {
				_ = eh(c, err)
			}
		}

		// Log at Info level (errors logged by ErrorHandler at Error/Warn)
		logger := LoggerFrom(c.Context())
		logger.Info("request completed", attrs...)

		return err
	}
}

// RequestIDFrom extracts the request ID from context.
// Returns "req_unknown" if not set.
func RequestIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return "req_unknown"
}

// LoggerFrom extracts the request-scoped logger from context.
// Falls back to slog.Default() if not set.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ErrorHandler is the central error handler implementing the error contract.
// Resolves errors via errors.As in order: apperr.Error, validator.ValidationErrors,
// json.UnmarshalTypeError, json.SyntaxError, fiber.Error, then 500.
func ErrorHandler(c fiber.Ctx, err error) error {
	ctx := c.Context()
	status := fiber.StatusInternalServerError
	resp := ErrorResponse{
		Code:      string(apperr.INTERNAL_ERROR),
		Message:   "Internal Server Error",
		Errors:    map[string][]apperr.FieldError{},
		RequestID: RequestIDFrom(ctx),
	}

	var (
		ae *apperr.Error
		ve validator.ValidationErrors
		te *json.UnmarshalTypeError
		se *json.SyntaxError
		fe *fiber.Error
	)

	// Use errors.As for resolution order
	// 1. apperr.Error
	errors.As(err, &ae)

	// 2. validator.ValidationErrors
	errors.As(err, &ve)

	// 3. json.UnmarshalTypeError
	errors.As(err, &te)

	// 4. json.SyntaxError
	errors.As(err, &se)

	// 5. fiber.Error
	errors.As(err, &fe)

	switch {
	case ae != nil:
		status = apperr.Status(ae.Code)
		resp.Code = string(ae.Code)
		resp.Message = ae.Message
		if ae.Fields != nil {
			resp.Errors = ae.Fields
		}
		if ae.RetryAfter > 0 {
			c.Set("Retry-After", strconv.Itoa(int(ae.RetryAfter.Seconds())))
		}

	case ve != nil:
		status = fiber.StatusUnprocessableEntity
		resp.Code = string(apperr.VALIDATION_FAILED)
		resp.Message = "Request validation failed"
		resp.Errors = formatValidationErrors(ve)

	case te != nil:
		status = fiber.StatusUnprocessableEntity
		resp.Code = string(apperr.VALIDATION_FAILED)
		resp.Message = "Request validation failed"
		resp.Errors = map[string][]apperr.FieldError{
			te.Field: {{Code: "invalid_type", Message: "has the wrong type"}},
		}

	case se != nil:
		status = fiber.StatusBadRequest
		resp.Code = string(apperr.BAD_REQUEST)
		resp.Message = "Malformed JSON"

	case fe != nil:
		status = fe.Code
		resp.Code = string(apperr.CodeFromStatus(fe.Code))
		resp.Message = fe.Message
	}

	if status >= 500 {
		resp.Message = defaultMessage(resp.Code)
		resp.Errors = map[string][]apperr.FieldError{}
	}

	lvl := slog.LevelInfo
	attrs := []any{"code", resp.Code, "status", status, "route", c.Route().Path}
	switch {
	case status >= 500:
		lvl = slog.LevelError
		attrs = append(attrs, "err", err)
	case status == 429:
		lvl = slog.LevelWarn
	}
	LoggerFrom(ctx).Log(ctx, lvl, "request error", attrs...)

	c.Set("Cache-Control", "no-store")
	c.Set("X-Request-ID", RequestIDFrom(ctx))
	return c.Status(status).JSON(resp)
}

// ErrorResponse matches the contract shape (section 4).
type ErrorResponse struct {
	Code      string                         `json:"code"`
	Message   string                         `json:"message"`
	Errors    map[string][]apperr.FieldError `json:"errors"`
	RequestID string                         `json:"request_id"`
}

// defaultMessage returns the generic message for 5xx codes.
func defaultMessage(code string) string {
	switch apperr.Code(code) {
	case apperr.INTERNAL_ERROR:
		return "Internal Server Error"
	case apperr.BAD_GATEWAY:
		return "Bad Gateway"
	case apperr.SERVICE_UNAVAILABLE:
		return "Service Unavailable"
	case apperr.GATEWAY_TIMEOUT:
		return "Gateway Timeout"
	default:
		return "Internal Server Error"
	}
}

// formatValidationErrors converts validator errors to the contract shape.
// Uses JSON field names via TagNameFunc (keys like "email", "items[0].name").
func formatValidationErrors(ve validator.ValidationErrors) map[string][]apperr.FieldError {
	errs := make(map[string][]apperr.FieldError)
	for _, fe := range ve {
		key := fe.Field()
		// Strip root struct name if present (e.g., "Request.Email" -> "email")
		if idx := strings.Index(key, "."); idx != -1 {
			key = key[idx+1:]
		}
		// Convert Go field name to JSON name if needed (camelCase)
		key = toJSONKey(key)

		code := fieldCodeFromTag(fe.Tag(), fe.Param())
		msg := fieldMessageFromCode(code, fe.Param())

		errs[key] = append(errs[key], apperr.FieldError{Code: code, Message: msg})
	}
	return errs
}

// toJSONKey converts Go field name to JSON key (camelCase).
func toJSONKey(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// fieldCodeFromTag maps validator tag to field error code (section 5.4).
func fieldCodeFromTag(tag, param string) string {
	switch tag {
	case "required":
		return "required"
	case "email", "url", "uuid", "datetime", "e164":
		return "invalid_format"
	case "min":
		return "too_short"
	case "max":
		return "too_long"
	case "gte", "lte", "gt", "lt":
		return "out_of_range"
	case "oneof":
		return "invalid_value"
	default:
		return "invalid"
	}
}

// fieldMessageFromCode returns default English message for field code.
func fieldMessageFromCode(code, param string) string {
	switch code {
	case "required":
		return "is required"
	case "invalid_format":
		return "has an invalid format"
	case "too_short":
		return "is too short (min " + param + ")"
	case "too_long":
		return "is too long (max " + param + ")"
	case "out_of_range":
		return "is out of range"
	case "too_few":
		return "has too few items (min " + param + ")"
	case "too_many":
		return "has too many items (max " + param + ")"
	case "invalid_value":
		return "has an invalid value (must be one of: " + param + ")"
	case "invalid_type":
		return "has the wrong type"
	default:
		return "is invalid"
	}
}

// Server wraps Fiber app configured per BE-022.
type Server struct {
	app *fiber.App
}

// New creates Fiber app with /api/v1 route group and middleware chain.
func New() *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Somotracker",
		ServerHeader: "Somotracker",
		ErrorHandler: ErrorHandler,
	})

	// Middleware order (section 7.1):
	// 1. RequestID - first, always
	// 2. AccessLog - one completion line per request
	// 3. Recover - panics become errors handled by ErrorHandler
	// 4. CORS, security headers, body limit (added by app composition)
	// 5. Session/auth, tenant resolution (added by app composition)
	// 6. Routes
	app.Use(RequestID())
	app.Use(AccessLog())
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, e interface{}) {
			// Log stack trace at Error level
			logger := LoggerFrom(c.Context())
			logger.Error("panic recovered",
				"err", e,
				"stack", "VERIFY: fiber v3 recover stack trace API",
			)
		},
	}))

	v1 := app.Group("/api/v1")
	_ = v1

	return &Server{app: app}
}

// App returns the underlying Fiber app for middleware configuration.
func (s *Server) App() *fiber.App {
	return s.app
}

// RegisterRoutes registers domain handlers (thin, BE-006).
func (s *Server) RegisterRoutes() {
	// TODO: wire services with real dependencies
	// For now, handler is scaffolded; real wiring happens in internal/app
	_ = handler.New
}

// Start listens on addr.
func (s *Server) Start(ctx context.Context, addr string) error {
	go func() {
		<-ctx.Done()
		slog.Info("shutting down HTTP server")
		_ = s.app.Shutdown()
	}()
	slog.Info("HTTP server listening", "addr", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

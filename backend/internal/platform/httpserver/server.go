package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"somotracker/internal/platform/apperr"
)

// Server wraps a Fiber app.
type Server struct {
	app *fiber.App
}

// New creates a Fiber server with default middleware.
func New(env string) *Server {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := http.StatusInternalServerError
			appCode := apperr.INTERNAL_ERROR
			message := "internal error"

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			} else if appErr, ok := err.(*apperr.Error); ok {
				switch appErr.Code {
				case apperr.INVALID_REQUEST, apperr.INVALID_EMAIL:
					code = http.StatusBadRequest
				case apperr.RATE_LIMITED:
					code = http.StatusTooManyRequests
				case apperr.UNAUTHENTICATED:
					code = http.StatusUnauthorized
				case apperr.FORBIDDEN:
					code = http.StatusForbidden
				case apperr.NOT_FOUND:
					code = http.StatusNotFound
				case apperr.CONFLICT:
					code = http.StatusConflict
				case apperr.VALIDATION_FAILED:
					code = http.StatusUnprocessableEntity
				case apperr.INVALID_TOKEN:
					code = http.StatusUnauthorized
				case apperr.BAD_GATEWAY:
					code = http.StatusBadGateway
				default:
					code = http.StatusInternalServerError
				}
				appCode = appErr.Code
				message = appErr.Message
			}

			slog.Error("request failed", "err", err, "path", c.Path(), "status", code)

			return c.Status(code).JSON(fiber.Map{
				"code":    appCode,
				"message": message,
			})
		},
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     ParseOrigins(os.Getenv("CORS_ORIGINS")),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Cookie"},
		AllowCredentials: os.Getenv("CORS_ORIGINS") != "" && os.Getenv("CORS_ORIGINS") != "*",
	}))

	return &Server{app: app}
}

// App returns the underlying Fiber app.
func (s *Server) App() *fiber.App {
	return s.app
}

// Start begins listening on the given port.
func (s *Server) Start(ctx context.Context, port string) error {
	return s.app.Listen(":" + port)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.Shutdown()
}

// RequireAuth is a placeholder for authentication middleware.
func RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Next()
	}
}

// TenantMiddleware is a placeholder for tenant/school context middleware.
func TenantMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Next()
	}
}

// CSRFMiddleware is a placeholder for CSRF protection.
func CSRFMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Next()
	}
}

// Public marks a route as public (no auth required).
func Public() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Next()
	}
}

// Helper to build CORS origins from comma-separated string.
func ParseOrigins(origins string) []string {
	if origins == "" {
		return []string{"*"}
	}
	parts := strings.Split(origins, ",")
	for i, o := range parts {
		parts[i] = strings.TrimSpace(o)
	}
	return parts
}

// DefaultTimeout for HTTP operations.
const DefaultTimeout = 10 * time.Second

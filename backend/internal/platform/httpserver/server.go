package httpserver

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

// Server wraps Fiber app configured per BE-022.
type Server struct {
	app *fiber.App
}

// New creates Fiber app with /api/v1 route group.
func New() *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Somotracker",
		ServerHeader: "Somotracker",
		ErrorHandler: customErrorHandler,
	})
	v1 := app.Group("/api/v1")
	_ = v1 // will be used by RegisterRoutes

	return &Server{app: app}
}

// App returns the underlying Fiber app for middleware configuration.
func (s *Server) App() *fiber.App {
	return s.app
}

// RegisterRoutes registers domain handlers (thin, BE-006).
func (s *Server) RegisterRoutes() {
	// TODO: register domain routes
	// Example:
	// identityHandler := handler.New(identityService)
	// identityHandler.RegisterRoutes(s.app.Group("/api/v1"))
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

// customErrorHandler implements BE-025 error contract.
func customErrorHandler(c fiber.Ctx, err error) error {
	// This will be replaced by apperr handler once implemented
	code := fiber.StatusInternalServerError
	msg := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		msg = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": msg,
	})
}
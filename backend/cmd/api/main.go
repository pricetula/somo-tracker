package main

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/cbcschools"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/logger"
	"somotracker/backend/internal/middleware"
)

func main() {
	fx.New(
		// Global dependencies
		config.Module,
		logger.Module,
		database.Module,

		// Feature modules
		cbcschools.Module,
		auth.Module,

		// Entrypoint – must be wrapped in fx.Invoke
		fx.Invoke(func(
			lc fx.Lifecycle,
			cfg config.Config,
			pools *database.Pools,
			authhandler *auth.Handler,
			log *zap.Logger,
		) {
			// Build Fiber app with the canonical error handler: domain errors are
			// mapped to the standard {code, message, errors} body via
			// middleware.HTTPError, while Fiber's built-in 404/405 responses are
			// preserved.
			app := fiber.New(fiber.Config{
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					if errors.Is(err, fiber.ErrNotFound) || errors.Is(err, fiber.ErrMethodNotAllowed) {
						return fiber.DefaultErrorHandler(c, err)
					}
					return middleware.HTTPError(c, err)
				},
				AppName: "somotracker-api",
				// Deliberate limits: 4 MB JSON body cap (import endpoints are
				// JSON job polling — no file uploads), plus server-level timeouts
				// so a hung client or handler can't hold connections open
				// indefinitely.
				BodyLimit:    4 * 1024 * 1024,
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			})

			// Health check
			app.Get("/health", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"status": "ok"})
			})

			// Global security + context middleware (session resolver, CSRF guard,
			// rate limiters, device fingerprint). Must run before routes so that
			// middleware.RequireAuth (used by protected auth routes) can read the
			// resolved session from Locals (D1).
			middleware.Register(app, pools, cfg, log.Sugar())

			// Register auth routes
			authhandler.RegisterRoutes(app)

			// Lifecycle hooks for non-blocking Fiber server
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Info("starting server", zap.String("port", cfg.Port))
					go func() {
						if err := app.Listen(":" + cfg.Port); err != nil {
							log.Error("server listen failed", zap.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					log.Info("shutting down fiber server")
					return app.ShutdownWithContext(ctx)
				},
			})
		}),
	).Run()
}

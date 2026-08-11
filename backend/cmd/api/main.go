package main

import (
	"context"
	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		// Global dependencies
		config.Module,
		logger.Module,
		database.Module,

		// Feature modules
		auth.Module,

		// Entrypoint – must be wrapped in fx.Invoke
		fx.Invoke(func(
			lc fx.Lifecycle,
			cfg config.Config,
			authhandler *auth.Handler,
			log *zap.Logger, // Injected if provided in your dependencies, or remove if unused
		) {
			// Build Fiber app
			app := fiber.New()

			// Health check
			app.Get("/health", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"status": "ok"})
			})

			// Register auth routes
			authhandler.RegisterRoutes(app)

			// Start server
			port := cfg.Port
			if port == "" {
				port = "3000"
			}

			// Lifecycle hooks for non-blocking Fiber server
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Info("starting server", zap.String("port", port))
					go func() {
						if err := app.Listen(":" + port); err != nil {
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

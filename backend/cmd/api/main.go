package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

func main() {
	app := fx.New(
		fx.Provide(config.Load),
		fx.Provide(newLogger),
		fx.Provide(newFiberApp),
		fx.Invoke(registerHooks),
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 15*time.Second)
	if err := app.Start(startCtx); err != nil && !errors.Is(err, context.Canceled) {
		cancelStart()
		os.Exit(1)
	}
	cancelStart()

	<-sigCh

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 30*time.Second)
	if err := app.Stop(stopCtx); err != nil {
		cancelStop()
		os.Exit(1)
	}
	cancelStop()
}

// newLogger creates a zap.Logger based on the injected config.
func newLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.IsProduction() {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

// newFiberApp creates and configures a Fiber v3 application with health endpoints.
func newFiberApp(cfg *config.Config, logger *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "somotracker-api",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Do not treat 404 route misses as server errors.
			if errors.Is(err, fiber.ErrNotFound) || (err != nil && err.Error() == "Not Found") {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"code":    "not_found",
					"message": "Resource not found",
					"errors":  fiber.Map{},
				})
			}

			logger.Error("unhandled error",
				zap.Error(err),
				zap.String("env", cfg.Environment),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    "internal_error",
				"message": "An unexpected error occurred",
				"errors":  fiber.Map{},
			})
		},
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "somotracker-api",
		})
	})

	app.Get("/livez", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/readyz", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	return app
}

// registerHooks wires the Fiber server start/stop lifecycle using the injected
// config. The listen address is derived cleanly from Config.ListenAddr().
func registerHooks(lc fx.Lifecycle, cfg *config.Config, app *fiber.App, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				addr := cfg.ListenAddr()
				logger.Info("starting HTTP server",
					zap.String("address", addr),
					zap.Int("port", cfg.Port),
					zap.String("env", cfg.Environment),
				)
				if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("server error",
						zap.Error(err),
						zap.String("address", addr),
					)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down HTTP server",
				zap.String("address", cfg.ListenAddr()),
			)
			return app.Shutdown()
		},
	})
}

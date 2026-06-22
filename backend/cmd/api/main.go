// @title           Somotracker API
// @version         0.1.0
// @description     REST API for the Somotracker educational analytics platform.
//
// @contact.name   Somotracker Team
//
// @license.name  Proprietary
//
// @host           localhost:3030
// @BasePath       /
//
// @tag.name       Auth
// @tag.description Authentication and session management endpoints
// @tag.name       Tenants
// @tag.description Tenant (school) management endpoints
// @tag.name       Education Systems
// @tag.description Education system (curriculum framework) endpoints
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hibiken/asynq"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/imports"
	"somotracker/backend/internal/members"
	"somotracker/backend/internal/middleware"
	"somotracker/backend/internal/tenant"
	"somotracker/backend/internal/utils"
)

// Global Fiber error handler registered in fiber.Config.
// This is the last-resort catcher for any error that escapes handler functions
// (including panics caught by Fiber's recover middleware).
// It logs with slog.ErrorContext and returns the standard error response body.
func globalErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var message string
	var errorCode string

	// Try to get status code from Fiber's built-in error type
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	// Default to internal_error
	if message == "" {
		message = "an unexpected error occurred"
	}
	if errorCode == "" {
		errorCode = "internal_error"
	}

	// Log the error
	slog.LogAttrs(c.Context(), slog.LevelError,
		"global error handler",
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", code),
		slog.String("error", err.Error()),
	)

	return c.Status(code).JSON(fiber.Map{
		"code":    errorCode,
		"message": message,
	})
}

func main() {
	fx.New(
		config.Module,
		database.Module,
		utils.Module,
		tenant.Module,
		auth.Module,
		members.Module,
		imports.AsynqModule,
		imports.AsynqServerModule,
		imports.Module,

		fx.Provide(newLogger),
		fx.Invoke(runMigrations),
		fx.Invoke(registerApp),
		fx.Invoke(startAsynqWorker),
		fx.Invoke(consumeSafeClient),
	).Run()
}

func newLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

func errToStatus(err error) string {
	if err == nil {
		return "healthy"
	}
	return "unhealthy: " + err.Error()
}

// startAsynqWorker starts the Asynq background worker for import processing.
func startAsynqWorker(lc fx.Lifecycle, asynqServer *asynq.Server, importWorker *imports.Worker, logger *zap.Logger) {
	// Register the import processor handler
	mux := asynq.NewServeMux()
	mux.HandleFunc(imports.TypeProcessImport, importWorker.ProcessImport)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("starting asynq worker")
				if err := asynqServer.Start(mux); err != nil {
					logger.Error("asynq server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down asynq worker")
			asynqServer.Shutdown()
			return nil
		},
	})
}

func consumeSafeClient(client *http.Client) {
	// intentional no-op: ensures the SSRF-safe client is wired into
	// the fx container so it is available to future consumers without
	// triggering an unused-provision warning.
}

// runMigrations applies pending database migrations before the HTTP server
// starts. Invoked via fx so it runs during the container startup phase, before
// any lifecycle OnStart hooks.
func runMigrations(cfg config.Config) error {
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		// Log with slog and return the error so fx refuses to start the app
		slog.Error("migration failed", "error", err)
		return err
	}
	return nil
}

func registerApp(
	lc fx.Lifecycle,
	cfg config.Config,
	pools *database.Pools,
	tenantHandler *tenant.Handler,
	authHandler *auth.Handler,
	membersHandler *members.Handler,
	importsHandler *imports.Handler,
) {
	app := fiber.New(fiber.Config{
		AppName:      "somotracker",
		ErrorHandler: globalErrorHandler,
	})

	// Register Fiber's built-in recover middleware before all routes
	// so that handler panics are caught and routed to the error handler
	// rather than crashing the process.
	app.Use(fiberrecover.New())

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Mount the full security middleware pipeline
			middleware.Register(app, pools, cfg)

			// Register global health endpoint
			app.Get("/health", func(c *fiber.Ctx) error {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				pgErr := pools.PG.Ping(ctx)
				redErr := pools.Redis.Ping(ctx).Err()
				return c.JSON(fiber.Map{
					"status":   "ok",
					"postgres": errToStatus(pgErr),
					"redis":    errToStatus(redErr),
					"env":      cfg.AppEnv,
				})
			})

			// Mount domain routes
			tenantHandler.RegisterRoutes(app)
			authHandler.RegisterRoutes(app)
			membersHandler.RegisterRoutes(app)
			importsHandler.RegisterRoutes(app)

			// Start Fiber in a non-blocking goroutine
			go func() {
				if err := app.Listen(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
					// Log fatal since this means the server failed to start
					slog.Error("fiber listen fatal", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Bounded shutdown window: 15 seconds total
			shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			var shutdownErr error

			// 1. Gracefully drain Fiber (in-flight requests)
			if err := app.ShutdownWithContext(shutdownCtx); err != nil {
				shutdownErr = errors.Join(shutdownErr, err)
			}

			// 2. Close Postgres pool
			pools.PG.Close()

			// 3. Close Redis client
			if err := pools.Redis.Close(); err != nil {
				shutdownErr = errors.Join(shutdownErr, err)
			}

			if shutdownErr != nil {
				slog.ErrorContext(ctx, "registerApp.OnStop: shutdown error", "error", shutdownErr)
			}

			return shutdownErr
		},
	})
}

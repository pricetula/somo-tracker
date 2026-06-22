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
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
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
	_ = client
}

// runMigrations applies pending database migrations before the HTTP server
// starts. Invoked via fx so it runs during the container startup phase, before
// any lifecycle OnStart hooks.
func runMigrations(cfg config.Config) {
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("[migrate] fatal: %v", err)
	}
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
		AppName: "somotracker",
	})

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
					log.Fatalf("fiber listen: %v", err)
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

			return shutdownErr
		},
	})
}

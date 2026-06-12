package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
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

		fx.Provide(newLogger),
		fx.Invoke(registerApp),
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

func consumeSafeClient(client *http.Client) {
	// intentional no-op: ensures the SSRF-safe client is wired into
	// the fx container so it is available to future consumers without
	// triggering an unused-provision warning.
	_ = client
}

func registerApp(
	lc fx.Lifecycle,
	cfg config.Config,
	pools *database.Pools,
	tenantHandler *tenant.Handler,
	authHandler *auth.Handler,
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

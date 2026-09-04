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
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/api"
	"somotracker/backend/internal/api/middleware"
	"somotracker/backend/internal/api/middleware/ratelimit"
	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/database/sqlc"
	"somotracker/backend/internal/observability"
	"somotracker/backend/internal/redis"
	"somotracker/backend/internal/services"
	"somotracker/backend/internal/stytch"
)

func main() {
	app := fx.New(
		config.Module,
		fx.Provide(newLogger),
		fx.Provide(database.NewPool),
		fx.Provide(newQuerier),
		fx.Provide(services.NewTenantService),
		fx.Provide(services.NewUserService),
		fx.Provide(services.NewAuthService),
		fx.Provide(api.NewRouter),
		fx.Provide(observability.NewTracerProvider),
		fx.Provide(observability.NewMeterProvider),
		fx.Invoke(observability.MetricsInvoke),
		fx.Provide(middleware.NewRequestIDHandler),
		fx.Provide(newFiberApp),
		fx.Invoke(database.RunMigrations),
		redis.Module,
		ratelimit.Module,
		stytch.Module,
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

// newQuerier adapts the sqlc constructor for Fx. Fx can resolve concrete
// types automatically, but it cannot infer that *sqlc.Queries satisfies
// sqlc.Querier. A named adapter (returning the interface) is the cleanest
// way to make the interface the public DI type without leaking the pool.
func newQuerier(pool *pgxpool.Pool) *sqlc.Queries {
	return sqlc.New(pool)
}

// newFiberApp creates and configures a Fiber v3 application with health
// endpoints. The /readyz handler pings the database connection pool so the
// API only reports ready when PostgreSQL is reachable.
func newFiberApp(cfg *config.Config, logger *zap.Logger, pool *pgxpool.Pool, router *api.Router, reqIDHandler fiber.Handler) *fiber.App {
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

	// Liveness: process is running. Cheap — no I/O.
	app.Get("/livez", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Readiness: process is ready to serve traffic. Pings PostgreSQL.
	// On failure we return 503 so the orchestrator (k8s, load balancer)
	// stops routing requests until the database is reachable again.
	app.Get("/readyz", func(c fiber.Ctx) error {
		// Cap the readiness probe at the request's own deadline (Fiber
		// already enforces a server-level timeout). Ping has its own
		// internal cap of 2s.
		if err := database.Ping(c.Context(), pool); err != nil {
			logger.Warn("readiness probe failed: database unreachable",
				zap.Error(err),
			)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"code":    "database_unavailable",
				"message": "Database is not reachable",
				"errors":  fiber.Map{},
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	app.Use(reqIDHandler)
	router.RegisterRoutes(app)
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

package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/pressly/goose/v3"

	"somotracker/internal/auth/handler"
	"somotracker/internal/auth/service"
	"somotracker/internal/auth/store"
	storegen "somotracker/internal/auth/store/gen"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/brevo"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/db"
	"somotracker/internal/platform/httpserver"
	"somotracker/internal/platform/logger"
	"somotracker/internal/platform/redis"
)

// App is the HTTP API composition root.
type App struct {
	cfg      *config.Config
	dbPool   *db.Pool
	redis    *redis.Client
	server   *httpserver.Server
	log      *logger.Logger
	authSvc  *service.Service
	authHdl  *handler.Handler
}

// New creates the API app with manual DI.
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	log := logger.New(cfg.LogLevel)

	// Database pool
	dbPool, err := db.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// Run database migrations on startup
	if err := runMigrations(ctx, dbPool.StdlibDB, cfg, log); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Redis client
	redisClient, err := redis.New(ctx, cfg.RedisURL)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	// HTTP server
	server := httpserver.New(cfg.AppEnv)

	// Brevo email sender
	codeSender := brevo.New(cfg.BrevoAPIKey, cfg.BrevoSenderEmail, cfg.BrevoSenderName)

	// SQL queries (sqlc uses stdlib DBTX interface)
	queries := storegen.New(dbPool.StdlibDB)

	// Auth store
	authStore := store.New(queries, dbPool)

	// Auth service
	authSvc := service.New(authStore, codeSender, cfg, log)

	// Auth handler
	authHdl := handler.New(authSvc, cfg, log)

	return &App{
		cfg:       cfg,
		dbPool:    dbPool,
		redis:     redisClient,
		server:    server,
		log:       log,
		authSvc:   authSvc,
		authHdl:   authHdl,
	}, nil
}

// Start starts the HTTP server.
func (a *App) Start(ctx context.Context) error {
	app := a.server.App()

	// Middleware chain
	app.Use(func(c fiber.Ctx) error {
		return c.Next()
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Public auth routes
	auth := api.Group("/auth")
	a.authHdl.RegisterRoutes(auth)

	// Health check
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/readyz", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.dbPool.Ping(ctx); err != nil {
			return c.Status(http.StatusServiceUnavailable).SendString("DB not ready")
		}
		if err := a.redis.Client.Ping(ctx).Err(); err != nil {
			return c.Status(http.StatusServiceUnavailable).SendString("Redis not ready")
		}
		return c.SendString("Ready")
	})

	slog.Info("starting API", "port", a.cfg.ServerPort)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Start(ctx, a.cfg.ServerPort)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	return a.Shutdown(ctx)
}

// Shutdown gracefully stops the app.
func (a *App) Shutdown(ctx context.Context) error {
	slog.Info("shutting down API")
	if err := a.server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "err", err)
	}
	if err := a.redis.Close(); err != nil {
		slog.Error("redis close failed", "err", err)
	}
	a.dbPool.Close()
	slog.Info("shutdown complete")
	return nil
}

// LoadConfig loads configuration from environment.
func LoadConfig() (*config.Config, error) {
	return config.Load()
}

// ErrorHandler is a centralized Fiber error handler that renders standardized errors.
func ErrorHandler() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		code := http.StatusInternalServerError
		appCode := apperr.INTERNAL_ERROR
		message := "internal error"

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		} else if appErr, ok := err.(*apperr.Error); ok {
			switch appErr.Code {
			case apperr.INVALID_REQUEST, apperr.INVALID_EMAIL, apperr.TURNSTILE_REQUIRED, apperr.TURNSTILE_INVALID:
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
			default:
				code = http.StatusInternalServerError
			}
			appCode = appErr.Code
			message = appErr.Message
		}

		return c.Status(code).JSON(fiber.Map{
			"code":    appCode,
			"message": message,
		})
	}
}

// runMigrations applies all pending database migrations and logs the version.
func runMigrations(ctx context.Context, db *sql.DB, cfg *config.Config, log *logger.Logger) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get migration version: %w", err)
	}

	log.Info(ctx, "migrations applied", "version", version, "dir", cfg.MigrationsDir)
	return nil
}

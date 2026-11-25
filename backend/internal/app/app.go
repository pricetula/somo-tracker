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
	eduStore "somotracker/internal/education/store"
	eduHandler "somotracker/internal/education/handler"
	eduService "somotracker/internal/education/service"
	"somotracker/internal/platform/brevo"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/db"
	"somotracker/internal/platform/httpserver"
	"somotracker/internal/platform/logger"
	"somotracker/internal/platform/middleware"
	"somotracker/internal/platform/ratelimit"
	"somotracker/internal/platform/redis"
)

// App is the HTTP API composition root.
type App struct {
	cfg               *config.Config
	dbPool            *db.Pool
	redis             *redis.Client
	server            *httpserver.Server
	log               *logger.Logger
	authSvc           *service.Service
	authHdl           *handler.Handler
	eduSvc            *eduService.Service
	eduHdl            *eduHandler.Handler
	globalRateLimiter *ratelimit.GlobalLimiter
	sessionResolver   *middleware.SessionResolver
}

// New creates the API app with manual DI.
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	log := logger.New(cfg.LogLevel)

	// Database pool
	dbPool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// Run database migrations on startup
	if err := runMigrations(ctx, dbPool.StdlibDB, log); err != nil {
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
	authSvc := service.New(authStore, codeSender, redisClient, cfg, log)

	// Rate limiter (Redis-backed sliding window)
	rateLimitStore := ratelimit.NewRedisStore(redisClient)
	rateLimiter := ratelimit.NewRedisLimiter(rateLimitStore)
	if err := rateLimiter.LoadScript(ctx); err != nil {
		return nil, fmt.Errorf("load rate limit script: %w", err)
	}
	keyBuilder := ratelimit.NewKeyBuilder(cfg)

	// Auth handler
	authHdl := handler.New(authSvc, cfg, log, rateLimiter, keyBuilder)

	// Education store, service, handler
	eduStoreInst := eduStore.New(queries)
	eduSvc := eduService.New(eduStoreInst, cfg, log)
	eduHdl := eduHandler.New(eduSvc, cfg, log)

	// Global rate limiter (for all /api/v1/* routes)
	globalRateLimiter := ratelimit.NewGlobalLimiter(rateLimiter, keyBuilder, cfg)

	// Session resolution middleware
	sessionResolver := middleware.NewSessionResolver(authSvc.Store(), cfg)

	return &App{
		cfg:                cfg,
		dbPool:             dbPool,
		redis:              redisClient,
		server:             server,
		log:                log,
		authSvc:            authSvc,
		authHdl:            authHdl,
		eduSvc:             eduSvc,
		eduHdl:             eduHdl,
		globalRateLimiter:  globalRateLimiter,
		sessionResolver:    sessionResolver,
	}, nil
}

// Start starts the HTTP server.
func (a *App) Start(ctx context.Context) error {
	app := a.server.App()

	// ============ MIDDLEWARE CHAIN ============
	// Order matters: security first, then identification, then rate limiting

	// 1. Exempt paths (health, readiness, webhooks) - no auth required
	exemptPaths := map[string]bool{
		"/healthz": true,
		"/readyz":  true,
	}
	app.Use(func(c fiber.Ctx) error {
		if exemptPaths[c.Path()] {
			return c.Next()
		}
		return c.Next() // Continue to next middleware
	})

	// 2. Verify BFF Service Secret (blocks direct public-URL access)
	app.Use(middleware.VerifyBFFServiceSecret(a.cfg))

	// 3. Verify Trusted BFF CIDR (optional, additional defense)
	app.Use(middleware.VerifyTrustedBFFCIDR(a.cfg))

	// 4. Extract and normalize Client IP from X-Client-IP header
	app.Use(middleware.ExtractClientIP(a.cfg))

	// 5. Session resolution middleware (attaches userID to context)
	app.Use(a.sessionResolver.Resolve())

	// 6. Global rate limit middleware (keys on userID > IP)
	app.Use(a.globalRateLimiter.Middleware())

	// ============ ROUTES ============
	// API v1 routes
	api := app.Group("/api/v1")

	// Public auth routes
	auth := api.Group("/auth")
	a.authHdl.RegisterRoutes(auth)

	// Education routes (public read)
	edu := api.Group("/education")
	a.eduHdl.RegisterRoutes(edu)

	// Health check (exempt from auth)
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

// migrationsDir is the fixed path to the SQL migration files. It lives in the
// repo and never changes at runtime, so it is not configuration.
const migrationsDir = "./migrations"

// runMigrations applies all pending database migrations and logs the version.
func runMigrations(ctx context.Context, db *sql.DB, log *logger.Logger) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get migration version: %w", err)
	}

	log.Info(ctx, "migrations applied", "version", version, "dir", migrationsDir)
	return nil
}
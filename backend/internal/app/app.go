package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	_ "github.com/jackc/pgx/v5/stdlib"

	redisPkg "somotracker/internal/platform/redis"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/db"
	"somotracker/internal/platform/httpserver"
	"somotracker/internal/platform/migrator"
	"somotracker/internal/platform/ratelimit"
	"somotracker/internal/platform/stytch"
	identityHandler "somotracker/internal/identity/handler"
	identityMiddleware "somotracker/internal/identity/middleware"
	identityService "somotracker/internal/identity/service"
	identityStore "somotracker/internal/identity/store"
	storegen "somotracker/internal/identity/store/gen"
)

// App is the HTTP API composition root.
type App struct {
	cfg         *config.Config
	dbPool      *db.Pool
	sqlDB       *sql.DB
	redisClient *redisPkg.Client
	server      *httpserver.Server
}

// New creates the API app with manual DI (BE-005).
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// Database pool for Fiber
	dbPool, err := db.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	// SQL DB for migrations and sqlc queries
	sqlDB, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		dbPool.Close()
		return nil, err
	}

	// Migrator runs Goose migrations on startup
	m := migrator.New(sqlDB, cfg.PostgresDSN)
	if _, err := m.Up(ctx); err != nil {
		_ = sqlDB.Close()
		dbPool.Close()
		return nil, err
	}

	// Redis client
	redisClient, err := redisPkg.New(ctx, cfg.RedisAddr)
	if err != nil {
		_ = sqlDB.Close()
		dbPool.Close()
		return nil, err
	}

	// HTTP server
	server := httpserver.New(cfg.Env)

	return &App{
		cfg:         cfg,
		dbPool:      dbPool,
		sqlDB:       sqlDB,
		redisClient: redisClient,
		server:      server,
	}, nil
}

// Start starts HTTP server with graceful shutdown support.
func (a *App) Start(ctx context.Context) error {
	// Configure Fiber app
	app := a.server.App()
	app.Use(recover.New())
	app.Use(helmet.New())
	// Security headers middleware
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if a.cfg.Env == "production" && c.Secure() {
			c.Set("Strict-Transport-Security", "max-age=31536000")
		}
		return c.Next()
	})
	origins := strings.Split(a.cfg.CORSOrigins, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} ${method} ${path} ${latency}\n",
		TimeFormat: time.RFC3339,
		TimeZone:   "UTC",
	}))

	// Health check (not under /api per BE-026)
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/readyz", func(c fiber.Ctx) error {
		// Check DB connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.dbPool.Ping(ctx); err != nil {
			return c.Status(http.StatusServiceUnavailable).SendString("DB not ready")
		}
		// Check Redis connectivity
		if err := a.redisClient.Ping(ctx); err != nil {
			return c.Status(http.StatusServiceUnavailable).SendString("Redis not ready")
		}
		return c.SendString("Ready")
	})

	// Wire identity domain
	baseURL := "https://test.stytch.com"
	if a.cfg.StytchEnv == "live" {
		baseURL = "https://api.stytch.com"
	}
	stytchClient := stytch.NewSDKClient(a.cfg.StytchProjectID, a.cfg.StytchSecret, baseURL)
	queries := storegen.New(a.sqlDB)
	identityStore := identityStore.New(queries)
	identitySvc := identityService.New(identityStore, stytchClient, a.sqlDB)
	rl := ratelimit.New(a.redisClient.Client, a.cfg.EmailHMACKey)
	identityHandler := identityHandler.New(identitySvc, a.cfg, rl)

	// Create auth middleware
	authMiddleware := identityMiddleware.NewAuthMiddleware(identitySvc, a.cfg)

	// Public routes (no auth required)
	publicAPI := a.server.App().Group("/api/v1")
	identityHandler.RegisterRoutes(publicAPI)

	// Protected routes (require authentication)
	protectedAPI := a.server.App().Group("/api/v1")
	protectedAPI.Use(authMiddleware.RequireAuth())
	protectedAPI.Use(authMiddleware.TenantMiddleware())
	// TODO: Register protected domain handlers here
	// e.g., studentsHandler.RegisterRoutes(protectedAPI)

	// Mutations require CSRF protection
	mutationAPI := a.server.App().Group("/api/v1")
	mutationAPI.Use(authMiddleware.RequireAuth())
	mutationAPI.Use(authMiddleware.TenantMiddleware())
	mutationAPI.Use(authMiddleware.CSRFMiddleware())
	// TODO: Register mutation handlers here

	// Register API routes (legacy)
	a.server.RegisterRoutes()

	slog.Info("starting API", "port", a.cfg.ServerPort)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Start(ctx, ":"+a.cfg.ServerPort)
	}()

	// Wait for shutdown signal or server error
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

	// Shutdown HTTP server
	if err := a.server.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown failed", "err", err)
	}

	// Close Redis
	if err := a.redisClient.Close(); err != nil {
		slog.Error("Redis close failed", "err", err)
	}

	// Close DB pool
	a.dbPool.Close()

	slog.Info("shutdown complete")
	return nil
}

// LoadConfig loads typed config from environment via Doppler (BE-033).
func LoadConfig() (*config.Config, error) {
	return config.Load()
}

// Worker is the background worker composition root.
type Worker struct {
	cfg         *config.Config
	dbPool      *db.Pool
	redisClient *redisPkg.Client
}

// NewWorker creates worker with manual DI.
func NewWorker(cfg *config.Config) (*Worker, error) {
	ctx := context.Background()

	dbPool, err := db.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	redisClient, err := redisPkg.New(ctx, cfg.RedisAddr)
	if err != nil {
		dbPool.Close()
		return nil, err
	}

	return &Worker{
		cfg:         cfg,
		dbPool:      dbPool,
		redisClient: redisClient,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	slog.Info("starting worker")
	// TODO: start asynq server
	<-ctx.Done()
	return w.Shutdown(ctx)
}

func (w *Worker) Shutdown(ctx context.Context) error {
	slog.Info("shutting down worker")
	if err := w.redisClient.Close(); err != nil {
		slog.Error("Redis close failed", "err", err)
	}
	w.dbPool.Close()
	return nil
}
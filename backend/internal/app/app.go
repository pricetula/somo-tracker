package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"

	identityHandler "somotracker/internal/identity/handler"
	identityMiddleware "somotracker/internal/identity/middleware"
	identityService "somotracker/internal/identity/service"
	identityStore "somotracker/internal/identity/store"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/platform/audit"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/db"
	"somotracker/internal/platform/httpserver"
	"somotracker/internal/platform/ratelimit"
	redisPkg "somotracker/internal/platform/redis"
	"somotracker/internal/platform/seed"
	"somotracker/internal/platform/stytch"
)

// App is the HTTP API composition root.
type App struct {
	cfg           *config.Config
	dbPool        *db.Pool
	redisClient   *redisPkg.Client
	server        *httpserver.Server
	rateLimiter   *ratelimit.Limiter
	auditRecorder *audit.Recorder
	txManager     *db.TxManager
}

// New creates the API app with manual DI (BE-005).
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// Database pool for Fiber
	dbPool, err := db.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	// Seed reference data after migrations (opt-in via SEED_REFERENCE_DATA)
	// Migrations must be run as a separate deploy step (not on app startup)
	// with a privileged role while the app connects as a non-owner role.
	if cfg.SeedReferenceData {
		slog.Info("seeding reference data")
		seeder := seed.New(dbPool.DB)
		if err := seeder.Run(ctx); err != nil {
			_ = dbPool.Close()
			return nil, err
		}
	}

	// Redis client
	redisClient, err := redisPkg.New(ctx, cfg.RedisAddr)
	if err != nil {
		_ = dbPool.Close()
		return nil, err
	}

	// HTTP server
	server := httpserver.New(cfg.Env)

	// Audit recorder
	auditRecorder := audit.New(slog.Default())

	// Transaction manager for per-service boundaries and RLS GUC
	txManager := db.NewTxManager(dbPool.DB)

	return &App{
		cfg:           cfg,
		dbPool:        dbPool,
		redisClient:   redisClient,
		server:        server,
		rateLimiter:   ratelimit.New(redisClient.Client, cfg.EmailHMACKey),
		auditRecorder: auditRecorder,
		txManager:     txManager,
	}, nil
}

// Start starts HTTP server with graceful shutdown support.
func (a *App) Start(ctx context.Context) error {
	// Configure Fiber app
	app := a.server.App()

	// Middleware chain order (critical for security):
	// 1. RequestID - first, always
	app.Use(httpserver.RequestID())

	// 2. Recover - catches panics, converts to 500 via ErrorHandler
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, e interface{}) {
			slog.Error("panic recovered", "err", e, "request_id", c.Locals("request_id"))
		},
	}))

	// 3. Helmet - security headers
	app.Use(helmet.New())

	// 4. Custom security headers (CSP, HSTS, etc.)
	app.Use(a.securityHeadersMiddleware())

	// 5. CORS - configured origins only, no wildcards with credentials
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

	// 6. AccessLog - single completion line per request
	app.Use(httpserver.AccessLog())

	// Initialize services
	stytchClient := stytch.NewSDKClient(a.cfg.StytchProjectID, a.cfg.StytchSecret)
	queries := storegen.New(a.dbPool)
	identityStore := identityStore.New(queries, a.dbPool)
	identitySvc, err := identityService.New(identityStore, stytchClient, a.cfg.ISTEncryptionKey, a.cfg.SessionIdleTimeout)
	if err != nil {
		return fmt.Errorf("init identity service: %w", err)
	}

	// Rate limiter with in-memory fallback
	a.rateLimiter = ratelimit.New(a.redisClient.Client, a.cfg.EmailHMACKey)

	// Initialize rate limit middleware
	rateLimitMiddleware := httpserver.NewRateLimitMiddleware(a.rateLimiter, a.cfg)

	// Start memory fallback cleanup goroutine
	go a.rateLimiterCleanup(ctx)

	identityHandler := identityHandler.New(identitySvc, a.cfg, a.rateLimiter, a.auditRecorder)

	// Create auth middleware
	authMiddleware := identityMiddleware.NewAuthMiddleware(identitySvc, a.cfg, a.auditRecorder)

	// Wrap every request in a transaction so RLS GUC (SET LOCAL) is set
	// inside a transaction by TxManager.Run, and all store operations share
	// the same transaction context via db.GetTx(ctx).
	app.Use(func(c fiber.Ctx) error {
		return a.txManager.Run(c.Context(), func(ctx context.Context) error {
			c.SetContext(ctx)
			return c.Next()
		})
	})

	// API v1 routes — one group, sub-groups for different middleware chains.
	// This avoids registering the same path prefix on multiple independent
	// groups, which can cause duplicate route handling.
	api := a.server.App().Group("/api/v1")

	// Public routes (no auth required)
	publicAPI := api.Group("", rateLimitMiddleware.PublicRateLimit())
	identityHandler.RegisterRoutes(publicAPI)

	// Protected routes (require authentication + tenant context)
	// No protected domain handlers registered yet — deferred to later specs.
	_ = api.Group("",
		rateLimitMiddleware.UnauthenticatedRateLimit(), // Early IP limiter before RequireAuth
		authMiddleware.RequireAuth(),
		authMiddleware.TenantMiddleware(),
		rateLimitMiddleware.ProtectedRateLimit(),
	)

	// Mutation routes (require auth + tenant + CSRF)
	// No mutation handlers registered yet — deferred to later specs.
	_ = api.Group("",
		authMiddleware.RequireAuth(),
		authMiddleware.TenantMiddleware(),
		rateLimitMiddleware.MutationRateLimit(),
		authMiddleware.CSRFMiddleware(),
	)

	// Legacy routes
	a.server.RegisterRoutes()

	// Health checks (no auth, no rate limit)
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/readyz", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.dbPool.PingContext(ctx); err != nil {
			return c.Status(http.StatusServiceUnavailable).SendString("DB not ready")
		}
		if err := a.redisClient.Ping(ctx); err != nil {
			return c.Status(http.StatusServiceUnavailable).SendString("Redis not ready")
		}
		return c.SendString("Ready")
	})

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

// securityHeadersMiddleware applies security headers to all responses.
func (a *App) securityHeadersMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		if a.cfg.Env == "production" && (c.Secure() || c.Get("X-Forwarded-Proto") == "https") {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		if a.cfg.Env == "production" {
			c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		} else if a.cfg.CSPReportOnlyEnabled {
			policy := a.cfg.CSPReportOnlyPolicy
			if a.cfg.CSPReportOnlyURI != "" {
				policy += "; report-uri " + a.cfg.CSPReportOnlyURI
			}
			c.Set("Content-Security-Policy-Report-Only", policy)
		}

		return c.Next()
	}
}

// rateLimiterCleanup periodically cleans up in-memory rate limit store.
func (a *App) rateLimiterCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.rateLimiter.MemoryCleanup(24 * time.Hour)
		}
	}
}

// Shutdown gracefully stops the app.
func (a *App) Shutdown(ctx context.Context) error {
	slog.Info("shutting down API")
	if err := a.server.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown failed", "err", err)
	}
	if err := a.redisClient.Close(); err != nil {
		slog.Error("Redis close failed", "err", err)
	}
	_ = a.dbPool.Close()
	slog.Info("shutdown complete")
	return nil
}

// LoadConfig loads typed config from environment via Doppler (BE-033).
func LoadConfig() (*config.Config, error) {
	return config.Load()
}

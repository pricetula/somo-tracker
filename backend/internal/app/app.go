package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	_ "github.com/jackc/pgx/v5/stdlib"

	identityHandler "somotracker/internal/identity/handler"
	identityMiddleware "somotracker/internal/identity/middleware"
	identityService "somotracker/internal/identity/service"
	identityStore "somotracker/internal/identity/store"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/db"
	"somotracker/internal/platform/httpserver"
	"somotracker/internal/platform/migrator"
	"somotracker/internal/platform/ratelimit"
	redisPkg "somotracker/internal/platform/redis"
	"somotracker/internal/platform/stytch"
)

// App is the HTTP API composition root.
type App struct {
	cfg         *config.Config
	dbPool      *db.Pool
	sqlDB       *sql.DB
	redisClient *redisPkg.Client
	server      *httpserver.Server
	rateLimiter *ratelimit.Limiter
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
	// NOTE: In production, migrations should run as a separate deploy step
	// with a privileged role, while the app connects as a non-owner role
	// with FORCE ROW LEVEL SECURITY enabled.
	m := migrator.New(sqlDB, "migrations")
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
		// Vary: Origin is automatically added by Fiber when AllowCredentials=true
	}))

	// 6. AccessLog - single completion line per request
	// Note: We do NOT use fiber's logger.New() here - it duplicates AccessLog
	// and can leak sensitive data in production. httpserver.AccessLog()
	// emits a single structured line per request.
	app.Use(httpserver.AccessLog())

	// Initialize services
	stytchClient := stytch.NewSDKClient(a.cfg.StytchProjectID, a.cfg.StytchSecret)
	queries := storegen.New(a.sqlDB)
	identityStore := identityStore.New(queries)
	identitySvc, err := identityService.New(identityStore, stytchClient, a.sqlDB, a.cfg.ISTEncryptionKey)
	if err != nil {
		return fmt.Errorf("init identity service: %w", err)
	}

	// Rate limiter with in-memory fallback
	a.rateLimiter = ratelimit.New(a.redisClient.Client, a.cfg.EmailHMACKey)

	// Start memory fallback cleanup goroutine
	go a.rateLimiterCleanup(ctx)

	identityHandler := identityHandler.New(identitySvc, a.cfg, a.rateLimiter)

	// Create auth middleware
	authMiddleware := identityMiddleware.NewAuthMiddleware(identitySvc, a.cfg)
	rbacMiddleware := identityMiddleware.NewRBACMiddleware(identitySvc)

	// Public routes (no auth required)
	publicAPI := a.server.App().Group("/api/v1")
	identityHandler.RegisterRoutes(publicAPI)

	// Protected routes (require authentication + tenant context)
	protectedAPI := a.server.App().Group("/api/v1")
	protectedAPI.Use(authMiddleware.RequireAuth())
	protectedAPI.Use(authMiddleware.TenantMiddleware())
	protectedAPI.Use(a.setTenantContextMiddleware()) // Sets app.current_tenant_id for RLS
	// TODO: Register protected domain handlers here

	// Mutation routes (require auth + tenant + CSRF)
	mutationAPI := a.server.App().Group("/api/v1")
	mutationAPI.Use(authMiddleware.RequireAuth())
	mutationAPI.Use(authMiddleware.TenantMiddleware())
	mutationAPI.Use(a.setTenantContextMiddleware())
	mutationAPI.Use(authMiddleware.CSRFMiddleware())
	// TODO: Register mutation handlers here

	// Example: Finance routes with RBAC
	// financeAPI := mutationAPI.Group("/fees")
	// financeAPI.Use(rbacMiddleware.RequireFinanceAccess())
	// feesHandler.RegisterRoutes(financeAPI)

	_ = rbacMiddleware // used in commented example above

	// Legacy routes
	a.server.RegisterRoutes()

	// Health checks (no auth, no rate limit)
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/readyz", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.dbPool.Ping(ctx); err != nil {
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
		// Standard headers
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HSTS - detect HTTPS via X-Forwarded-Proto (set by trusted proxy)
		// c.Secure() reads from fasthttp which respects ProxyHeader when TrustProxy=true
		if a.cfg.Env == "production" && (c.Secure() || c.Get("X-Forwarded-Proto") == "https") {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// CSP for API: default-src 'none'; frame-ancestors 'none'
		// This is safe for JSON APIs and prevents any accidental HTML rendering
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

// setTenantContextMiddleware sets app.current_tenant_id for RLS policies.
// Must run after RequireAuth/TenantMiddleware which populate tenant_id in locals.
func (a *App) setTenantContextMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID, ok := c.Locals("tenant_id").(string)
		if !ok {
			// Try UUID type
			if tid, ok2 := c.Locals("tenant_id").([16]byte); ok2 {
				tenantID = fmtUUID(tid)
			}
		}
		if tenantID != "" {
			// Set LOCAL (transaction-scoped) GUC for RLS
			// This requires the DB connection to be in a transaction
			// For pgx pool, we set it per-request; the connection is released after
			// In practice, sqlc queries run in their own tx, so we need a wrapper
			// For now, store in context for repositories to use
			ctx := context.WithValue(c.Context(), httpserver.TenantIDKey{}, tenantID)
			c.SetContext(ctx)
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
			a.rateLimiter.MemoryCleanup(24 * time.Hour) // clean entries older than 24h
		}
	}
}

// fmtUUID formats a [16]byte as UUID string.
func fmtUUID(b [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
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
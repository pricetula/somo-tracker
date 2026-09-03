// Package database owns the lifecycle of the pgx connection pool and the
// automated migration runner used by the Somotracker backend.
//
// It exposes:
//
//  1. [NewPool] — Fx-managed constructor that builds a *pgxpool.Pool from
//     configuration values supplied by internal/config.
//  2. An Fx OnStop hook that gracefully closes the pool during application
//     shutdown.
//  3. [Ping] — health check used by the /readyz readiness endpoint.
//  4. [NewMigrator] — builds a golang-migrate instance from the shared pool
//     and the embedded migration file system.
//  5. [RunMigrations] — Fx Invoke hook that calls Migrator.Up() inside the
//     Fx startup sequence (after pool, before HTTP server).
//
// All errors are wrapped with package/type/method context so callers and the
// HTTP error handler can surface meaningful messages.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// Default timeouts used by [Ping] and pool construction.
const (
	// DefaultPingTimeout caps any individual health check against the pool.
	DefaultPingTimeout = 2 * time.Second

	// DefaultAcquireTimeout caps pool acquisition during Ping.
	DefaultAcquireTimeout = 1 * time.Second
)

// NewPool constructs a *pgxpool.Pool from the supplied configuration and
// registers an OnStop hook so the pool is closed cleanly when the Fx
// application terminates.
//
// The constructor intentionally returns (T, error) — required by the backend
// AGENTS.md DI contract — so any startup failure (bad DSN, unreachable host,
// pool init error) is propagated to fx, which refuses to start.
func NewPool(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*pgxpool.Pool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database.NewPool: config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("database.NewPool: logger is required")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database.NewPool: parse DATABASE_URL: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.DBMaxConns) //nolint:gosec // bounds-checked in validate()
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.DBMaxConnIdleTime
	poolCfg.MinConns = 0

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database.NewPool: build pool: %w", err)
	}

	// Probe the pool immediately so misconfiguration fails fast.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database.NewPool: initial ping: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("database pool initialized",
				zap.Int("max_conns", cfg.DBMaxConns),
				zap.Duration("max_conn_lifetime", cfg.DBMaxConnLifetime),
				zap.Duration("max_conn_idle_time", cfg.DBMaxConnIdleTime),
			)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			logger.Info("closing database pool")
			pool.Close()
			// pgxpool.Close is synchronous and does not return an error.
			// We still return nil so the hook satisfies the signature.
			return nil
		},
	})

	return pool, nil
}

// Ping verifies that the pool can acquire a connection and round-trip a
// health check within the configured timeout. It is intended for readiness
// probes (e.g. /readyz). Any error is wrapped with context so the HTTP layer
// can decide between 503 (transient) and 500 (unexpected).
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database.Ping: pool is nil")
	}
	if ctx == nil {
		return fmt.Errorf("database.Ping: context is nil")
	}

	pingCtx, cancel := context.WithTimeout(ctx, DefaultPingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("database.Ping: %w", err)
	}
	return nil
}

// RunMigrations is an Fx Invoke hook that executes database migrations
// immediately after the pool is initialized and before the HTTP server
// listens. It builds a Migrator from the shared pool and embedded FS,
// applies all pending migrations via Migrator.Up(), and logs progress
// through Zap.
//
// If migrations succeed, it returns nil. If migrations fail (file error,
// database error, or dirty state), it returns an error that aborts Fx
// startup — the container exits fast rather than running with stale schema.
//
// migrate.ErrNoChange is handled gracefully: an info-level log message is
// emitted ("database schema is up to date") and the function returns nil.
func RunMigrations(pool *pgxpool.Pool, logger *zap.Logger) error {
	migrator, err := NewMigrator(pool, logger)
	if err != nil {
		return fmt.Errorf("database.RunMigrations: build migrator: %w", err)
	}
	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			logger.Warn("migrator close error",
				zap.Error(closeErr),
			)
		}
	}()

	if upErr := migrator.Up(context.Background()); upErr != nil {
		return fmt.Errorf("database.RunMigrations: %w", upErr)
	}

	return nil
}

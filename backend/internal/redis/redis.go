// Package redis provides an Fx-managed Redis client with OpenTelemetry
// instrumentation, structured logging, and graceful lifecycle management.
//
// It exposes:
//
//  1. [Module] — a self-contained Fx module that can be imported by any Fx
//     application to gain a *redis.Client via dependency injection.
//  2. [NewClient] — Fx-managed constructor that builds a *redis.Client from
//     the supplied Config and wires lifecycle hooks.
//  3. [Ping] — health check used by readiness probes.
//
// OpenTelemetry tracing is enabled automatically via
// [github.com/redis/go-redis/extra/redisotel/v9]. All Redis commands emit
// spans with command name, key pattern, and error status.
//
// All errors are wrapped with package/type/method context so callers and the
// HTTP error handler can surface meaningful messages.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// Module is the Fx module that encapsulates all Redis dependencies.
// Import it in your fx.App with:
//
//	fx.Options(
//	    config.Module,
//	    redis.Module, // provides *redis.Client
//	)
//
// Other modules in the application can declare *redis.Client as a direct
// dependency without knowing about this package's internals.
var Module = fx.Module(
	"redis",
	fx.Provide(NewClient),
)

// Default timeouts used by [Ping] and client construction. These are applied
// after [redis.ParseURL] so any explicit overrides in REDIS_URL (e.g. via
// query string) take precedence.
const (
	// DefaultDialTimeout caps the TCP connection establishment to Redis.
	DefaultDialTimeout = 5 * time.Second

	// DefaultReadTimeout caps individual Redis command round-trips.
	DefaultReadTimeout = 3 * time.Second

	// DefaultWriteTimeout caps write round-trips. Set equal to ReadTimeout
	// unless you have commands that write large payloads (e.g. MSet, GeoAdd).
	DefaultWriteTimeout = 3 * time.Second

	// DefaultPoolSize is the default number of concurrent socket connections.
	// The go-redis client multiplexes commands over this pool, so a value
	// of 10 is usually sufficient for a single-process API server.
	DefaultPoolSize = 10

	// DefaultPingTimeout caps the health-check ping.
	DefaultPingTimeout = 2 * time.Second

	// ShutdownTimeout caps how long we wait for rdb.Close() to flush
	// pending commands before the process exits.
	ShutdownTimeout = 5 * time.Second
)

// NewClient constructs a [*redis.Client] from the supplied configuration and
// registers Fx lifecycle hooks for startup verification and graceful shutdown.
//
// The client is built via [redis.ParseURL] using [config.Config.RedisURL] —
// the single Redis environment variable (Doppler-style). Any timeouts,
// pool size, or TLS settings embedded in the URL's query string are honored.
//
// OpenTelemetry tracing is attached via [redisotel.InstrumentTracing] which
// uses the global otel.TracerProvider set up by the observability package, so
// every subsequent Redis command emits a span automatically.
//
// Startup (OnStart):
//
//   - Attempts a PING to verify Redis is reachable. On failure, the hook
//     returns an error which halts Fx startup — the process exits fast rather
//     than running with an unavailable cache.
//
//   - On success, logs address, DB index, pool size, and OTel status.
//
// Shutdown (OnStop):
//
//   - Calls rdb.Close() to drain the connection pool. Any in-flight commands
//     are given up to ShutdownTimeout to complete before the socket is torn
//     down.
//
// The constructor intentionally returns (T, error) — required by the backend
// AGENTS.md DI contract — so any startup failure (bad URL, unreachable host,
// auth failure) is propagated to Fx and aborts startup.
func NewClient(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis.NewClient: config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("redis.NewClient: logger is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis.NewClient: REDIS_URL is required")
	}

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("redis.NewClient: parse REDIS_URL: %w", err)
	}

	// Apply defaults for any unset timeouts / pool size. ParseURL only
	// populates fields explicitly set in the URL query string, so we
	// layer our defaults on top.
	if opts.DialTimeout == 0 {
		opts.DialTimeout = DefaultDialTimeout
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = DefaultReadTimeout
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = DefaultWriteTimeout
	}
	if opts.PoolSize == 0 {
		opts.PoolSize = DefaultPoolSize
	}

	client := redis.NewClient(opts)

	// Attach OpenTelemetry tracing instrumentation. InstrumentTracing uses
	// the global otel.TracerProvider set by observability.NewTracerProvider,
	// so it must run after the TracerProvider is wired (which it is, via Fx
	// ordering: observability.NewTracerProvider is provided before redis.Module
	// is invoked because Fx builds the dependency graph in order).
	if err := redisotel.InstrumentTracing(client); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis.NewClient: attach otel tracing: %w", err)
	}
	// Best-effort metrics — failure here is non-fatal because the cache is
	// still usable without metric export.
	if err := redisotel.InstrumentMetrics(client); err != nil {
		logger.Warn("redis: otel metrics instrumentation failed",
			zap.Error(err),
		)
	}

	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			pingCtx, cancel := context.WithTimeout(startCtx, DefaultPingTimeout)
			defer cancel()

			if err := client.Ping(pingCtx).Err(); err != nil {
				logger.Error("redis: initial ping failed",
					zap.String("addr", opts.Addr),
					zap.Int("db", opts.DB),
					zap.Error(err),
				)
				return fmt.Errorf("redis.NewClient: initial ping: %w", err)
			}

			logger.Info("redis: connected",
				zap.String("addr", opts.Addr),
				zap.Int("db", opts.DB),
				zap.Int("pool_size", opts.PoolSize),
				zap.Bool("otel_tracing", true),
			)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			logger.Info("redis: closing connection pool")

			// rdb.Close() is synchronous and does not block on stopCtx
			// itself, but the shutdown timeout protects against hung
			// connections (e.g. a server that stopped acking) by
			// running the close in a bounded goroutine.
			done := make(chan error, 1)
			go func() {
				done <- client.Close()
			}()

			select {
			case err := <-done:
				if err != nil {
					logger.Warn("redis: close error",
						zap.Error(err),
					)
					return fmt.Errorf("redis.NewClient: close: %w", err)
				}
				logger.Info("redis: connection pool closed cleanly")
				return nil
			case <-time.After(ShutdownTimeout):
				logger.Warn("redis: close timed out",
					zap.Duration("timeout", ShutdownTimeout),
				)
				return fmt.Errorf("redis.NewClient: close exceeded %s timeout", ShutdownTimeout)
			case <-stopCtx.Done():
				return fmt.Errorf("redis.NewClient: stop context cancelled during close: %w", stopCtx.Err())
			}
		},
	})

	return client, nil
}

// Ping verifies that the Redis client can reach the server and process a
// command within the configured timeout. It is intended for readiness probes
// and health check endpoints.
//
// Any error is wrapped with context so the HTTP layer can decide between
// 503 (transient connectivity issue) and 500 (unexpected).
func Ping(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return fmt.Errorf("redis.Ping: client is nil")
	}
	if ctx == nil {
		return fmt.Errorf("redis.Ping: context is nil")
	}

	pingCtx, cancel := context.WithTimeout(ctx, DefaultPingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("redis.Ping: %w", err)
	}
	return nil
}

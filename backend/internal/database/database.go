package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"somotracker/backend/internal/config"
)

// Pools holds references to the PostgreSQL connection pool and Redis client.
type Pools struct {
	PG    *pgxpool.Pool
	Redis *redis.Client
}

// Connect establishes connections to PostgreSQL and Redis using the provided config.
func Connect(cfg config.Config) (*Pools, error) {
	pgCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	pgCfg.MaxConns = 25
	pgCfg.MinConns = 5
	pgCfg.MaxConnLifetime = 30 * time.Minute
	pgCfg.MaxConnIdleTime = 5 * time.Minute

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	// Parse the full connection string directly
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("redis connection: %w", err)
	}

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		if closeErr := rdb.Close(); closeErr != nil {
			slog.WarnContext(ctx, "database.NewPools: failed to close redis after ping failure",
				slog.String("error", closeErr.Error()),
			)
		}
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Pools{PG: pool, Redis: rdb}, nil
}

// Module is an fx-compatible provider for *Pools.
var Module = fx.Provide(Connect)

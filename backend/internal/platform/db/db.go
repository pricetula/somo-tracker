package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Pool wraps pgxpool.Pool.
type Pool struct {
	*pgxpool.Pool
	// StdlibDB is a *sql.DB wrapper for sqlc compatibility.
	StdlibDB *sql.DB
}

// New creates a database pool from a DSN.
func New(ctx context.Context, dsn string) (*Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	// Use CacheDescribe instead of the default CacheStatement so pgx does not
	// cache named prepared statements on server-side connections. This avoids
	// "prepared statement name is already in use (08P01)" when the same
	// physical connection is reused across transactions (e.g. goose migrations
	// or connection pooling / PgBouncer on Render).
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Create stdlib wrapper for sqlc compatibility
	stdlibDB := stdlib.OpenDB(*pool.Config().ConnConfig)

	return &Pool{Pool: pool, StdlibDB: stdlibDB}, nil
}

// Close releases the pool.
func (p *Pool) Close() {
	_ = p.StdlibDB.Close()
	p.Pool.Close()
}

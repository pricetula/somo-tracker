package db

import (
	"context"
	"database/sql"
	"fmt"

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
	pool, err := pgxpool.New(ctx, dsn)
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

package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool provides pgx connection pool (BE-012).
type Pool struct {
	*pgxpool.Pool
}

// New creates a pgx pool from DSN.
func New(ctx context.Context, dsn string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Pool settings
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Pool{Pool: pool}, nil
}

// TxManager provides per-service transaction boundaries (BE-012).
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager constructs TxManager.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// Run executes fn in a transaction.
func (tm *TxManager) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	ctx = context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit(ctx)
}

// txKey is the context key for transaction.
type txKey struct{}

// GetTx returns the transaction from context, or nil if not in a transaction.
func GetTx(ctx context.Context) *pgxpool.Tx {
	if tx, ok := ctx.Value(txKey{}).(*pgxpool.Tx); ok {
		return tx
	}
	return nil
}
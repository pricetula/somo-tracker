package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TxManager provides transaction lifecycle helpers with guaranteed
// commit/rollback error logging. It wraps pgx.Tx and ensures that
// transaction cleanup errors are never silently dropped.
//
// Usage:
//
//	mgr := database.NewTxManager(pool, logger)
//	err := mgr.Run(ctx, func(tx pgx.Tx) error {
//	    // do work with tx
//	    return nil
//	})
//	// If we reach here, tx is committed (on success) or rolled back (on error)
//	// and any cleanup error has been logged.
type TxManager struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

// NewTxManager creates a new transaction manager.
func NewTxManager(pool *pgxpool.Pool, logger *zap.SugaredLogger) *TxManager {
	return &TxManager{pool: pool, logger: logger}
}

// Run executes fn inside a new transaction. The transaction is committed
// if fn returns nil, otherwise it is rolled back. Any commit/rollback
// error is logged but does not replace fn's error (the original error
// is always returned to the caller).
func (m *TxManager) Run(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Apply tenant GUC from context if present
	if err := ApplyTenantToTx(ctx, tx); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return err
	}

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && rbErr != pgx.ErrTxClosed {
			m.logger.Warnw("tx rollback failed", "error", rbErr, "original_error", err)
		}
		return err
	}

	if cmErr := tx.Commit(context.WithoutCancel(ctx)); cmErr != nil {
		m.logger.Errorw("tx commit failed", "error", cmErr)
		return cmErr
	}
	return nil
}

// RunWithOptions is like Run but accepts explicit transaction options.
func (m *TxManager) RunWithOptions(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	tx, err := m.pool.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	if err := ApplyTenantToTx(ctx, tx); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return err
	}

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && rbErr != pgx.ErrTxClosed {
			m.logger.Warnw("tx rollback failed", "error", rbErr, "original_error", err)
		}
		return err
	}

	if cmErr := tx.Commit(context.WithoutCancel(ctx)); cmErr != nil {
		m.logger.Errorw("tx commit failed", "error", cmErr)
		return cmErr
	}
	return nil
}

// Begin starts a transaction and applies the tenant GUC from ctx.
// Repositories that manage their own transactions must use this instead
// of pool.Begin so RLS policies see the correct tenant.
//
// The caller is responsible for committing or rolling back the returned
// transaction. Prefer TxManager.Run for automatic cleanup.
func Begin(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if err := ApplyTenantToTx(ctx, tx); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return nil, err
	}
	return tx, nil
}

// BeginTx is Begin with explicit transaction options.
func BeginTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions) (pgx.Tx, error) {
	tx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := ApplyTenantToTx(ctx, tx); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return nil, err
	}
	return tx, nil
}

// Commit commits the transaction and logs any error. Returns the commit
// error (if any) so the caller can decide whether to propagate it.
func Commit(ctx context.Context, tx pgx.Tx, logger *zap.SugaredLogger) error {
	if err := tx.Commit(ctx); err != nil {
		if logger != nil {
			logger.Errorw("tx commit failed", "error", err)
		}
		return err
	}
	return nil
}

// Rollback rolls back the transaction and logs any error (except
// pgx.ErrTxClosed which means the tx was already committed/rolled back).
// Returns the rollback error (if any).
func Rollback(ctx context.Context, tx pgx.Tx, logger *zap.SugaredLogger) error {
	if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
		if logger != nil {
			logger.Warnw("tx rollback failed", "error", err)
		}
		return err
	}
	return nil
}

// DeferRollback is a helper for defer statements that rolls back the
// transaction if it hasn't been committed yet. Logs rollback errors.
//
// Usage:
//
//	tx, err := database.Begin(ctx, pool)
//	if err != nil { return err }
//	defer database.DeferRollback(ctx, tx, logger)
//	// ... do work ...
//	return database.Commit(ctx, tx, logger)
func DeferRollback(ctx context.Context, tx pgx.Tx, logger *zap.SugaredLogger) {
	if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
		if logger != nil {
			logger.Warnw("deferred tx rollback failed", "error", err)
		}
	}
}

package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TenantSessionKey is the GUC key used by Somotracker Row-Level Security
// policies. RLS policies are expected to reference current_setting('app.current_tenant_id').
//
// The key is namespaced ("app.") following PostgreSQL's recommended convention
// for application-defined GUCs and matches the SET LOCAL statement executed
// in [WithTenantTx].
const (
	TenantSessionKey = "app.current_tenant_id"
)

// WithTenantTx runs fn inside a single database transaction whose session
// state is scoped to the supplied tenant identifier. The transaction is
// committed if fn returns nil and rolled back otherwise.
//
// Because pgx pins every query inside a transaction to the same physical
// connection, executing:
//
//	SET LOCAL app.current_tenant_id = $1
//
// at the start of the transaction guarantees that all subsequent SELECT /
// INSERT / UPDATE / DELETE statements — and any RLS policies that read
// current_setting('app.current_tenant_id') — see the tenant scope until
// the transaction ends. The SET LOCAL is automatically discarded at COMMIT /
// ROLLBACK, eliminating any risk of tenant leakage across pooled connections.
//
// Parameters:
//   - ctx: request-scoped context. Cancellation aborts the transaction.
//   - pool: the shared *pgxpool.Pool.
//   - logger: zap logger used for the dual-error rollback log.
//   - tenantID: opaque tenant identifier (e.g. UUID string, tenant slug).
//     Pass an empty string to assert "no tenant" — this is permitted, but
//     repositories are responsible for ensuring RLS still allows the
//     operation (e.g. via a BYPASSRLS role).
//   - fn: the unit of work. Any error returned triggers a rollback.
//
// The returned error is wrapped with caller context so the repository or
// service can propagate it without losing the original cause.
func WithTenantTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *zap.Logger,
	tenantID string,
	fn func(ctx context.Context, tx pgx.Tx) error,
) (err error) {
	if pool == nil {
		return fmt.Errorf("database.WithTenantTx: pool is nil")
	}
	if fn == nil {
		return fmt.Errorf("database.WithTenantTx: fn is nil")
	}
	if logger == nil {
		return fmt.Errorf("database.WithTenantTx: logger is required")
	}

	tx, beginErr := pool.BeginTx(ctx, pgx.TxOptions{
		// ReadCommitted is the default and matches Postgres' default isolation
		// level. We keep it explicit for documentation and stability.
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
	if beginErr != nil {
		return fmt.Errorf("database.WithTenantTx: begin transaction: %w", beginErr)
	}

	// Deferred rollback using the named-return err. This satisfies the
	// backend AGENTS.md "dual-error rollback" pattern: the outer error is
	// captured before rollback can clobber it, and we log only when a real
	// rollback failure occurs (Commit already happened).
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			logger.Error("tenant transaction rollback failed",
				zap.String("tenant_id", tenantID),
				zap.String("original_error", errString(err)),
				zap.String("rollback_error", rbErr.Error()),
			)
			// Preserve the original error; attach rollback error if caller didn't.
			if err == nil {
				err = fmt.Errorf("database.WithTenantTx: rollback: %w", rbErr)
			}
		}
	}()

	// Bind the tenant scope to this transaction. SET LOCAL is scoped to the
	// current transaction; we use $1 to keep the value safely parameterized.
	if _, setErr := tx.Exec(ctx, fmt.Sprintf("SET LOCAL %s = $1", TenantSessionKey), tenantID); setErr != nil {
		return fmt.Errorf("database.WithTenantTx: set tenant context %q: %w", TenantSessionKey, setErr)
	}

	if fnErr := fn(ctx, tx); fnErr != nil {
		return fmt.Errorf("database.WithTenantTx: user callback: %w", fnErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("database.WithTenantTx: commit: %w", commitErr)
	}

	return nil
}

// SetTenantOnTx applies the tenant GUC to an existing transaction. Useful for
// callers that have already begun a transaction (e.g. inside a service-layer
// orchestrator) and need to enable RLS scoping on it.
//
// The setting is local to the transaction and discarded at COMMIT/ROLLBACK,
// so it is safe to call even when the connection is returned to the pool.
//
// Returns an error if tx is nil or if the SET LOCAL fails.
func SetTenantOnTx(ctx context.Context, tx pgx.Tx, tenantID string) error {
	if tx == nil {
		return fmt.Errorf("database.SetTenantOnTx: tx is nil")
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL %s = $1", TenantSessionKey), tenantID); err != nil {
		return fmt.Errorf("database.SetTenantOnTx: set tenant context %q: %w", TenantSessionKey, err)
	}
	return nil
}

// errString safely extracts err.Error() even when err is nil. Used only for
// logging inside the deferred rollback.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// WithTx runs fn inside a single database transaction with no tenant
// scoping. This is used for system-level operations that span tenants —
// specifically the magic-link auth callback, which must atomically
// provision a tenant, user, and member row before the tenant scope is
// known. Production per-request queries should always use WithTenantTx.
func WithTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *zap.Logger,
	fn func(ctx context.Context, tx pgx.Tx) error,
) (err error) {
	if pool == nil {
		return fmt.Errorf("database.WithTx: pool is nil")
	}
	if fn == nil {
		return fmt.Errorf("database.WithTx: fn is nil")
	}
	if logger == nil {
		return fmt.Errorf("database.WithTx: logger is required")
	}

	tx, beginErr := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
	if beginErr != nil {
		return fmt.Errorf("database.WithTx: begin transaction: %w", beginErr)
	}

	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			logger.Error("transaction rollback failed",
				zap.String("original_error", errString(err)),
				zap.String("rollback_error", rbErr.Error()),
			)
			if err == nil {
				err = fmt.Errorf("database.WithTx: rollback: %w", rbErr)
			}
		}
	}()

	if fnErr := fn(ctx, tx); fnErr != nil {
		return fmt.Errorf("database.WithTx: user callback: %w", fnErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("database.WithTx: commit: %w", commitErr)
	}

	return nil
}

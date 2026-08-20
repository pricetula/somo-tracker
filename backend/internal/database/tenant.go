package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Context keys shared with the HTTP middleware. The middleware stores values
// in Fiber locals, which are visible through the fasthttp RequestCtx's
// Value() — the same context.Context that handlers pass to repositories.
// Background workers and services that manage their own transactions set the
// same keys via WithTenantTx / WithTenantID.
// Using custom types avoids collisions (SA1029).
type (
	tenantIDKey string
	tenantTxKey string
)

const (
	TenantIDKey tenantIDKey = "tenant_id"
	TenantTxKey tenantTxKey = "tenant_tx"
)

// Executor is the subset of query operations shared by *pgxpool.Pool and
// pgx.Tx, so repository code can transparently run against the request's
// tenant-scoped transaction (when one exists) or fall back to the shared
// pool otherwise.
type Executor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Executor returns the request-scoped tenant transaction when the context
// carries one; otherwise it returns the provided fallback (usually the pool).
// This is the single choke point that makes RLS tenant context work: every
// repository query should route through it.
func FromContext(ctx context.Context, fallback Executor) Executor {
	if tx, ok := ctx.Value(TenantTxKey).(pgx.Tx); ok {
		return tx
	}
	return fallback
}

// TenantID returns the tenant ID carried by the context, if any.
func TenantID(ctx context.Context) (string, bool) {
	s, _ := ctx.Value(TenantIDKey).(string)
	return s, s != ""
}

// WithTenantID returns a context carrying the tenant ID. Used by background
// workers and services that know the tenant but are not inside an HTTP
// request with a resolved session.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// WithTenantTx returns a context carrying the given transaction as the
// tenant-scoped executor. Used by background workers and services that
// manage their own transactions.
func WithTenantTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TenantTxKey, tx)
}

// ApplyTenantToTx sets app.current_tenant_id (transaction-scoped) on a
// freshly started transaction so RLS policies evaluate for the right tenant.
func ApplyTenantToTx(ctx context.Context, tx pgx.Tx) error {
	tid, ok := TenantID(ctx)
	if !ok {
		return nil
	}
	_, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tid)
	return err
}

// Begin starts a transaction and applies the tenant GUC from ctx. Repositories
// that manage their own transactions must use this instead of pool.Begin so
// RLS policies see the correct tenant.
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

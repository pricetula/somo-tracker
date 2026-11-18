package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"somotracker/internal/platform/httpserver"
)

// Pool provides a sql.DB connection pool (BE-012).
type Pool struct {
	*sql.DB
}

// New creates a pgx connection pool from DSN using the database/sql interface.
// This is required because sqlc-generated code uses database/sql DBTX.
func New(ctx context.Context, dsn string) (*Pool, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// Pool settings
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Pool{db}, nil
}

// TxManager provides per-service transaction boundaries (BE-012).
type TxManager struct {
	db *sql.DB
}

// NewTxManager constructs TxManager.
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// Run executes fn in a transaction. If a transaction already exists in the
// context, fn runs within it without nesting (database/sql does not support
// nested transactions). The tenant GUC is set via SET LOCAL inside the
// transaction so RLS policies see the correct tenant for the duration of the tx.
func (tm *TxManager) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	// Don't nest transactions: if one is already in context, run within it.
	if GetTx(ctx) != nil {
		return fn(ctx)
	}

	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Set tenant GUC for RLS within the transaction (SET LOCAL is transaction-scoped).
	if err := SetTenantGUC(ctx, func(ctx context.Context, sql string, args ...any) (any, error) {
		return tx.ExecContext(ctx, sql, args...)
	}); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("set tenant context: %w", err)
	}

	ctx = context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

// ExecFunc is the function signature for executing SQL, satisfied by both
// *sql.Tx.ExecContext and *sql.DB.ExecContext (same signature).
type ExecFunc func(ctx context.Context, sql string, arguments ...any) (any, error)

// SetTenantGUC sets app.current_tenant_id via SET LOCAL on the given ExecFunc.
// The tenant ID is read from request context (set by TenantMiddleware). The
// value is validated as a UUID before interpolation to avoid SQL injection.
// SET LOCAL does not support query parameters, so strict UUID validation is
// the only safe way to interpolate.
func SetTenantGUC(ctx context.Context, exec ExecFunc) error {
	tenantID := httpserver.TenantIDFrom(ctx)
	if tenantID == "" {
		return nil
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}
	setSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
	_, err := exec(ctx, setSQL)
	return err
}

// txKey is the context key for transaction.
type txKey struct{}

// GetTx returns the transaction from context, or nil if not in a transaction.
func GetTx(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}

// Package testdb is the integration-test helper for the Somotracker backend.
//
// It connects to the shared long-lived Postgres instance started outside of
// Go (via docker-compose.test.yml + make test-db-up) and exposes:
//
//  1. [DB] — a lazily-initialized process-wide *sql.DB against the test
//     instance, used by migrations and by tests that want full access to
//     the connection (DCL, schema inspection, etc.).
//  2. [BeginTx] — opens a per-test *sql.Tx and registers a t.Cleanup hook
//     that rolls the transaction back. This is the pattern every integration
//     test should use: each test sees a stable schema but never persists
//     data, eliminating the need to truncate tables between runs.
//
// Application code that talks to the database must accept either *sql.DB or
// *sql.Tx through the [DBTX] interface declared here, rather than opening
// its own connection. Hardcoded global pools would bypass the rollback
// pattern and leak test data across cases. See DBTX for the contract.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver for database/sql
)

// DefaultDSN is the DSN used when neither TEST_DATABASE_URL is set nor an
// explicit URL is passed. It matches docker-compose.test.yml.
const DefaultDSN = "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"

// DBTX is the union of methods that both *sql.DB and *sql.Tx satisfy. Any
// repository/service method that needs to talk to the database should accept
// a DBTX so it can be called with either the production connection pool or
// the per-test transaction.
//
// Example:
//
//	type Repository struct{}
//
//	func (r *Repository) FindUser(ctx context.Context, db testdb.DBTX, id string) (*User, error) {
//	    var u User
//	    if err := db.QueryRowContext(ctx, "SELECT ...").Scan(&u.ID, &u.Name); err != nil {
//	        return nil, fmt.Errorf("repositories.Repository.FindUser: %w", err)
//	    }
//	    return &u, nil
//	}
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var (
	sharedDB *sql.DB
	dbOnce   sync.Once
	dbErr    error
)

// DB returns the process-wide *sql.DB connected to the shared test instance.
// It is lazily initialized on first call and reused for the lifetime of the
// `go test` invocation. Concurrent calls are safe.
//
// The connection pool is sized for parallel test execution (MaxOpenConns=20).
// Each test acquires its own connection via BeginTx and releases it back on
// rollback.
func DB(t *testing.T) *sql.DB {
	t.Helper()
	dbOnce.Do(func() {
		dsn := os.Getenv("TEST_DATABASE_URL")
		if dsn == "" {
			dsn = DefaultDSN
		}

		sharedDB, dbErr = sql.Open("pgx", dsn)
		if dbErr != nil {
			return
		}

		sharedDB.SetMaxOpenConns(20)
		sharedDB.SetMaxIdleConns(5)
		sharedDB.SetConnMaxLifetime(30 * time.Minute)
		sharedDB.SetConnMaxIdleTime(5 * time.Minute)

		// Probe the connection so the test fails fast (with a clear message)
		// if the test DB isn't running.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		dbErr = sharedDB.PingContext(ctx)
	})
	if dbErr != nil {
		t.Fatalf("testdb.DB: failed to open test database connection: %v\n"+
			"  hint: run `make test-db-up` and ensure TEST_DATABASE_URL is set", dbErr)
	}
	return sharedDB
}

// Close releases the shared *sql.DB. It is normally not called by tests —
// `go test` exits and lets the OS reclaim resources — but it is exported for
// test binaries that want to swap databases mid-run.
func Close() error {
	if sharedDB == nil {
		return nil
	}
	return sharedDB.Close()
}

// BeginTx opens a *sql.Tx from the shared pool and registers a t.Cleanup
// hook that rolls the transaction back when the test ends — whether it
// passed, failed, or panicked. The transaction is named so PostgreSQL shows
// it in pg_locks, which makes stuck tests easier to debug.
//
// Each call opens a fresh transaction bound to a single connection. Tests
// that call t.Parallel() each get their own transaction, so writes from one
// test never become visible to another.
//
// The returned *sql.Tx is suitable for passing directly to repository code
// that accepts a [DBTX].
//
// Example:
//
//	func TestCreateUser(t *testing.T) {
//	    t.Parallel()
//	    tx := testdb.BeginTx(t)
//
//	    repo := users.NewRepository()
//	    user, err := repo.CreateUser(context.Background(), tx, "ada@example.com")
//	    if err != nil {
//	        t.Fatalf("create failed: %v", err)
//	    }
//	    if user.ID == "" {
//	        t.Error("expected user ID to be populated")
//	    }
//	    // No teardown needed: the tx is rolled back via t.Cleanup.
//	}
func BeginTx(t *testing.T) *sql.Tx {
	t.Helper()
	db := DB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	name := fmt.Sprintf("test:%s", t.Name())
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		t.Fatalf("testdb.BeginTx: begin transaction: %v", err)
	}

	// Tag the transaction with a comment so PostgreSQL surfaces it in
	// pg_stat_activity / pg_locks with the test's name. This is invaluable
	// when a test hangs and you need to identify which lock to kill.
	// Use SET APPLICATION_NAME so PostgreSQL shows the transaction name.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET application_name = %s", quoteString(name))); err != nil {
		t.Logf("testdb.BeginTx: could not name transaction: %v", err)
	}

	t.Cleanup(func() {
		// Always roll back. We use a fresh context because t.Context() may
		// already be cancelled by the time Cleanup runs.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			t.Logf("testdb.BeginTx: rollback failed for %q: %v", name, rbErr)
		}
		_ = ctx // currently unused; reserved for future driver context support
	})

	return tx
}

// MustExec is a convenience helper that runs a query against the test
// database and fails the test on error. Use it for setup outside the
// per-test transaction (e.g. creating extensions that should persist across
// the whole test run).
//
// Queries executed here are NOT rolled back; they apply to the shared test
// instance. Use [BeginTx] for test data.
func MustExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := DB(t).ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("testdb.MustExec: %s: %v", query, err)
	}
}

// quoteString safely quotes a Go string for inclusion in SQL.
func quoteString(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '\'')
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '\'')
		} else {
			out = append(out, s[i])
		}
	}
	out = append(out, '\'')
	return string(out)
}

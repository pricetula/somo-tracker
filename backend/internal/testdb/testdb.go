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

// Close releases the shared *sql.DB.
func Close() error {
	if sharedDB == nil {
		return nil
	}
	return sharedDB.Close()
}

// BeginTx opens a *sql.Tx from the shared pool and registers a t.Cleanup
// hook that rolls the transaction back when the test ends.
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

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET application_name = %s", quoteString(name))); err != nil {
		t.Logf("testdb.BeginTx: could not name transaction: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			t.Logf("testdb.BeginTx: rollback failed for %q: %v", name, rbErr)
		}
		_ = ctx
	})

	return tx
}

// MustExec is a convenience helper that runs a query against the test
// database and fails the test on error.
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

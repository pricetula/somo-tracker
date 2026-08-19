// Package testhelper provides shared utilities for database integration tests.
// All repository-level integration tests should use this package to start
// a PostgreSQL container via testcontainers.
package testhelper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPG starts a PostgreSQL 16 Alpine testcontainer and returns the pool
// plus a cleanup function. The caller must call cleanup when done.
func StartPG(t testing.TB) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       "somotracker_test",
			"POSTGRES_USER":     "somo_admin",
			"POSTGRES_PASSWORD": "somo_secure_password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("testhelper.StartPG: start container: %v", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("testhelper.StartPG: get host: %v", err)
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("testhelper.StartPG: get mapped port: %v", err)
	}

	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s:%s/somotracker_test?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("testhelper.StartPG: connect: %v", err)
	}

	cleanup := func() {
		pool.Close()
		if err := c.Terminate(ctx); err != nil {
			t.Logf("testhelper.StartPG: terminate container: %v", err)
		}
	}

	return pool, cleanup
}

// ApplyMigration reads and executes a SQL migration file.
func ApplyMigration(t testing.TB, pool *pgxpool.Pool, migrationSQL string) {
	t.Helper()

	_, err := pool.Exec(context.Background(), migrationSQL)
	if err != nil {
		t.Fatalf("testhelper.ApplyMigration: %v", err)
	}
}

// SeededPool starts PG, applies all migrations (schema + seed), and returns
// the pool with cleanup. Useful for tests that need a full schema.
func SeededPool(t testing.TB) (*pgxpool.Pool, func()) {
	t.Helper()

	pool, cleanup := StartPG(t)

	// Apply all schema migrations + seed in sorted order.
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "migrations")
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		cleanup()
		t.Fatalf("testhelper.SeededPool: glob migrations: %v", err)
	}
	for _, path := range files {
		sql, err := os.ReadFile(path)
		if err != nil {
			cleanup()
			t.Fatalf("testhelper.SeededPool: read migration %s: %v", path, err)
		}
		ApplyMigration(t, pool, string(sql))
	}

	return pool, cleanup
}

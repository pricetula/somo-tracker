package testcontainers

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer holds a testcontainers PostgreSQL instance
type PostgresContainer struct {
	Container tc.Container
	pool      *pgxpool.Pool
	dsn       string
}

// SetupPostgres starts a PostgreSQL container using testcontainers-go
// and returns a PostgresContainer with a connected pool.
// The container is automatically terminated when the test completes (via ryuk).
// Note: Migrations are NOT run automatically. Call RunMigrations(pool) in your test.
func SetupPostgres(t *testing.T) *PostgresContainer {
	ctx := context.Background()

	// Use the postgres module for convenient setup
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
		tc.WithTmpfs(map[string]string{
			"/var/lib/postgresql/data": "rw,size=256m",
		}),
	)
	require.NoError(t, err, "failed to start postgres container")

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	// Create pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "failed to create pgxpool")

	// Verify connection
	require.NoError(t, pool.Ping(ctx), "failed to ping database")

	pc := &PostgresContainer{
		Container: pgContainer,
		pool:      pool,
		dsn:       connStr,
	}

	// Cleanup on test completion
	// Note: testcontainers-go uses ryuk for automatic container cleanup,
	// so we only close the pool here. The container will be terminated by ryuk.
	t.Cleanup(func() {
		if pc.pool != nil {
			pc.pool.Close()
		}
	})

	return pc
}

// SetupPostgresWithCustomConfig starts a PostgreSQL container with custom
// configuration options. Useful for tests that need specific postgres settings.
// Note: Migrations are NOT run automatically. Call RunMigrations(pool) in your test.
func SetupPostgresWithCustomConfig(t *testing.T, opts ...tc.ContainerCustomizer) *PostgresContainer {
	ctx := context.Background()

	// Default options
	defaultOpts := []tc.ContainerCustomizer{
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
		),
		tc.WithTmpfs(map[string]string{
			"/var/lib/postgresql/data": "rw,size=256m",
		}),
	}

	allOpts := append(defaultOpts, opts...)

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine", allOpts...)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "failed to create pgxpool")

	require.NoError(t, pool.Ping(ctx), "failed to ping database")

	pc := &PostgresContainer{
		Container: pgContainer,
		pool:      pool,
		dsn:       connStr,
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		if pc.pool != nil {
			pc.pool.Close()
		}
	})

	return pc
}

// Pool returns the pgxpool for direct database access
func (pc *PostgresContainer) Pool() *pgxpool.Pool {
	return pc.pool
}

// DSN returns the connection string for the container
func (pc *PostgresContainer) DSN() string {
	return pc.dsn
}

// Terminate manually terminates the container (usually not needed due to ryuk).
// Use this if you need to terminate before the test ends.
func (pc *PostgresContainer) Terminate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pc.pool != nil {
		pc.pool.Close()
	}
	if pc.Container != nil {
		if err := pc.Container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}
}

// SkipIfNoDocker marks the test as skipped if Docker is not available.
// Use this for tests that require testcontainers but should not fail CI
// when Docker is unavailable.
func SkipIfNoDocker(t *testing.T) {
	_, err := tc.NewDockerClientWithOpts(context.Background())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
}

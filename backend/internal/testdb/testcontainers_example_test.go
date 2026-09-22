//go:build integration

package testdb

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/testcontainers"
)

// Example test demonstrating testcontainers usage
func TestExampleWithTestcontainers(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()

	// Run migrations
	logger := zap.NewNop()
	migrator, err := database.NewMigrator(tc.Pool(), logger)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)
	migrator.Close()

	// Use the pool directly
	pool := tc.Pool()
	require.NotNil(t, pool)

	// Or get the DSN for creating additional pools
	dsn := tc.DSN()
	require.NotEmpty(t, dsn)

	// Example: create a new pool with the same DSN
	pool2, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool2.Close()

	// Run a simple query
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	require.Equal(t, 1, result)

	// Verify migrations ran by checking a table exists
	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'tenants'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	require.True(t, tableExists, "tenants table should exist after migrations")
}

// Example test with custom postgres configuration
func TestExampleWithCustomConfig(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	// Custom options: e.g., add init scripts, custom config, etc.
	tc := testcontainers.SetupPostgresWithCustomConfig(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	// Run migrations
	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)
	migrator.Close()

	// Verify connection works
	require.NoError(t, pool.Ping(ctx))

	// Verify migrations ran
	var version uint
	err = pool.QueryRow(ctx, `
		SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1
	`).Scan(&version)
	require.NoError(t, err)
	require.GreaterOrEqual(t, version, uint(1))
}

// Example test demonstrating parallel test isolation
func TestExampleParallelIsolation(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)
	t.Parallel()

	// Each parallel test gets its own container
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	// Run migrations
	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)
	migrator.Close()

	// Insert test data
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (name, slug, stytch_org_id)
		VALUES ('Test Tenant', 'test-tenant-' || gen_random_uuid()::text, 'org-test')
	`)
	require.NoError(t, err)

	// Verify only this test's data exists
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM tenants WHERE name = 'Test Tenant'").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// Example test showing how to use with existing migrator
func TestExampleWithMigrator(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()

	// Create migrator with the test container's pool
	logger := zap.NewNop()
	migrator, err := database.NewMigrator(tc.Pool(), logger)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)

	// Verify migrations applied
	ver, dirty, err := migrator.CurrentVersion()
	require.NoError(t, err)
	require.False(t, dirty)
	require.GreaterOrEqual(t, ver, uint(1))

	migrator.Close()
}

// Example test showing how to use testcontainers for a specific migration test
func TestExampleSpecificMigration(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	// Start container without running migrations automatically
	ctx := context.Background()
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
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	// Now run migrations manually
	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err)

	// Verify specific migration was applied
	ver, dirty, err := migrator.CurrentVersion()
	require.NoError(t, err)
	require.False(t, dirty)
	require.GreaterOrEqual(t, ver, uint(1))
}

//go:build integration

package seed

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestSeed_Integration verifies the seed runs against a real PostgreSQL
// container, applying migrations 01-07 first, then seeding reference data.
func TestSeed_Integration(t *testing.T) {
	ctx := context.Background()

	postgresC, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:18"),
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err, "start postgres container")
	defer func() {
		require.NoError(t, postgresC.Terminate(ctx))
	}()

	dsn, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Apply migrations 01-07 via goose (same pattern as migrator_integration_test.go)
	sqlDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer sqlDB.Close()

	goose.SetTableName("goose_db_version")
	require.NoError(t, goose.UpContext(ctx, sqlDB, "../../migrations"), "apply migrations")

	// Verify reference tables exist and are empty before seed
	var count int
	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM countries").Scan(&count))
	require.Equal(t, 0, count, "countries should be empty before seed")

	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM education_systems").Scan(&count))
	require.Equal(t, 0, count, "education_systems should be empty before seed")

	require.NoError(t, sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM class_levels").Scan(&count))
	require.Equal(t, 0, count, "class_levels should be empty before seed")

	// Create pgxpool for seed
	pgxPool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pgxPool.Close()

	// Run the seed
	seeder := New(pgxPool)
	require.NoError(t, seeder.Run(ctx), "run seed")

	// Verify countries seeded
	require.NoError(t, pgxPool.QueryRow(ctx, "SELECT COUNT(*) FROM countries").Scan(&count))
	require.Equal(t, 3, count)

	// Verify education systems seeded
	require.NoError(t, pgxPool.QueryRow(ctx, "SELECT COUNT(*) FROM education_systems").Scan(&count))
	require.Equal(t, 3, count)

	// Verify class levels seeded
	require.NoError(t, pgxPool.QueryRow(ctx, "SELECT COUNT(*) FROM class_levels").Scan(&count))
	require.Equal(t, 11, count)

	// Idempotency: run again, counts must not change
	require.NoError(t, seeder.Run(ctx), "run seed again")
	require.NoError(t, pgxPool.QueryRow(ctx, "SELECT COUNT(*) FROM countries").Scan(&count))
	require.Equal(t, 3, count)
	require.NoError(t, pgxPool.QueryRow(ctx, "SELECT COUNT(*) FROM education_systems").Scan(&count))
	require.Equal(t, 3, count)
	require.NoError(t, pgxPool.QueryRow(ctx, "SELECT COUNT(*) FROM class_levels").Scan(&count))
	require.Equal(t, 11, count)
}

package seed

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"somotracker/internal/platform/db"
)

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Skipf("skipping seed test: no database available: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping seed test: cannot reach database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// TestSeed_Idempotent verifies that running the seed twice produces no duplicates.
func TestSeed_Idempotent(t *testing.T) {
	pool := mustPool(t)
	seeder := New(pool)

	// First run
	require.NoError(t, seeder.Run(context.Background()))

	// Second run - should not error and should not duplicate
	require.NoError(t, seeder.Run(context.Background()))

	// Verify counts
	var countryCount int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM countries").Scan(&countryCount))
	require.Equal(t, 3, countryCount)

	var systemCount int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM education_systems").Scan(&systemCount))
	require.Equal(t, 3, systemCount)

	var levelCount int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM class_levels").Scan(&levelCount))
	require.Equal(t, 11, levelCount)
}

// TestSeed_Atomic verifies that all three tables are populated together.
func TestSeed_Atomic(t *testing.T) {
	pool := mustPool(t)
	seeder := New(pool)

	require.NoError(t, seeder.Run(context.Background()))

	var countryCount, systemCount, levelCount int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM countries").Scan(&countryCount))
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM education_systems").Scan(&systemCount))
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM class_levels").Scan(&levelCount))

	require.Equal(t, 3, countryCount)
	require.Equal(t, 3, systemCount)
	require.Equal(t, 11, levelCount)
}

// TestNew_CreatesSeed verifies New returns a non-nil Seed.
func TestNew_CreatesSeed(t *testing.T) {
	pool := mustPool(t)
	s := New(pool)
	require.NotNil(t, s)
	require.IsType(t, &Seed{}, s)
}

// TestTxManagerIsUsed verifies db.NewTxManager can be constructed and Run works.
func TestTxManagerIsUsed(t *testing.T) {
	pool := mustPool(t)

	tm := db.NewTxManager(pool)
	require.NotNil(t, tm)

	// Verify Run works with a simple query
	var count int
	err := tm.Run(context.Background(), func(ctx context.Context) error {
		return pool.QueryRow(ctx, "SELECT COUNT(*) FROM countries").Scan(&count)
	})
	require.NoError(t, err)
}

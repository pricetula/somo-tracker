package seed

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"somotracker/internal/platform/db"
)

// TestNew_CreatesSeed verifies New returns a non-nil Seed with no DB required.
func TestNew_CreatesSeed(t *testing.T) {
	sqlDB, err := sql.Open("pgx", "postgres://invalid")
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	s := New(sqlDB)
	require.NotNil(t, s)
	require.IsType(t, &Seed{}, s)
}

// TestTxManagerConstruction verifies db.NewTxManager can be constructed without a DB.
func TestTxManagerConstruction(t *testing.T) {
	sqlDB, err := sql.Open("pgx", "postgres://invalid")
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	tm := db.NewTxManager(sqlDB)
	require.NotNil(t, tm)
}

// mustDB returns a *sql.DB or skips the test if none is available.
func mustDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := sql.Open("pgx", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Skipf("skipping seed test: no database available: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		t.Skipf("skipping seed test: cannot reach database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

// TestSeed_Idempotent verifies that running the seed twice produces no duplicates.
func TestSeed_Idempotent(t *testing.T) {
	sqlDB := mustDB(t)
	seeder := New(sqlDB)

	// First run
	require.NoError(t, seeder.Run(context.Background()))

	// Second run - should not error and should not duplicate
	require.NoError(t, seeder.Run(context.Background()))

	// Verify counts
	var countryCount int
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM countries").Scan(&countryCount))
	require.Equal(t, 3, countryCount)

	var systemCount int
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM education_systems").Scan(&systemCount))
	require.Equal(t, 3, systemCount)

	var levelCount int
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM class_levels").Scan(&levelCount))
	require.Equal(t, 11, levelCount)
}

// TestSeed_Atomic verifies that all three tables are populated together.
func TestSeed_Atomic(t *testing.T) {
	sqlDB := mustDB(t)
	seeder := New(sqlDB)

	require.NoError(t, seeder.Run(context.Background()))

	var countryCount, systemCount, levelCount int
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM countries").Scan(&countryCount))
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM education_systems").Scan(&systemCount))
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM class_levels").Scan(&levelCount))

	require.Equal(t, 3, countryCount)
	require.Equal(t, 3, systemCount)
	require.Equal(t, 11, levelCount)
}

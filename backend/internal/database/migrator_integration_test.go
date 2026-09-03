//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testdb"
)

func TestMigrator_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	dsn := "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "pgxpool.New should not fail")
	t.Cleanup(pool.Close)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err, "NewMigrator should not fail")
	t.Cleanup(func() { _ = migrator.Close() })

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	ver, dirty, err := migrator.CurrentVersion()
	require.NoError(t, err, "CurrentVersion should not fail")
	require.False(t, dirty, "schema should not be dirty after a clean run")
	require.GreaterOrEqual(t, ver, uint(1), "at least migration 1 should be applied")
}

func TestMigrator_Extensions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Use the shared testdb pool for queries. The migrator owns the connection
	// for DDL; the shared pool connection is used only for reads.
	db := testdb.DB(t)

	dsn := "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = migrator.Close() })

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// Query pg_extension to assert the required extensions are present.
	// pg_uuidv7 is optional; uuid-ossp and pgcrypto are required.
	rows, err := db.QueryContext(ctx, `
		SELECT extname FROM pg_extension
		WHERE extname IN ('pgcrypto', 'uuid-ossp', 'pg_uuidv7')
		ORDER BY extname
	`)
	require.NoError(t, err, "querying pg_extension should not fail")
	defer rows.Close()

	found := make(map[string]bool)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		found[name] = true
	}
	require.NoError(t, rows.Err())

	require.True(t, found["pgcrypto"], "pgcrypto extension should be active")
	require.True(t, found["uuid-ossp"], "uuid-ossp extension should be active")
	// pg_uuidv7 is tested separately; it may or may not be present.
}

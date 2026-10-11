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

func TestMigrator_GenericBulkJobs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
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

	// Verify generic bulk job tables exist
	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'bulk_jobs'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "bulk_jobs table should exist for generic invitation processor")
}

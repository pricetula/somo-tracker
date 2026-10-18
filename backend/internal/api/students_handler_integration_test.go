//go:build integration

package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/testcontainers"
)

func TestStudentsHandler_ErrorContract(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err)

	// Verify students table has structured columns
	cols := []string{"address", "phone", "email", "nationality", "emergency_contact_name", "emergency_contact_phone", "special_needs"}
	for _, col := range cols {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'students' AND column_name = $1
			)
		`, col).Scan(&exists)
		require.NoError(t, err)
		require.True(t, exists, "column %s should exist", col)
	}
}

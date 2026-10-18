//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testcontainers"
)

// TestMigrator_StudentsRLS verifies RLS is enabled on students table and tenant isolation policy exists.
func TestMigrator_StudentsRLS(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- RLS is enabled on students ---
	var rlsEnabled, rlsForced bool
	err = pool.QueryRow(ctx, `
		SELECT relrowsecurity, relforcerowsecurity
		FROM pg_class
		WHERE relname = 'students'
		  AND relnamespace = 'public'::regnamespace
	`).Scan(&rlsEnabled, &rlsForced)
	require.NoError(t, err)
	require.True(t, rlsEnabled, "RLS should be enabled on students")
	require.True(t, rlsForced, "RLS should be FORCED on students")

	// --- RLS policy exists ---
	var policyName string
	err = pool.QueryRow(ctx, `
		SELECT policyname FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'students'
		  AND policyname = 'students_tenant_isolation'
	`).Scan(&policyName)
	require.NoError(t, err)
	require.Equal(t, "students_tenant_isolation", policyName, "students_tenant_isolation policy should be installed")

	var policyQual string
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(qual, '') FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'students'
		  AND policyname = 'students_tenant_isolation'
	`).Scan(&policyQual)
	require.NoError(t, err)
	require.Contains(t, policyQual, "current_setting")
	require.Contains(t, policyQual, "app.current_tenant_id")
	require.Contains(t, policyQual, "schools.tenant_id")

	// --- New structured columns exist ---
	cols := []string{
		"address", "phone", "email", "nationality",
		"emergency_contact_name", "emergency_contact_phone", "special_needs",
	}
	for _, col := range cols {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'students' AND column_name = $1
			)
		`, col).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "column %q should exist on students", col)
	}
}

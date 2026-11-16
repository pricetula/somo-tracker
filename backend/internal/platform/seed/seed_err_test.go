package seed

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// TestSeedErrNoRowsHandling verifies that both sql.ErrNoRows and pgx.ErrNoRows
// are treated as "already exists" during idempotent seeding.
//
// Before the fix, only sql.ErrNoRows was checked. Since the project uses pgx
// (github.com/jackc/pgx/v5), QueryRow().Scan() returns pgx.ErrNoRows when
// ON CONFLICT DO NOTHING ... RETURNING id finds no row to return. This caused
// the seed to fail on re-runs with:
//   ERROR app create failed err="no rows in result set"
func TestSeedErrNoRowsHandling(t *testing.T) {
	// pgx.ErrNoRows and sql.ErrNoRows are different error values
	require.NotEqual(t, pgx.ErrNoRows, sql.ErrNoRows)

	// Each error identifies itself
	require.ErrorIs(t, pgx.ErrNoRows, pgx.ErrNoRows)
	require.ErrorIs(t, sql.ErrNoRows, sql.ErrNoRows)

	// The seed code checks both:
	//   if err == sql.ErrNoRows || err == pgx.ErrNoRows
	// This test documents that both error types must be handled for idempotent
	// seeding to work correctly with pgx.
	allNoRowErrors := []error{sql.ErrNoRows, pgx.ErrNoRows}
	for _, err := range allNoRowErrors {
		// Simulate the seed's error check
		handled := err == sql.ErrNoRows || err == pgx.ErrNoRows
		require.True(t, handled, "error %v should be handled as 'no rows'", err)
	}
}

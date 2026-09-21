//go:build integration

package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database/sqlc"
)

func TestListAttendanceSessions_EmptyDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	svc := NewAttendanceService(pool, queries)

	// Use a dummy school ID - the query will just return empty results
	schoolID := uuid.New()

	params := ListAttendanceSessionsParams{
		SchoolID: schoolID,
		Page:     1,
		Limit:    50,
	}

	sessions, total, err := svc.ListAttendanceSessions(ctx, params)
	if err != nil {
		t.Fatalf("ListAttendanceSessions failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}

	t.Logf("Success! Empty database returns total=%d, sessions=%d", total, len(sessions))
}

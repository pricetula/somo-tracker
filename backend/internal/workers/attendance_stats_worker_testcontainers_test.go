//go:build integration

package workers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
	"somotracker/backend/internal/testcontainers"
)

// TestAttendanceStatsWorker_Upsert verifies the summary table can be written to and read back.
func TestAttendanceStatsWorker_Upsert(t *testing.T) {
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

	// Create minimal school / academic term / student for the summary row
	schoolID := uuid.New()
	termID := uuid.New()
	studentID := uuid.New()

	// Insert minimal rows to satisfy FKs
	_, err = pool.Exec(ctx, `INSERT INTO schools (id, tenant_id, name) VALUES ($1, $2, $3)`, schoolID, uuid.New(), "Test School")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, school_id, name) VALUES ($1, $2, $3)`, uuid.New(), schoolID, "2026")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, academic_year_id, school_id, name, start_date, end_date) VALUES ($1, $2, $3, $4, $5, $6)`,
		termID, uuid.New(), schoolID, "Term 1", "2026-01-01", "2026-06-30")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO students (student_id, school_id, admission_number, full_name, date_of_birth, gender) VALUES ($1, $2, $3, $4, $5, $6)`,
		studentID, schoolID, "2026-001", "Test Student", "2010-01-01", "M")
	require.NoError(t, err)

	// Insert summary row directly – worker stub will upsert in real implementation
	_, err = pool.Exec(ctx, `
		INSERT INTO student_term_attendance_stats (school_id, academic_term_id, student_id, present_count, absent_count, late_count, attendance_rate)
		VALUES ($1, $2, $3, 80, 10, 10, 80.0)
		ON CONFLICT (school_id, academic_term_id, student_id) DO UPDATE SET attendance_rate = EXCLUDED.attendance_rate
	`, schoolID, termID, studentID)
	require.NoError(t, err)

	var rate float64
	err = pool.QueryRow(ctx, `
		SELECT attendance_rate FROM student_term_attendance_stats
		WHERE school_id = $1 AND academic_term_id = $2 AND student_id = $3
	`, schoolID, termID, studentID).Scan(&rate)
	require.NoError(t, err)
	require.Equal(t, 80.0, rate)
}

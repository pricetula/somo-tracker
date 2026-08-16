package timetablestructure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

// ── Test helpers ─────────────────────────────────────────────────────────

func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for dir != "/" {
		if filepath.Base(dir) == "backend" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, "internal", "database", "migrations")
}

func startPG(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       "somotracker_test",
			"POSTGRES_USER":     "somo_admin",
			"POSTGRES_PASSWORD": "somo_secure_password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	host, err := c.Host(ctx)
	require.NoError(t, err)
	port, err := c.MappedPort(ctx, "5432")
	require.NoError(t, err)
	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s:%s/somotracker_test?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	cleanup := func() { pool.Close(); _ = c.Terminate(ctx) }
	return pool, cleanup
}

func applyAllMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(migrationsDir(), "*.up.sql"))
	require.NoError(t, err, "glob migration files")
	for _, path := range files {
		sql, err := os.ReadFile(path)
		require.NoError(t, err, "read migration %s", path)
		_, err = pool.Exec(context.Background(), string(sql))
		require.NoError(t, err, "apply migration %s", path)
	}
}

func seedTenantSchool(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	return tenantID, schoolID
}

func seedAcademicYear(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID string) (yearID string) {
	t.Helper()
	ctx := context.Background()
	yearID = uuid.New().String()
	// Create a default user for FK constraints
	userID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`,
		userID, "sys-timetable@test.com", tenantID, "Timetable System")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		yearID, tenantID, schoolID, "2025", "2025-01-01", "2025-12-31", userID, userID)
	require.NoError(t, err)
	return yearID
}

func newRepo(pool *pgxpool.Pool) *PgRepository {
	return NewRepository(&database.Pools{PG: pool}, zap.NewNop().Sugar())
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestPgRepository_CreateAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	block, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      DayMonday,
		PeriodName:     "Morning Assembly",
		StartTime:      "07:30",
		EndTime:        "07:50",
		IsBreak:        false,
		AcademicYearID: yearID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, block.ID)
	require.Equal(t, "Morning Assembly", block.PeriodName)
	require.Equal(t, "07:30", block.StartTime)
	require.Equal(t, "07:50", block.EndTime)

	// Get by ID
	fetched, err := repo.GetByID(ctx, block.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	require.Equal(t, "Morning Assembly", fetched.PeriodName)
}

func TestPgRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	_, err := repo.GetByID(ctx, "nonexistent-id", tenantID, schoolID)
	require.Error(t, err)
}

func TestPgRepository_ListByDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create two blocks on Monday
	_, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	_, err = repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	// Create a block on Tuesday (should not appear in Monday results)
	_, err = repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayTuesday, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	blocks, err := repo.ListByDay(ctx, tenantID, schoolID, yearID, DayMonday)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	require.Equal(t, "Period 1", blocks[0].PeriodName)
	require.Equal(t, "Period 2", blocks[1].PeriodName)
}

func TestPgRepository_ListByDay_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	blocks, err := repo.ListByDay(ctx, tenantID, schoolID, yearID, DayMonday)
	require.NoError(t, err)
	require.Empty(t, blocks)
}

func TestPgRepository_ListAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create blocks on two different days
	_, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Mon P1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	_, err = repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayTuesday, PeriodName: "Tue P1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	blocks, err := repo.ListAll(ctx, tenantID, schoolID, yearID)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
}

func TestPgRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	block, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Original", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	updated, err := repo.Update(ctx, block.ID, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      DayMonday,
		PeriodName:     "Updated",
		StartTime:      "08:00",
		EndTime:        "08:45",
		IsBreak:        true,
		AcademicYearID: yearID,
	})
	require.NoError(t, err)
	require.Equal(t, "Updated", updated.PeriodName)
	require.Equal(t, "08:45", updated.EndTime)
	require.True(t, updated.IsBreak)
}

func TestPgRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	block, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "To Delete", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	err = repo.Delete(ctx, block.ID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, block.ID, tenantID, schoolID)
	require.Error(t, err)
}

func TestPgRepository_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	err := repo.Delete(ctx, "nonexistent-id", tenantID, schoolID)
	require.Error(t, err)
}

func TestPgRepository_DeleteByDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create blocks on Monday and Tuesday
	_, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Mon P1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	_, err = repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayTuesday, PeriodName: "Tue P1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	// Delete Monday blocks
	err = repo.DeleteByDay(ctx, tenantID, schoolID, yearID, DayMonday)
	require.NoError(t, err)

	blocks, err := repo.ListByDay(ctx, tenantID, schoolID, yearID, DayMonday)
	require.NoError(t, err)
	require.Empty(t, blocks)

	// Tuesday should still have blocks
	blocks, err = repo.ListByDay(ctx, tenantID, schoolID, yearID, DayTuesday)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
}

func TestPgRepository_FindOverlappingBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create a block from 08:00 to 08:40
	block, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	// Check for overlap with a block that starts before existing ends
	overlap, err := repo.FindOverlappingBlock(ctx, tenantID, schoolID, DayMonday, "08:30", "09:00", "")
	require.NoError(t, err)
	require.NotNil(t, overlap)
	require.Equal(t, "Period 1", overlap.PeriodName)

	// Check for overlap excluding the same block (should be nil)
	overlap, err = repo.FindOverlappingBlock(ctx, tenantID, schoolID, DayMonday, "08:30", "09:00", block.ID)
	require.NoError(t, err)
	require.Nil(t, overlap)

	// Check non-overlapping time (should be nil)
	overlap, err = repo.FindOverlappingBlock(ctx, tenantID, schoolID, DayMonday, "09:00", "10:00", "")
	require.NoError(t, err)
	require.Nil(t, overlap)
}

func TestPgRepository_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	blocks, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{
		{DayOfWeek: DayMonday, PeriodName: "Period 1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "Period 2", StartTime: "08:40", EndTime: "09:20", IsBreak: false, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "Break", StartTime: "09:20", EndTime: "09:40", IsBreak: true, AcademicYearID: yearID},
	})
	require.NoError(t, err)
	require.Len(t, blocks, 3)

	// Verify all were created
	listed, err := repo.ListByDay(ctx, tenantID, schoolID, yearID, DayMonday)
	require.NoError(t, err)
	require.Len(t, listed, 3)
}

func TestPgRepository_BatchCreate_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	blocks, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{})
	require.NoError(t, err)
	require.Empty(t, blocks)
}

func TestPgRepository_ReplicateDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create blocks on Monday
	_, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{
		{DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "P2", StartTime: "08:40", EndTime: "09:20", IsBreak: false, AcademicYearID: yearID},
	})
	require.NoError(t, err)

	// Replicate Monday -> Tuesday, Wednesday
	blocks, err := repo.ReplicateDay(ctx, tenantID, schoolID, DayMonday, []int{DayTuesday, DayWednesday})
	require.NoError(t, err)
	require.Len(t, blocks, 4) // 2 blocks × 2 target days

	// Verify Tuesday has the same schedule
	tueBlocks, err := repo.ListByDay(ctx, tenantID, schoolID, yearID, DayTuesday)
	require.NoError(t, err)
	require.Len(t, tueBlocks, 2)
	require.Equal(t, "P1", tueBlocks[0].PeriodName)
	require.Equal(t, "P2", tueBlocks[1].PeriodName)
}

func TestPgRepository_ListByPeriodName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create "Break" blocks on Monday and Tuesday
	_, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{
		{DayOfWeek: DayMonday, PeriodName: "Break", StartTime: "09:20", EndTime: "09:40", IsBreak: true, AcademicYearID: yearID},
		{DayOfWeek: DayTuesday, PeriodName: "Break", StartTime: "09:20", EndTime: "09:40", IsBreak: true, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, AcademicYearID: yearID},
	})
	require.NoError(t, err)

	blocks, err := repo.ListByPeriodName(ctx, tenantID, schoolID, yearID, "Break", "")
	require.NoError(t, err)
	require.Len(t, blocks, 2)

	// Exclude one
	blocks, err = repo.ListByPeriodName(ctx, tenantID, schoolID, yearID, "Break", blocks[0].ID)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
}

func TestPgRepository_ListByDayAfter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	_, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{
		{DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "P2", StartTime: "08:40", EndTime: "09:20", IsBreak: false, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "Break", StartTime: "09:20", EndTime: "09:40", IsBreak: true, AcademicYearID: yearID},
	})
	require.NoError(t, err)

	blocks, err := repo.ListByDayAfter(ctx, tenantID, schoolID, yearID, DayMonday, "09:00", "")
	require.NoError(t, err)
	require.Len(t, blocks, 1) // only Break starts at or after 09:00
	require.Equal(t, "Break", blocks[0].PeriodName)
}

func TestPgRepository_DeleteByPeriodName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	_, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{
		{DayOfWeek: DayMonday, PeriodName: "Break", StartTime: "09:20", EndTime: "09:40", IsBreak: true, AcademicYearID: yearID},
		{DayOfWeek: DayTuesday, PeriodName: "Break", StartTime: "09:20", EndTime: "09:40", IsBreak: true, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, AcademicYearID: yearID},
	})
	require.NoError(t, err)

	deleted, err := repo.DeleteByPeriodName(ctx, tenantID, schoolID, yearID, "Break")
	require.NoError(t, err)
	require.Equal(t, 2, deleted)

	// Verify: P1 should remain
	blocks, err := repo.ListByDay(ctx, tenantID, schoolID, yearID, DayMonday)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	require.Equal(t, "P1", blocks[0].PeriodName)
}

func TestPgRepository_BatchUpdateBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	blocks, err := repo.BatchCreate(ctx, tenantID, schoolID, []CreateTimeBlockPayload{
		{DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40", IsBreak: false, AcademicYearID: yearID},
		{DayOfWeek: DayMonday, PeriodName: "P2", StartTime: "08:40", EndTime: "09:20", IsBreak: false, AcademicYearID: yearID},
	})
	require.NoError(t, err)

	// Update both blocks with new non-overlapping times (wide enough gap to
	// avoid the exclusion constraint during the per-statement check)
	updated, err := repo.BatchUpdateBlocks(ctx, tenantID, schoolID, []BatchBlockUpdate{
		{ID: blocks[0].ID, StartTime: "08:05", EndTime: "08:30"},
		{ID: blocks[1].ID, StartTime: "08:35", EndTime: "09:00"},
	})
	require.NoError(t, err)
	require.Len(t, updated, 2)
	require.Equal(t, "08:05", updated[0].StartTime)
	require.Equal(t, "09:00", updated[1].EndTime)
}

func TestPgRepository_HasLinkedLessons(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	block, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	// No linked lessons yet
	count, err := repo.HasLinkedLessons(ctx, block.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestPgRepository_HasLinkedLessonsForBlocks_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	repo := newRepo(pool)

	count, err := repo.HasLinkedLessonsForBlocks(ctx, []string{})
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestPgRepository_CreateOverlapError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create first block
	_, err := repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "P1", StartTime: "08:00", EndTime: "08:40",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	// Attempt to create an overlapping block
	_, err = repo.Create(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek: DayMonday, PeriodName: "Overlap", StartTime: "08:20", EndTime: "09:00",
		IsBreak: false, AcademicYearID: yearID,
	})
	require.Error(t, err)
}

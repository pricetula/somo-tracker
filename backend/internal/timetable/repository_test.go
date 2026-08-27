package timetable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"somotracker/backend/internal/database"
)

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

func seedMinimalTenant(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID string) {
	t.Helper()
	ctx := context.Background()

	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	userID = uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "user@test.com", tenantID, "Test User")
	require.NoError(t, err)

	return tenantID, schoolID, userID
}

func seedAcademicYear(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, userID string) (academicYearID string) {
	t.Helper()
	ctx := context.Background()

	academicYearID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	return academicYearID
}

func seedTrack(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, academicYearID, userID string) (trackID string) {
	t.Helper()
	ctx := context.Background()

	// Need academic term first
	academicTermID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', true, $5, $5)`,
		academicTermID, tenantID, schoolID, academicYearID, userID)
	require.NoError(t, err)

	trackID = uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_tracks (id, tenant_id, school_id, academic_year_id, academic_term_id, name, is_default) VALUES ($1, $2, $3, $4, $5, 'Main Track', true)`,
		trackID, tenantID, schoolID, academicYearID, academicTermID)
	require.NoError(t, err)

	return trackID
}

func seedStreamClassLearningArea(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, academicYearID string) (streamID, classID, learningAreaID string) {
	t.Helper()
	ctx := context.Background()

	streamID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	classID = uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	learningAreaID = uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'English', 'ENG', 'Upper_Primary', 'G4')`,
		learningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	return streamID, classID, learningAreaID
}

func newRepo(pool *pgxpool.Pool) Repository {
	return NewRepository(&database.Pools{PG: pool}, zap.NewNop().Sugar())
}

// setupTimeBlockTestData applies migrations and seeds common data for time block tests.
// Returns tenantID, schoolID, userID, academicYearID, trackID
func setupTimeBlockTestData(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID, academicYearID, trackID string) {
	t.Helper()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID = seedMinimalTenant(t, pool)
	academicYearID = seedAcademicYear(t, pool, tenantID, schoolID, userID)
	trackID = seedTrack(t, pool, tenantID, schoolID, academicYearID, userID)

	return tenantID, schoolID, userID, academicYearID, trackID
}

func TestPgRepository_ListBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Insert some time blocks
	structID1 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false, 1)`,
		structID1, tenantID, schoolID, trackID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false, 2)`,
		structID2, tenantID, schoolID, trackID)
	require.NoError(t, err)

	structID3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Break', '09:25', '09:40', true, 3)`,
		structID3, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Different day
	structID4 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 2, 'Lesson 1', '08:00', '08:40', false, 1)`,
		structID4, tenantID, schoolID, trackID)
	require.NoError(t, err)

	blocks, err := repo.ListBlocks(ctx, tenantID, schoolID, academicYearID)
	require.NoError(t, err)
	require.Len(t, blocks, 4)

	// Verify ordering: by day_of_week, then order_index, then start_time
	require.Equal(t, structID1, blocks[0].ID)
	require.Equal(t, 1, blocks[0].DayOfWeek)
	require.Equal(t, "Lesson 1", blocks[0].PeriodName)
	require.False(t, blocks[0].IsBreak)

	require.Equal(t, structID2, blocks[1].ID)

	require.Equal(t, structID3, blocks[2].ID)
	require.True(t, blocks[2].IsBreak)

	require.Equal(t, structID4, blocks[3].ID)
	require.Equal(t, 2, blocks[3].DayOfWeek)

	// Empty result for different academic year
	otherYearID := uuid.New().String()
	blocks, err = repo.ListBlocks(ctx, tenantID, schoolID, otherYearID)
	require.NoError(t, err)
	require.Empty(t, blocks)
}

func TestPgRepository_ListBlocks_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, academicYearID, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// No blocks inserted - should return empty slice, not error
	blocks, err := repo.ListBlocks(ctx, tenantID, schoolID, academicYearID)
	require.NoError(t, err)
	require.Empty(t, blocks)
	require.Len(t, blocks, 0)
}

func TestPgRepository_GetBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false, 1)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Found
	block, err := repo.GetBlock(ctx, structID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, structID, block.ID)
	require.Equal(t, "Lesson 1", block.PeriodName)
	require.Equal(t, 1, block.DayOfWeek)
	require.False(t, block.IsBreak)

	// Not found (wrong ID)
	_, err = repo.GetBlock(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Not found (wrong tenant)
	_, err = repo.GetBlock(ctx, structID, uuid.New().String(), schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Not found (wrong school)
	_, err = repo.GetBlock(ctx, structID, tenantID, uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_CreateBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create a time block
	block, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, block.ID)
	require.Equal(t, trackID, block.TrackID)
	require.Equal(t, 1, block.DayOfWeek)
	require.Equal(t, "Lesson 1", block.PeriodName)
	require.True(t, strings.HasPrefix(block.StartTime, "08:00"), "StartTime: %s", block.StartTime)
	require.True(t, strings.HasPrefix(block.EndTime, "08:40"), "EndTime: %s", block.EndTime)
	require.False(t, block.IsBreak)
	require.NotEmpty(t, block.AcademicYearID) // derived from track

	// Verify it's in the database
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM timetable_blocks WHERE id = $1`, block.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestPgRepository_CreateBlock_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create first block
	_, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.NoError(t, err)

	// Try to create overlapping block (same day, overlapping time) — should fail due to exclusion constraint
	_, err = repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 2",
		StartTime:  "08:20", // Overlaps with 08:00-08:40
		EndTime:    "09:00",
		IsBreak:    false,
		OrderIndex: 2,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBlockOverlap)
}

func TestPgRepository_UpdateBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create a block
	block, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.NoError(t, err)

	// Update the block
	updated, err := repo.UpdateBlock(ctx, block.ID, tenantID, schoolID, UpdateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  2,
		PeriodName: "Lesson 2",
		StartTime:  "09:00",
		EndTime:    "09:40",
		IsBreak:    true,
		OrderIndex: 3,
	})
	require.NoError(t, err)
	require.Equal(t, block.ID, updated.ID)
	require.Equal(t, trackID, updated.TrackID)
	require.Equal(t, 2, updated.DayOfWeek)
	require.Equal(t, "Lesson 2", updated.PeriodName)
	require.True(t, strings.HasPrefix(updated.StartTime, "09:00"), "StartTime: %s", updated.StartTime)
	require.True(t, strings.HasPrefix(updated.EndTime, "09:40"), "EndTime: %s", updated.EndTime)
	require.True(t, updated.IsBreak)

	// Verify in DB
	var periodName string
	err = pool.QueryRow(ctx, `SELECT period_name FROM timetable_blocks WHERE id = $1`, block.ID).Scan(&periodName)
	require.NoError(t, err)
	require.Equal(t, "Lesson 2", periodName)
}

func TestPgRepository_UpdateBlock_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, err := repo.UpdateBlock(ctx, uuid.New().String(), tenantID, schoolID, UpdateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpdateBlock_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create two non-overlapping blocks
	_, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.NoError(t, err)

	block2, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 2",
		StartTime:  "08:45",
		EndTime:    "09:25",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.NoError(t, err)

	// Try to update block2 to overlap with block1
	_, err = repo.UpdateBlock(ctx, block2.ID, tenantID, schoolID, UpdateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 2 Updated",
		StartTime:  "08:20", // Overlaps with block1
		EndTime:    "09:00",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBlockOverlap)
}

func TestPgRepository_DeleteBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create a block
	block, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		TrackID:    trackID,
		DayOfWeek:  1,
		PeriodName: "Lesson 1",
		StartTime:  "08:00",
		EndTime:    "08:40",
		IsBreak:    false,
		OrderIndex: 1,
	})
	require.NoError(t, err)

	// Delete it
	err = repo.DeleteBlock(ctx, block.ID, tenantID, schoolID)
	require.NoError(t, err)

	// Verify it's gone
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM timetable_blocks WHERE id = $1`, block.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestPgRepository_DeleteBlock_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	err := repo.DeleteBlock(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_ListAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Seed stream, class, learning area
	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	// Create timetable structures (blocks)
	structID1 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID1, tenantID, schoolID, trackID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false)`,
		structID2, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Create allocations (NO academic_year_id column)
	allocID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, block_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		allocID1, tenantID, schoolID, structID1, classID, learningAreaID, userID)
	require.NoError(t, err)

	allocID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, block_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		allocID2, tenantID, schoolID, structID2, classID, learningAreaID, userID)
	require.NoError(t, err)

	// List all
	filter := AllocationFilter{TenantID: tenantID, SchoolID: schoolID}
	allocations, err := repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	// Filter by academic year
	filter.AcademicYearID = academicYearID
	allocations, err = repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	// Filter by structure
	filter.BlockID = structID1
	allocations, err = repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, allocID1, allocations[0].ID)

	// Filter by class
	filter.BlockID = ""
	filter.ClassID = classID
	allocations, err = repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	// Filter by teacher
	filter.ClassID = ""
	filter.TeacherID = userID
	allocations, err = repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	// Filter by learning area
	filter.TeacherID = ""
	filter.LearningAreaID = learningAreaID
	allocations, err = repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	// Empty result for different tenant
	filter = AllocationFilter{TenantID: uuid.New().String(), SchoolID: schoolID}
	allocations, err = repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Empty(t, allocations)
}

func TestPgRepository_ListAllocations_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, academicYearID, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// No allocations inserted - should return empty slice, not error
	filter := AllocationFilter{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: academicYearID}
	allocations, err := repo.ListAllocations(ctx, filter)
	require.NoError(t, err)
	require.Empty(t, allocations)
	require.Len(t, allocations, 0)
}

func TestPgRepository_GetAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	allocID := uuid.New().String()
	room := "Room 101"
	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, block_id, class_id, learning_area_id, teacher_id, room_identifier) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		allocID, tenantID, schoolID, structID, classID, learningAreaID, userID, room)
	require.NoError(t, err)

	// Found
	alloc, err := repo.GetAllocation(ctx, allocID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, allocID, alloc.ID)
	require.Equal(t, tenantID, alloc.TenantID)
	require.Equal(t, schoolID, alloc.SchoolID)
	require.Equal(t, academicYearID, alloc.AcademicYearID) // derived from track
	require.Equal(t, structID, alloc.BlockID)
	require.Equal(t, classID, alloc.ClassID)
	require.Equal(t, learningAreaID, alloc.LearningAreaID)
	require.Equal(t, userID, alloc.TeacherID)
	require.NotNil(t, alloc.RoomIdentifier)
	require.Equal(t, "Room 101", *alloc.RoomIdentifier)
	// Check joined fields
	require.NotEmpty(t, alloc.ClassName)
	require.NotEmpty(t, alloc.LearningAreaName)
	require.NotEmpty(t, alloc.LearningAreaCode)
	require.NotEmpty(t, alloc.TeacherName)
	require.NotEmpty(t, alloc.RoomName)

	// Not found (wrong ID)
	_, err = repo.GetAllocation(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Not found (wrong tenant)
	_, err = repo.GetAllocation(ctx, allocID, uuid.New().String(), schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Not found (wrong school)
	_, err = repo.GetAllocation(ctx, allocID, tenantID, uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_CreateAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Create allocation (no academicYearID parameter)
	alloc, err := repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, alloc.ID)
	require.Equal(t, tenantID, alloc.TenantID)
	require.Equal(t, schoolID, alloc.SchoolID)
	require.Equal(t, academicYearID, alloc.AcademicYearID) // derived from track via block
	require.Equal(t, structID, alloc.BlockID)
	require.Equal(t, classID, alloc.ClassID)
	require.Equal(t, learningAreaID, alloc.LearningAreaID)
	require.Equal(t, userID, alloc.TeacherID)
	require.NotNil(t, alloc.RoomIdentifier)
	require.Equal(t, "Room 101", *alloc.RoomIdentifier)
	// Joined fields should be empty on create (not fetched)
	require.Empty(t, alloc.ClassName)
	require.Empty(t, alloc.LearningAreaName)
	require.Empty(t, alloc.LearningAreaCode)
	require.Empty(t, alloc.TeacherName)
	require.Empty(t, alloc.RoomName)
}

func TestPgRepository_CreateAllocation_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Create first allocation
	_, err = repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)

	// Same class + same block with different subject is now allowed (unique_class_slot removed)
	_, err = repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        classID,             // Same class + structure — allowed
		LearningAreaID: uuid.New().String(), // Different learning area
		TeacherID:      uuid.New().String(), // Different teacher
		RoomIdentifier: ptr("Room 102"),
	})
	require.NoError(t, err) // Should succeed now

	// Try to create another allocation with same teacher + structure (unique_teacher_slot constraint)
	_, err = repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        uuid.New().String(), // Different class
		LearningAreaID: uuid.New().String(), // Different learning area
		TeacherID:      userID,              // Same teacher + structure
		RoomIdentifier: ptr("Room 103"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)

	// Try to create another allocation with same room + structure (unique_room_slot constraint)
	_, err = repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        uuid.New().String(),
		LearningAreaID: uuid.New().String(),
		TeacherID:      uuid.New().String(),
		RoomIdentifier: ptr("Room 101"), // Same room + structure
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRoomDoubleBooked)
}

func TestPgRepository_BatchCreateAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID1, learningAreaID1 := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	// Second class
	streamID2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Green')`,
		streamID2, tenantID, schoolID)
	require.NoError(t, err)
	classID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID2, tenantID, schoolID, academicYearID, streamID2)
	require.NoError(t, err)

	learningAreaID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		learningAreaID2, tenantID, schoolID)
	require.NoError(t, err)

	// Create structures
	structID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID1, tenantID, schoolID, trackID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false)`,
		structID2, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Batch create allocations (no academicYearID parameter)
	payloads := []CreateAllocationPayload{
		{BlockID: structID1, ClassID: classID1, LearningAreaID: learningAreaID1, TeacherID: userID, RoomIdentifier: ptr("Room 101")},
		{BlockID: structID2, ClassID: classID2, LearningAreaID: learningAreaID2, TeacherID: userID, RoomIdentifier: ptr("Room 102")},
	}

	allocs, err := repo.BatchCreateAllocations(ctx, tenantID, schoolID, payloads)
	require.NoError(t, err)
	require.Len(t, allocs, 2)
	require.Equal(t, structID1, allocs[0].BlockID)
	require.Equal(t, classID1, allocs[0].ClassID)
	require.Equal(t, structID2, allocs[1].BlockID)
	require.Equal(t, classID2, allocs[1].ClassID)
}

func TestPgRepository_BatchCreateAllocations_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	allocs, err := repo.BatchCreateAllocations(ctx, tenantID, schoolID, []CreateAllocationPayload{})
	require.NoError(t, err)
	require.Empty(t, allocs)
}

func TestPgRepository_BatchCreateAllocations_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Batch with duplicate teacher + same block (unique_teacher_slot constraint)
	payloads := []CreateAllocationPayload{
		{BlockID: structID, ClassID: classID, LearningAreaID: learningAreaID, TeacherID: userID},
		{BlockID: structID, ClassID: uuid.New().String(), LearningAreaID: uuid.New().String(), TeacherID: userID}, // Duplicate teacher + structure
	}

	_, err = repo.BatchCreateAllocations(ctx, tenantID, schoolID, payloads)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConflict)
}

func TestPgRepository_UpdateAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Create allocation
	alloc, err := repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)

	// Update allocation
	newLearningAreaID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Kiswahili', 'KISW', 'Upper_Primary', 'G4')`,
		newLearningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	newTeacherID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		newTeacherID, "teacher2@test.com", tenantID, "Teacher 2")
	require.NoError(t, err)

	updated, err := repo.UpdateAllocation(ctx, alloc.ID, tenantID, schoolID, UpdateAllocationPayload{
		LearningAreaID: newLearningAreaID,
		TeacherID:      newTeacherID,
		RoomIdentifier: ptr("Room 202"),
	})
	require.NoError(t, err)
	require.Equal(t, alloc.ID, updated.ID)
	require.Equal(t, newLearningAreaID, updated.LearningAreaID)
	require.Equal(t, newTeacherID, updated.TeacherID)
	require.NotNil(t, updated.RoomIdentifier)
	require.Equal(t, "Room 202", *updated.RoomIdentifier)
}

func TestPgRepository_UpdateAllocation_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, err := repo.UpdateAllocation(ctx, uuid.New().String(), tenantID, schoolID, UpdateAllocationPayload{
		LearningAreaID: uuid.New().String(),
		TeacherID:      uuid.New().String(),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpdateAllocation_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID1, learningAreaID1 := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	// Second class
	streamID2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Green')`,
		streamID2, tenantID, schoolID)
	require.NoError(t, err)
	classID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID2, tenantID, schoolID, academicYearID, streamID2)
	require.NoError(t, err)

	learningAreaID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		learningAreaID2, tenantID, schoolID)
	require.NoError(t, err)

	// Create two structures
	structID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID1, tenantID, schoolID, trackID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false)`,
		structID2, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Create two allocations with different classes but same teacher
	_, err = repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID1,
		ClassID:        classID1,
		LearningAreaID: learningAreaID1,
		TeacherID:      userID,
	})
	require.NoError(t, err)

	alloc2, err := repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID2,
		ClassID:        classID2,
		LearningAreaID: learningAreaID2,
		TeacherID:      userID,
	})
	require.NoError(t, err)

	// Try to update alloc2 to have same teacher + structure as alloc1 (conflict on unique_teacher_slot)
	_, err = repo.UpdateAllocation(ctx, alloc2.ID, tenantID, schoolID, UpdateAllocationPayload{
		LearningAreaID: learningAreaID2,
		TeacherID:      userID, // Same teacher, but different structure — this should be OK
		RoomIdentifier: ptr("Room 202"),
	})
	require.NoError(t, err) // Different structure, so OK

	// Now create a third structure and try to create an allocation with teacher conflict on SAME structure
	structID3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 3', '09:30', '10:10', false)`,
		structID3, tenantID, schoolID, trackID)
	require.NoError(t, err)

	// Update alloc2 to use structID3 and same teacher as alloc1 — should conflict
	_, err = repo.UpdateAllocation(ctx, alloc2.ID, tenantID, schoolID, UpdateAllocationPayload{
		LearningAreaID: learningAreaID2,
		TeacherID:      userID, // Same teacher as alloc1, but alloc1 is on structID1
		RoomIdentifier: ptr("Room 202"),
	})
	require.NoError(t, err) // Different structure, so still OK

	// To test teacher conflict, we need to update to same structure
	// First, create an allocation on structID1 with a different teacher
	teacher3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		teacher3, "teacher3@test.com", tenantID, "Teacher 3")
	require.NoError(t, err)

	alloc3, err := repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID1,
		ClassID:        classID2,
		LearningAreaID: learningAreaID2,
		TeacherID:      teacher3,
	})
	require.NoError(t, err)

	// Now try to update alloc3 to use userID (teacher conflict on same structure)
	_, err = repo.UpdateAllocation(ctx, alloc3.ID, tenantID, schoolID, UpdateAllocationPayload{
		LearningAreaID: learningAreaID2,
		TeacherID:      userID, // Conflict with alloc1 on structID1
		RoomIdentifier: ptr("Room 203"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)
}

func TestPgRepository_DeleteAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID, trackID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	alloc, err := repo.CreateAllocation(ctx, tenantID, schoolID, CreateAllocationPayload{
		BlockID:        structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
	})
	require.NoError(t, err)

	// Delete it
	err = repo.DeleteAllocation(ctx, alloc.ID, tenantID, schoolID)
	require.NoError(t, err)

	// Verify it's gone
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM timetable_allocations WHERE id = $1`, alloc.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestPgRepository_DeleteAllocation_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	err := repo.DeleteAllocation(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

// Helper to create pointer to string
func ptr(s string) *string {
	return &s
}

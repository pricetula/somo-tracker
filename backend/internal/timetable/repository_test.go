package timetable

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
	"strings"

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
	// Add order_index column if missing (repository code expects it)
	_, err = pool.Exec(context.Background(), `ALTER TABLE timetable_structures ADD COLUMN IF NOT EXISTS order_index INT NOT NULL DEFAULT 0`)
	require.NoError(t, err, "add order_index column")
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
func setupTimeBlockTestData(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID, academicYearID string) {
	t.Helper()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID = seedMinimalTenant(t, pool)
	academicYearID = seedAcademicYear(t, pool, tenantID, schoolID, userID)

	return tenantID, schoolID, userID, academicYearID
}

func TestPgRepository_ListBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Insert some time blocks
	structID1 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false, 1)`,
		structID1, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false, 2)`,
		structID2, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	structID3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Break', '09:25', '09:40', true, 3)`,
		structID3, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Different day
	structID4 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 2, 'Lesson 1', '08:00', '08:40', false, 1)`,
		structID4, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	blocks, err := repo.ListBlocks(ctx, tenantID, schoolID, academicYearID)
	require.NoError(t, err)
	require.Len(t, blocks, 4)

	// Verify ordering: by day_of_week, then order_index, then start_time
	require.Equal(t, structID1, blocks[0].ID)
	require.Equal(t, 1, blocks[0].DayOfWeek)
	require.Equal(t, 1, blocks[0].Order)
	require.Equal(t, "Lesson 1", blocks[0].PeriodName)
	require.False(t, blocks[0].IsBreak)

	require.Equal(t, structID2, blocks[1].ID)
	require.Equal(t, 2, blocks[1].Order)

	require.Equal(t, structID3, blocks[2].ID)
	require.Equal(t, 3, blocks[2].Order)
	require.True(t, blocks[2].IsBreak)

	require.Equal(t, structID4, blocks[3].ID)
	require.Equal(t, 2, blocks[3].DayOfWeek)
	require.Equal(t, 1, blocks[3].Order)

	// Empty result for different academic year
	otherYearID := uuid.New().String()
	blocks, err = repo.ListBlocks(ctx, tenantID, schoolID, otherYearID)
	require.NoError(t, err)
	require.Empty(t, blocks)
}

func TestPgRepository_GetBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false, 1)`,
		structID, tenantID, schoolID, academicYearID)
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

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create a time block
	block, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, block.ID)
	require.Equal(t, 1, block.DayOfWeek)
	require.Equal(t, "Lesson 1", block.PeriodName)
	require.True(t, strings.HasPrefix(block.StartTime, "08:00"), "StartTime: %s", block.StartTime)
	require.True(t, strings.HasPrefix(block.EndTime, "08:40"), "EndTime: %s", block.EndTime)
	require.False(t, block.IsBreak)
	require.Equal(t, academicYearID, block.AcademicYearID)
	require.Equal(t, 1, block.Order)

	// Verify it's in the database
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM timetable_structures WHERE id = $1`, block.ID).Scan(&count)
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

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create first block
	_, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          1,
	})
	require.NoError(t, err)

	// Try to create overlapping block (same day, overlapping time) — should fail due to exclusion constraint
	_, err = repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 2",
		StartTime:      "08:20", // Overlaps with 08:00-08:40
		EndTime:        "09:00",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          2,
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

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create a block
	block, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          1,
	})
	require.NoError(t, err)

	// Update the block
	updated, err := repo.UpdateBlock(ctx, block.ID, tenantID, schoolID, UpdateTimeBlockPayload{
		DayOfWeek:      2,
		PeriodName:     "Lesson 2",
		StartTime:      "09:00",
		EndTime:        "09:40",
		IsBreak:        true,
		AcademicYearID: academicYearID,
		Order:          5,
	})
	require.NoError(t, err)
	require.Equal(t, block.ID, updated.ID)
	require.Equal(t, 2, updated.DayOfWeek)
	require.Equal(t, "Lesson 2", updated.PeriodName)
	require.True(t, strings.HasPrefix(updated.StartTime, "09:00"), "StartTime: %s", updated.StartTime)
	require.True(t, strings.HasPrefix(updated.EndTime, "09:40"), "EndTime: %s", updated.EndTime)
	require.True(t, updated.IsBreak)
	require.Equal(t, 5, updated.Order)

	// Verify in DB
	var periodName string
	err = pool.QueryRow(ctx, `SELECT period_name FROM timetable_structures WHERE id = $1`, block.ID).Scan(&periodName)
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

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, err := repo.UpdateBlock(ctx, uuid.New().String(), tenantID, schoolID, UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          1,
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

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create two non-overlapping blocks
	_, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          1,
	})
	require.NoError(t, err)

	block2, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 2",
		StartTime:      "08:45",
		EndTime:        "09:25",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          2,
	})
	require.NoError(t, err)

	// Try to update block2 to overlap with block1
	_, err = repo.UpdateBlock(ctx, block2.ID, tenantID, schoolID, UpdateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 2 Updated",
		StartTime:      "08:20", // Overlaps with block1
		EndTime:        "09:00",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          2,
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

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Create a block
	block, err := repo.CreateBlock(ctx, tenantID, schoolID, CreateTimeBlockPayload{
		DayOfWeek:      1,
		PeriodName:     "Lesson 1",
		StartTime:      "08:00",
		EndTime:        "08:40",
		IsBreak:        false,
		AcademicYearID: academicYearID,
		Order:          1,
	})
	require.NoError(t, err)

	// Delete it
	err = repo.DeleteBlock(ctx, block.ID, tenantID, schoolID)
	require.NoError(t, err)

	// Verify it's gone
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM timetable_structures WHERE id = $1`, block.ID).Scan(&count)
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

	tenantID, schoolID, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	err := repo.DeleteBlock(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_ListSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	// Seed stream, class, learning area
	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	// Create timetable structures
	structID1 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID1, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false)`,
		structID2, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Create slots
	slotID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slotID1, tenantID, schoolID, academicYearID, structID1, classID, learningAreaID, userID)
	require.NoError(t, err)

	slotID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slotID2, tenantID, schoolID, academicYearID, structID2, classID, learningAreaID, userID)
	require.NoError(t, err)

	// List all
	filter := SlotFilter{TenantID: tenantID, SchoolID: schoolID}
	slots, err := repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Len(t, slots, 2)

	// Filter by academic year
	filter.AcademicYearID = academicYearID
	slots, err = repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Len(t, slots, 2)

	// Filter by structure
	filter.StructureID = structID1
	slots, err = repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Len(t, slots, 1)
	require.Equal(t, slotID1, slots[0].ID)

	// Filter by class
	filter.StructureID = ""
	filter.ClassID = classID
	slots, err = repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Len(t, slots, 2)

	// Filter by teacher
	filter.ClassID = ""
	filter.TeacherID = userID
	slots, err = repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Len(t, slots, 2)

	// Filter by learning area
	filter.TeacherID = ""
	filter.LearningAreaID = learningAreaID
	slots, err = repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Len(t, slots, 2)

	// Empty result for different tenant
	filter = SlotFilter{TenantID: uuid.New().String(), SchoolID: schoolID}
	slots, err = repo.ListSlots(ctx, filter)
	require.NoError(t, err)
	require.Empty(t, slots)
}

func TestPgRepository_GetSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	slotID := uuid.New().String()
	room := "Room 101"
	_, err = pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id, room_identifier) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		slotID, tenantID, schoolID, academicYearID, structID, classID, learningAreaID, userID, room)
	require.NoError(t, err)

	// Found
	slot, err := repo.GetSlot(ctx, slotID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, slotID, slot.ID)
	require.Equal(t, tenantID, slot.TenantID)
	require.Equal(t, schoolID, slot.SchoolID)
	require.Equal(t, academicYearID, slot.AcademicYearID)
	require.Equal(t, structID, slot.StructureID)
	require.Equal(t, classID, slot.ClassID)
	require.Equal(t, learningAreaID, slot.LearningAreaID)
	require.Equal(t, userID, slot.TeacherID)
	require.NotNil(t, slot.RoomIdentifier)
	require.Equal(t, "Room 101", *slot.RoomIdentifier)

	// Not found (wrong ID)
	_, err = repo.GetSlot(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Not found (wrong tenant)
	_, err = repo.GetSlot(ctx, slotID, uuid.New().String(), schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Not found (wrong school)
	_, err = repo.GetSlot(ctx, slotID, tenantID, uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_CreateSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Create slot
	slot, err := repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, slot.ID)
	require.Equal(t, tenantID, slot.TenantID)
	require.Equal(t, schoolID, slot.SchoolID)
	require.Equal(t, academicYearID, slot.AcademicYearID)
	require.Equal(t, structID, slot.StructureID)
	require.Equal(t, classID, slot.ClassID)
	require.Equal(t, learningAreaID, slot.LearningAreaID)
	require.Equal(t, userID, slot.TeacherID)
	require.NotNil(t, slot.RoomIdentifier)
	require.Equal(t, "Room 101", *slot.RoomIdentifier)
}

func TestPgRepository_CreateSlot_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Create first slot
	_, err = repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)

	// Try to create another slot with same class + structure (unique_class_slot constraint)
	_, err = repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        classID,             // Same class + structure
		LearningAreaID: uuid.New().String(), // Different learning area
		TeacherID:      uuid.New().String(), // Different teacher
		RoomIdentifier: ptr("Room 102"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrClassSlotOccupied)

	// Try to create another slot with same teacher + structure (unique_teacher_slot constraint)
	_, err = repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        uuid.New().String(), // Different class
		LearningAreaID: uuid.New().String(), // Different learning area
		TeacherID:      userID,              // Same teacher + structure
		RoomIdentifier: ptr("Room 103"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)

	// Try to create another slot with same room + structure (unique_room_slot constraint)
	_, err = repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        uuid.New().String(),
		LearningAreaID: uuid.New().String(),
		TeacherID:      uuid.New().String(),
		RoomIdentifier: ptr("Room 101"), // Same room + structure
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRoomDoubleBooked)
}

func TestPgRepository_BatchCreateSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
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
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID1, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false)`,
		structID2, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Batch create slots
	payloads := []SlotPayload{
		{StructureID: structID1, ClassID: classID1, LearningAreaID: learningAreaID1, TeacherID: userID, RoomIdentifier: ptr("Room 101")},
		{StructureID: structID2, ClassID: classID2, LearningAreaID: learningAreaID2, TeacherID: userID, RoomIdentifier: ptr("Room 102")},
	}

	slots, err := repo.BatchCreateSlots(ctx, tenantID, schoolID, academicYearID, payloads)
	require.NoError(t, err)
	require.Len(t, slots, 2)
	require.Equal(t, structID1, slots[0].StructureID)
	require.Equal(t, classID1, slots[0].ClassID)
	require.Equal(t, structID2, slots[1].StructureID)
	require.Equal(t, classID2, slots[1].ClassID)
}

func TestPgRepository_BatchCreateSlots_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	slots, err := repo.BatchCreateSlots(ctx, tenantID, schoolID, academicYearID, []SlotPayload{})
	require.NoError(t, err)
	require.Empty(t, slots)
}

func TestPgRepository_BatchCreateSlots_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Batch with duplicate class+structure in same batch
	payloads := []SlotPayload{
		{StructureID: structID, ClassID: classID, LearningAreaID: learningAreaID, TeacherID: userID},
		{StructureID: structID, ClassID: classID, LearningAreaID: uuid.New().String(), TeacherID: uuid.New().String()}, // Duplicate class+structure
	}

	_, err = repo.BatchCreateSlots(ctx, tenantID, schoolID, academicYearID, payloads)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConflict)
}

func TestPgRepository_UpdateSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Create slot
	slot, err := repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
		RoomIdentifier: ptr("Room 101"),
	})
	require.NoError(t, err)

	// Update slot
	newLearningAreaID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Kiswahili', 'KISW', 'Upper_Primary', 'G4')`,
		newLearningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	newTeacherID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		newTeacherID, "teacher2@test.com", tenantID, "Teacher 2")
	require.NoError(t, err)

	updated, err := repo.UpdateSlot(ctx, slot.ID, tenantID, schoolID, UpdateSlotPayload{
		LearningAreaID: newLearningAreaID,
		TeacherID:      newTeacherID,
		RoomIdentifier: ptr("Room 202"),
	})
	require.NoError(t, err)
	require.Equal(t, slot.ID, updated.ID)
	require.Equal(t, newLearningAreaID, updated.LearningAreaID)
	require.Equal(t, newTeacherID, updated.TeacherID)
	require.NotNil(t, updated.RoomIdentifier)
	require.Equal(t, "Room 202", *updated.RoomIdentifier)
}

func TestPgRepository_UpdateSlot_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, err := repo.UpdateSlot(ctx, uuid.New().String(), tenantID, schoolID, UpdateSlotPayload{
		LearningAreaID: uuid.New().String(),
		TeacherID:      uuid.New().String(),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpdateSlot_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
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
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID1, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	structID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 2', '08:45', '09:25', false)`,
		structID2, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Create two slots with different classes but same teacher
	_, err = repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID1,
		ClassID:        classID1,
		LearningAreaID: learningAreaID1,
		TeacherID:      userID,
	})
	require.NoError(t, err)

	slot2, err := repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID2,
		ClassID:        classID2,
		LearningAreaID: learningAreaID2,
		TeacherID:      userID,
	})
	require.NoError(t, err)

	// Try to update slot2 to have same teacher + structure as slot1 (conflict on unique_teacher_slot)
	_, err = repo.UpdateSlot(ctx, slot2.ID, tenantID, schoolID, UpdateSlotPayload{
		LearningAreaID: learningAreaID2,
		TeacherID:      userID, // Same teacher, but different structure — this should be OK
		RoomIdentifier: ptr("Room 202"),
	})
	require.NoError(t, err) // Different structure, so OK

	// Now create a third structure and try to create a slot with teacher conflict on SAME structure
	structID3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 3', '09:30', '10:10', false)`,
		structID3, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Update slot2 to use structID3 and same teacher as slot1 — should conflict
	_, err = repo.UpdateSlot(ctx, slot2.ID, tenantID, schoolID, UpdateSlotPayload{
		LearningAreaID: learningAreaID2,
		TeacherID:      userID, // Same teacher as slot1, but slot1 is on structID1
		RoomIdentifier: ptr("Room 202"),
	})
	require.NoError(t, err) // Different structure, so still OK

	// To test teacher conflict, we need to update to same structure
	// First, create a slot on structID1 with a different teacher
	teacher3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		teacher3, "teacher3@test.com", tenantID, "Teacher 3")
	require.NoError(t, err)

	slot3, err := repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID1,
		ClassID:        classID2,
		LearningAreaID: learningAreaID2,
		TeacherID:      teacher3,
	})
	require.NoError(t, err)

	// Now try to update slot3 to use userID (teacher conflict on same structure)
	_, err = repo.UpdateSlot(ctx, slot3.ID, tenantID, schoolID, UpdateSlotPayload{
		LearningAreaID: learningAreaID2,
		TeacherID:      userID, // Conflict with slot1 on structID1
		RoomIdentifier: ptr("Room 203"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTeacherDoubleBooked)
}

func TestPgRepository_DeleteSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, userID, academicYearID := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	_, classID, learningAreaID := seedStreamClassLearningArea(t, pool, tenantID, schoolID, academicYearID)

	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '08:40', false)`,
		structID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	slot, err := repo.CreateSlot(ctx, tenantID, schoolID, academicYearID, SlotPayload{
		StructureID:    structID,
		ClassID:        classID,
		LearningAreaID: learningAreaID,
		TeacherID:      userID,
	})
	require.NoError(t, err)

	// Delete it
	err = repo.DeleteSlot(ctx, slot.ID, tenantID, schoolID)
	require.NoError(t, err)

	// Verify it's gone
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM cbc_timetable_slots WHERE id = $1`, slot.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestPgRepository_DeleteSlot_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	tenantID, schoolID, _, _ := setupTimeBlockTestData(t, pool)
	repo := newRepo(pool)

	err := repo.DeleteSlot(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

// Helper to create pointer to string
func ptr(s string) *string {
	return &s
}

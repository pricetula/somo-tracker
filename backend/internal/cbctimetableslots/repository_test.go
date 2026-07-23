package cbctimetableslots

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationsDir returns the absolute path to the migrations folder.
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

func TestPgRepository_ListEnriched(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	// Apply the full initial schema
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	// ─── Seed data ──────────────────────────────────────────────
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	schoolID := uuid.New().String()
	academicYearID := uuid.New().String()
	streamID := uuid.New().String()
	classID := uuid.New().String()
	learningAreaID := uuid.New().String()
	structureID := uuid.New().String()
	slotID := uuid.New().String()

	// 1. Tenant
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	// 2. User
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "teacher@test.com", tenantID, "John Teacher")
	require.NoError(t, err)

	// 3. School
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	// 4. Academic year (references user for created_by/updated_by)
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	// 5. Stream
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name, color)
		VALUES ($1, $2, $3, 'Blue', '#0000FF')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	// 6. Class
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active)
		VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	// 7. Learning area
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level)
		VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		learningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	// 8. Timetable structure (day 1 = Monday)
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break)
		VALUES ($1, $2, $3, $4, 1, 'Period 1', '08:00', '08:40', false)`,
		structureID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// 9. Timetable slot
	_, err = pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slotID, tenantID, schoolID, academicYearID, structureID, classID, learningAreaID, userID)
	require.NoError(t, err)

	// ─── Create repository ──────────────────────────────────────
	repo := &PgRepository{pool: pool}

	// ─── Test 1: ListEnriched without date ──────────────────────
	t.Run("no date filter", func(t *testing.T) {
		filter := SlotFilter{
			AcademicYearID: academicYearID,
			TenantID:       tenantID,
		}
		result, err := repo.ListEnriched(ctx, filter)
		require.NoError(t, err)
		require.Len(t, result, 1)

		item := result[0]
		require.Equal(t, slotID, item.ID)
		require.Equal(t, "G4 Blue", item.ClassName)
		require.Equal(t, "Period 1", item.PeriodName)
		require.Equal(t, 1, item.DayOfWeek)
		require.Equal(t, "08:00", item.StartTime)
		require.Equal(t, "08:40", item.EndTime)
		require.False(t, item.IsBreak)
		require.NotNil(t, item.LearningAreaName)
		require.Equal(t, "Mathematics", *item.LearningAreaName)
		require.NotNil(t, item.TeacherName)
		require.Equal(t, "John Teacher", *item.TeacherName)
		// Without date, session_status and skip_reason should NOT be populated
		require.Nil(t, item.SessionStatus)
		require.Nil(t, item.SkipReason)
	})

	// ─── Test 2: ListEnriched with date (Monday) ────────────────
	t.Run("with date filter matching day_of_week", func(t *testing.T) {
		// 2026-07-20 is a Monday (EXTRACT(DOW) = 1, mapped to our day_of_week 1)
		// Also, create an attendance session for this slot+date
		attendanceSessionID := uuid.New().String()
		_, err := pool.Exec(ctx, `INSERT INTO cbc_attendance_sessions (id, tenant_id, school_id, timetable_slot_id, date, status, skip_reason)
			VALUES ($1, $2, $3, $4, $5, 'SUBMITTED', NULL)`,
			attendanceSessionID, tenantID, schoolID, slotID, "2026-07-20")
		require.NoError(t, err)

		filter := SlotFilter{
			AcademicYearID: academicYearID,
			TenantID:       tenantID,
			Date:           "2026-07-20",
		}
		result, err := repo.ListEnriched(ctx, filter)
		require.NoError(t, err)
		require.Len(t, result, 1)

		item := result[0]
		require.Equal(t, slotID, item.ID)
		require.NotNil(t, item.SessionStatus)
		require.Equal(t, "SUBMITTED", *item.SessionStatus)
		require.Nil(t, item.SkipReason)
	})

	// ─── Test 3: ListEnriched with date that doesn't match ──────
	t.Run("with date filter not matching day_of_week", func(t *testing.T) {
		// 2026-07-21 is a Tuesday (DOW=2) — no structures for this day
		filter := SlotFilter{
			AcademicYearID: academicYearID,
			TenantID:       tenantID,
			Date:           "2026-07-21",
		}
		result, err := repo.ListEnriched(ctx, filter)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	// ─── Test 4: ListEnriched empty result ──────────────────────
	t.Run("no matching slots", func(t *testing.T) {
		filter := SlotFilter{
			AcademicYearID: uuid.New().String(),
			TenantID:       tenantID,
		}
		result, err := repo.ListEnriched(ctx, filter)
		require.NoError(t, err)
		require.Empty(t, result)
	})
}

func TestPgRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	// ─── Seed minimal data ──────────────────────────────────────
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	schoolID := uuid.New().String()
	academicYearID := uuid.New().String()
	streamID := uuid.New().String()
	classID := uuid.New().String()
	learningAreaID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "teacher@test.com", tenantID, "John Teacher")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name, color)
		VALUES ($1, $2, $3, 'Blue', '#0000FF')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active)
		VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level)
		VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		learningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	// Insert two structures (different period names) so slots don't collide on unique_class_slot
	structure1ID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break)
		VALUES ($1, $2, $3, $4, 1, 'Period 1', '08:00', '08:40', false)`,
		structure1ID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	structure2ID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_structures (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break)
		VALUES ($1, $2, $3, $4, 1, 'Period 2', '08:40', '09:20', false)`,
		structure2ID, tenantID, schoolID, academicYearID)
	require.NoError(t, err)

	// Insert two slots for the List test (different structures to avoid unique constraint)
	slot1ID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slot1ID, tenantID, schoolID, academicYearID, structure1ID, classID, learningAreaID, userID)
	require.NoError(t, err)

	slot2ID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, academic_year_id, structure_id, class_id, learning_area_id, teacher_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slot2ID, tenantID, schoolID, academicYearID, structure2ID, classID, learningAreaID, userID)
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	t.Run("list all slots", func(t *testing.T) {
		filter := SlotFilter{
			AcademicYearID: academicYearID,
			TenantID:       tenantID,
		}
		result, err := repo.List(ctx, filter)
		require.NoError(t, err)
		require.Len(t, result, 2)
	})

	t.Run("list with class filter", func(t *testing.T) {
		filter := SlotFilter{
			AcademicYearID: academicYearID,
			TenantID:       tenantID,
			ClassID:        classID,
		}
		result, err := repo.List(ctx, filter)
		require.NoError(t, err)
		require.Len(t, result, 2)
	})

	t.Run("list with non-matching filter", func(t *testing.T) {
		filter := SlotFilter{
			AcademicYearID: academicYearID,
			TenantID:       tenantID,
			ClassID:        uuid.New().String(),
		}
		result, err := repo.List(ctx, filter)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("list empty for unknown year", func(t *testing.T) {
		filter := SlotFilter{
			AcademicYearID: uuid.New().String(),
			TenantID:       tenantID,
		}
		result, err := repo.List(ctx, filter)
		require.NoError(t, err)
		require.Empty(t, result)
	})
}

// ─── Test helpers (following existing project pattern) ─────────────────────

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
	if err != nil {
		t.Fatalf("startPG: start container: %v", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("startPG: get host: %v", err)
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("startPG: get mapped port: %v", err)
	}

	dbURL := "postgres://somo_admin:somo_secure_password@" + host + ":" + port.Port() + "/somotracker_test?sslmode=disable"

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = c.Terminate(ctx)
		t.Fatalf("startPG: connect: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_ = c.Terminate(ctx)
	}

	return pool, cleanup
}

func applyMigration(t *testing.T, pool *pgxpool.Pool, filename string) {
	t.Helper()

	path := filepath.Join(migrationsDir(), filename)
	sql, err := os.ReadFile(path)
	require.NoError(t, err, "read migration %s", filename)

	_, err = pool.Exec(context.Background(), string(sql))
	require.NoError(t, err, "apply migration %s", filename)
}

// compile-time check that PgRepository satisfies Repository interface
var _ Repository = (*PgRepository)(nil)

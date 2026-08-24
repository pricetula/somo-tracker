package behavior

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

	cleanup := func() {
		pool.Close()
		_ = c.Terminate(ctx)
	}
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

func newRepo(pool *pgxpool.Pool) Repository {
	return NewRepository(&database.Pools{PG: pool})
}

// seedNoteDependencies inserts all FK chain records required for a behavior note
// and returns the student and timetable slot IDs.
func seedNoteDependencies(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, userID string) (studentID, slotID string) {
	t.Helper()
	ctx := context.Background()

	yearID := uuid.New().String()
	streamID := uuid.New().String()
	classID := uuid.New().String()
	learningAreaID := uuid.New().String()
	BlockID := uuid.New().String()
	studentID = uuid.New().String()
	slotID = uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by) VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id) VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classID, tenantID, schoolID, yearID, streamID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'English', 'ENG', 'Upper_Primary', 'G4')`,
		learningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender) VALUES ($1, $2, $3, 'Test Student', 'M')`,
		studentID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time) VALUES ($1, $2, $3, $4, 1, 'Period 1', '08:00', '08:40')`,
		BlockID, tenantID, schoolID, yearID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, academic_year_id, block_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slotID, tenantID, schoolID, yearID, BlockID, classID, learningAreaID, userID)
	require.NoError(t, err)

	return studentID, slotID
}

func TestPgRepository_CreateCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	// Create category
	cat, err := repo.CreateCategory(ctx, tenantID, schoolID, "Bullying", nil)
	require.NoError(t, err)
	require.NotEmpty(t, cat.ID)
	require.Equal(t, "Bullying", cat.Name)
	require.True(t, cat.IsActive)
	require.Equal(t, tenantID, cat.TenantID)

	// Create with default severity
	severity := "MINOR"
	cat2, err := repo.CreateCategory(ctx, tenantID, schoolID, "Tardiness", &severity)
	require.NoError(t, err)
	require.Equal(t, "Tardiness", cat2.Name)
	require.NotNil(t, cat2.DefaultSeverity)
	require.Equal(t, "MINOR", *cat2.DefaultSeverity)

	// List all
	cats, err := repo.ListCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, cats, 2)

	// List active only (both are active)
	active, err := repo.ListActiveCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, active, 2)
}

func TestPgRepository_ListCategories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	cat1, err := repo.CreateCategory(ctx, tenantID, schoolID, "Active", nil)
	require.NoError(t, err)

	// Manually deactivate
	_, err = pool.Exec(ctx, `UPDATE behavior_categories SET is_active = false WHERE id = $1`, cat1.ID)
	require.NoError(t, err)

	_, err = repo.CreateCategory(ctx, tenantID, schoolID, "Still Active", nil)
	require.NoError(t, err)

	// List all should include both
	all, err := repo.ListCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, all, 2)

	// List active only
	active, err := repo.ListActiveCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, "Still Active", active[0].Name)
}

func TestPgRepository_GetCategoryByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	cat, err := repo.CreateCategory(ctx, tenantID, schoolID, "Test Category", nil)
	require.NoError(t, err)

	// Get by ID
	found, err := repo.GetCategoryByID(ctx, cat.ID, tenantID)
	require.NoError(t, err)
	require.Equal(t, cat.ID, found.ID)
	require.Equal(t, "Test Category", found.Name)

	// Wrong tenant — not found
	_, err = repo.GetCategoryByID(ctx, cat.ID, uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Non-existent — not found
	_, err = repo.GetCategoryByID(ctx, uuid.New().String(), tenantID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpdateCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	cat, err := repo.CreateCategory(ctx, tenantID, schoolID, "Original", nil)
	require.NoError(t, err)

	// Update name
	newName := "Updated Name"
	updated, err := repo.UpdateCategory(ctx, cat.ID, tenantID, UpdateCategoryPayload{Name: &newName})
	require.NoError(t, err)
	require.Equal(t, "Updated Name", updated.Name)

	// Deactivate
	inactive := false
	updated, err = repo.UpdateCategory(ctx, cat.ID, tenantID, UpdateCategoryPayload{IsActive: &inactive})
	require.NoError(t, err)
	require.False(t, updated.IsActive)

	// Update non-existent — not found
	_, err = repo.UpdateCategory(ctx, uuid.New().String(), tenantID, UpdateCategoryPayload{Name: &newName})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_CreateNote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	// Create category
	cat, err := repo.CreateCategory(ctx, tenantID, schoolID, "Disruption", nil)
	require.NoError(t, err)

	// Seed FK dependencies for the note
	studentID, timetableSlotID := seedNoteDependencies(t, pool, tenantID, schoolID, userID)

	// Create note
	note, err := repo.CreateNote(ctx, tenantID, schoolID, CreateNotePayload{
		StudentID:             studentID,
		TimetableAllocationID: timetableSlotID,
		Date:                  "2026-07-15",
		CategoryID:            cat.ID,
		Description:           "Disruptive during class",
		IsUrgent:              true,
	}, userID)
	require.NoError(t, err)
	require.NotEmpty(t, note.ID)
	require.Equal(t, "Disruptive during class", note.Description)
	require.Equal(t, StatusPendingReview, note.Status)
	require.True(t, note.IsUrgent)
	require.Equal(t, userID, note.AuthoredByID)
}

func TestPgRepository_GetNoteByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	cat, err := repo.CreateCategory(ctx, tenantID, schoolID, "Test", nil)
	require.NoError(t, err)

	// Seed FK dependencies
	studentID, slotID := seedNoteDependencies(t, pool, tenantID, schoolID, userID)

	// Create a note directly
	noteID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO behavior_notes (id, tenant_id, school_id, student_id, timetable_allocation_id, date, category_id, description, authored_by_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'PENDING_REVIEW')`,
		noteID, tenantID, schoolID, studentID, slotID, "2026-07-15", cat.ID, "Test note", userID)
	require.NoError(t, err)

	found, err := repo.GetNoteByID(ctx, noteID, tenantID)
	require.NoError(t, err)
	require.Equal(t, noteID, found.ID)
	require.Equal(t, "Test note", found.Description)

	// Non-existent
	_, err = repo.GetNoteByID(ctx, uuid.New().String(), tenantID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_ReviewNote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedMinimalTenant(t, pool)

	repo := newRepo(pool)

	cat, err := repo.CreateCategory(ctx, tenantID, schoolID, "Test", nil)
	require.NoError(t, err)

	// Seed FK dependencies
	studentID, slotID := seedNoteDependencies(t, pool, tenantID, schoolID, userID)

	// Create a note directly
	noteID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO behavior_notes (id, tenant_id, school_id, student_id, timetable_allocation_id, date, category_id, description, authored_by_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'PENDING_REVIEW')`,
		noteID, tenantID, schoolID, studentID, slotID, "2026-07-15", cat.ID, "Needs review", userID)
	require.NoError(t, err)

	// Approve
	err = repo.ReviewNote(ctx, noteID, tenantID, userID, ReviewDecisionPayload{Decision: "APPROVED"})
	require.NoError(t, err)

	// Verify status
	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM behavior_notes WHERE id = $1`, noteID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "APPROVED", status)

	// Reject already-approved note — no longer PENDING_REVIEW so not found
	err = repo.ReviewNote(ctx, noteID, tenantID, userID, ReviewDecisionPayload{Decision: "REJECTED"})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

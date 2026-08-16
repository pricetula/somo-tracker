package cbcstreams

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationsDir returns the absolute path to the migrations folder.
func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	// Walk up to find backend/ directory
	dir := filepath.Dir(filename)
	for dir != "/" {
		if filepath.Base(dir) == "backend" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, "internal", "database", "migrations")
}

// startPG starts a PostgreSQL testcontainer and returns the pool + cleanup.
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

// applyMigration reads and applies a migration file.
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

func TestPgRepository_CreateStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	// Seed tenant + school + user
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Test: Create a stream
	stream, err := repo.Create(ctx, tenantID, schoolID, "Blue", "#0000FF")
	require.NoError(t, err)
	require.NotEmpty(t, stream.ID)
	require.Equal(t, "Blue", stream.Name)
	require.Equal(t, "#0000FF", stream.Color)
	require.False(t, stream.CreatedAt.IsZero())

	// Test: Duplicate name within same tenant+school is rejected
	_, err = repo.Create(ctx, tenantID, schoolID, "Blue", "#FF0000")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAlreadyExists)

	// Test: Same name in a different school is allowed
	schoolID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Kilimani', 'Private', true)`,
		schoolID2, tenantID, "School Two")
	require.NoError(t, err)

	stream2, err := repo.Create(ctx, tenantID, schoolID2, "Blue", "#00FF00")
	require.NoError(t, err)
	require.Equal(t, "Blue", stream2.Name)
}

func TestPgRepository_ListStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()
	schoolID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create 3 streams
	for _, s := range []struct{ name, color string }{
		{"Blue", "#0000FF"},
		{"Green", "#00FF00"},
		{"Red", "#FF0000"},
	} {
		_, err := repo.Create(ctx, tenantID, schoolID, s.name, s.color)
		require.NoError(t, err)
	}

	// List streams
	streams, err := repo.List(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, streams, 3)
	require.Equal(t, "Blue", streams[0].Name) // sorted by name ASC
	require.Equal(t, "Green", streams[1].Name)
	require.Equal(t, "Red", streams[2].Name)

	// List for non-existent school returns empty
	empty, err := repo.List(ctx, tenantID, uuid.New().String())
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestPgRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()
	schoolID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create a stream
	created, err := repo.Create(ctx, tenantID, schoolID, "Blue", "#0000FF")
	require.NoError(t, err)

	// Get by ID
	stream, err := repo.GetByID(ctx, created.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, created.ID, stream.ID)
	require.Equal(t, "Blue", stream.Name)

	// Wrong tenant scoping — not found
	_, err = repo.GetByID(ctx, created.ID, uuid.New().String(), schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Non-existent ID — not found
	_, err = repo.GetByID(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpdateStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()
	schoolID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create and update
	created, err := repo.Create(ctx, tenantID, schoolID, "Blue", "#0000FF")
	require.NoError(t, err)

	// Give updated_at time to advance
	time.Sleep(2 * time.Millisecond)

	updated, err := repo.Update(ctx, created.ID, tenantID, schoolID, "Sky Blue", "#87CEEB")
	require.NoError(t, err)
	require.Equal(t, "Sky Blue", updated.Name)
	require.Equal(t, "#87CEEB", updated.Color)
	require.True(t, updated.UpdatedAt.After(updated.CreatedAt), "updated_at should have advanced")

	// Update non-existent — not found
	_, err = repo.Update(ctx, uuid.New().String(), tenantID, schoolID, "Test", "#FFF")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Update to duplicate name — conflict
	_, err = repo.Create(ctx, tenantID, schoolID, "Red", "#FF0000")
	require.NoError(t, err)

	_, err = repo.Update(ctx, created.ID, tenantID, schoolID, "Red", "#FF0000")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAlreadyExists)
}

func TestPgRepository_DeleteStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()
	schoolID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create then delete
	created, err := repo.Create(ctx, tenantID, schoolID, "Blue", "#0000FF")
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ID, tenantID, schoolID)
	require.NoError(t, err)

	// Verify deleted
	_, err = repo.GetByID(ctx, created.ID, tenantID, schoolID)
	require.ErrorIs(t, err, ErrNotFound)

	// Delete non-existent — not found
	err = repo.Delete(ctx, uuid.New().String(), tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_HasActiveEnrollments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, 'System')`,
		userID, "sys@test.com", tenantID)
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create stream
	stream, err := repo.Create(ctx, tenantID, schoolID, "Blue", "#0000FF")
	require.NoError(t, err)

	// No classes → no enrollments
	active, err := repo.HasActiveEnrollments(ctx, stream.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.False(t, active)

	// Create an academic year, term, class, and student enrollment
	yearID := uuid.New().String()
	termID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', $5, $5)`,
		termID, tenantID, schoolID, yearID, userID)
	require.NoError(t, err)

	classID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classID, tenantID, schoolID, yearID, stream.ID)
	require.NoError(t, err)

	studentID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender)
		VALUES ($1, $2, $3, 'Test Student', 'M')`,
		studentID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, student_id, school_id, academic_term_id, academic_year_id, class_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE')`,
		uuid.New().String(), tenantID, studentID, schoolID, termID, yearID, classID)
	require.NoError(t, err)

	// Now there ARE active enrollments
	active, err = repo.HasActiveEnrollments(ctx, stream.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.True(t, active)

	// Delete with active enrollments is blocked
	err = repo.Delete(ctx, stream.ID, tenantID, schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStreamHasActiveEnrollments)
}

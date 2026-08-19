package teachers

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

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID string) {
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
		userID, "teacher@school.com", tenantID, "John Teacher")
	require.NoError(t, err)

	return tenantID, schoolID, userID
}

func seedTeacher(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, userID string) {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO memberships (id, tenant_id, user_id, school_id, role) VALUES ($1, $2, $3, $4, 'TEACHER')`,
		uuid.New().String(), tenantID, userID, schoolID)
	require.NoError(t, err)
}

func TestPgRepository_ListBySchool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	seedTeacher(t, pool, tenantID, schoolID, userID)

	// Create a second user without TEACHER role
	nonTeacherID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		nonTeacherID, "admin@school.com", tenantID, "Admin User")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (id, tenant_id, user_id, school_id, role) VALUES ($1, $2, $3, $4, 'SCHOOL_ADMIN')`,
		uuid.New().String(), tenantID, nonTeacherID, schoolID)
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Should only list the TEACHER role, not the SCHOOL_ADMIN
	teachers, total, err := repo.ListBySchool(ctx, tenantID, schoolID, false, 0, 50, "")
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, teachers, 1)
	require.Equal(t, "John Teacher", teachers[0].FullName)
	require.True(t, teachers[0].IsActive)
}

func TestPgRepository_ListBySchool_WithSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	seedTeacher(t, pool, tenantID, schoolID, userID)

	// Create another teacher
	userID2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID2, "jane@school.com", tenantID, "Jane Smith")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (id, tenant_id, user_id, school_id, role) VALUES ($1, $2, $3, $4, 'TEACHER')`,
		uuid.New().String(), tenantID, userID2, schoolID)
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Search by name
	teachers, total, err := repo.ListBySchool(ctx, tenantID, schoolID, false, 0, 50, "john")
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, teachers, 1)
	require.Equal(t, "John Teacher", teachers[0].FullName)
}

func TestPgRepository_ToggleActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	seedTeacher(t, pool, tenantID, schoolID, userID)

	repo := &PgRepository{pool: pool}

	// Toggle inactive
	err := repo.ToggleActive(ctx, tenantID, schoolID, userID, false)
	require.NoError(t, err)

	// Verify inactive
	var isActive bool
	err = pool.QueryRow(ctx, `SELECT is_active FROM memberships WHERE user_id = $1`, userID).Scan(&isActive)
	require.NoError(t, err)
	require.False(t, isActive)

	// Toggle back active
	err = repo.ToggleActive(ctx, tenantID, schoolID, userID, true)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `SELECT is_active FROM memberships WHERE user_id = $1`, userID).Scan(&isActive)
	require.NoError(t, err)
	require.True(t, isActive)
}

func TestPgRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	seedTeacher(t, pool, tenantID, schoolID, userID)

	repo := &PgRepository{pool: pool}

	// Delete
	err := repo.Delete(ctx, tenantID, schoolID, userID)
	require.NoError(t, err)

	// Verify membership deleted
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM memberships WHERE user_id = $1`, userID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

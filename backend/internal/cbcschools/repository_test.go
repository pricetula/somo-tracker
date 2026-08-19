package cbcschools

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

func TestPgRepository_CreateSchool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create school returns ID
	schoolID, err := repo.Create(ctx, tenantID, "Green Valley Academy")
	require.NoError(t, err)
	require.NotEmpty(t, schoolID)

	// Verify the school was created with defaults
	var name, county, subCounty, schoolType string
	var isActive bool
	err = pool.QueryRow(ctx, `SELECT name, county, sub_county, school_type, is_active FROM cbc_schools WHERE id = $1`, schoolID).
		Scan(&name, &county, &subCounty, &schoolType, &isActive)
	require.NoError(t, err)
	require.Equal(t, "Green Valley Academy", name)
	require.Equal(t, "Public", schoolType)
	require.True(t, isActive)
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

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create then retrieve
	schoolID, err := repo.Create(ctx, tenantID, "Test School")
	require.NoError(t, err)

	school, err := repo.GetByID(ctx, schoolID)
	require.NoError(t, err)
	require.Equal(t, schoolID, school.ID)
	require.Equal(t, tenantID, school.TenantID)
	require.Equal(t, "Test School", school.Name)
	require.False(t, school.CreatedAt.IsZero())

	// Non-existent — not found
	_, err = repo.GetByID(ctx, uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_ListByTenantID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantA := uuid.New().String()
	tenantB := uuid.New().String()

	for _, tid := range []string{tenantA, tenantB} {
		_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
			tid, "Tenant "+tid[:8], "slug-"+tid[:8], "stytch-"+tid[:8])
		require.NoError(t, err)
	}

	repo := &PgRepository{pool: pool}

	// Create 2 schools in tenantA
	schoolA1, err := repo.Create(ctx, tenantA, "Alpha School")
	require.NoError(t, err)
	_, err = repo.Create(ctx, tenantA, "Beta School")
	require.NoError(t, err)

	// Create 1 school in tenantB
	_, err = repo.Create(ctx, tenantB, "Gamma School")
	require.NoError(t, err)

	// List tenantA's schools
	userID := uuid.New().String()
	schools, err := repo.ListByTenantID(ctx, tenantA, userID)
	require.NoError(t, err)
	require.Len(t, schools, 2)
	require.Equal(t, "Alpha School", schools[0].Name) // sorted by name ASC
	require.Equal(t, "Beta School", schools[1].Name)
	require.Equal(t, tenantA, schools[0].TenantID)
	require.False(t, schools[0].IsMemberActiveSchool) // user has no membership

	// Delete school A1 and verify listing
	_, err = pool.Exec(ctx, `DELETE FROM cbc_schools WHERE id = $1`, schoolA1)
	require.NoError(t, err)

	schools, err = repo.ListByTenantID(ctx, tenantA, userID)
	require.NoError(t, err)
	require.Len(t, schools, 1)
	require.Equal(t, "Beta School", schools[0].Name)
}

func TestPgRepository_UpdateSchool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	schoolID, err := repo.Create(ctx, tenantID, "Old Name")
	require.NoError(t, err)

	// Update name and county
	newName := "New Name"
	newCounty := "Mombasa"
	err = repo.Update(ctx, SchoolUpdateFields{
		ID:     schoolID,
		Name:   &newName,
		County: &newCounty,
	})
	require.NoError(t, err)

	// Verify
	var name, county string
	err = pool.QueryRow(ctx, `SELECT name, county FROM cbc_schools WHERE id = $1`, schoolID).Scan(&name, &county)
	require.NoError(t, err)
	require.Equal(t, "New Name", name)
	require.Equal(t, "Mombasa", county)

	// Update is_active
	active := false
	err = repo.Update(ctx, SchoolUpdateFields{
		ID:       schoolID,
		IsActive: &active,
	})
	require.NoError(t, err)

	var isActive bool
	err = pool.QueryRow(ctx, `SELECT is_active FROM cbc_schools WHERE id = $1`, schoolID).Scan(&isActive)
	require.NoError(t, err)
	require.False(t, isActive)

	// Update non-existent — not found
	err = repo.Update(ctx, SchoolUpdateFields{ID: uuid.New().String(), Name: &newName})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)

	// Update with no fields — invalid input
	err = repo.Update(ctx, SchoolUpdateFields{ID: schoolID})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestPgRepository_DeleteSchool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)

	tenantID := uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	repo := &PgRepository{pool: pool}

	// Create then delete
	schoolID, err := repo.Create(ctx, tenantID, "To Delete")
	require.NoError(t, err)

	err = repo.Delete(ctx, schoolID)
	require.NoError(t, err)

	// Verify deleted
	_, err = repo.GetByID(ctx, schoolID)
	require.ErrorIs(t, err, ErrNotFound)

	// Delete non-existent — not found
	err = repo.Delete(ctx, uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

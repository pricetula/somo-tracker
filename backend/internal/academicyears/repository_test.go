package academicyears

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

func applyMigration(t *testing.T, pool *pgxpool.Pool, filename string) {
	t.Helper()
	path := filepath.Join(migrationsDir(), filename)
	sql, err := os.ReadFile(path)
	require.NoError(t, err, "read migration %s", filename)
	_, err = pool.Exec(context.Background(), string(sql))
	require.NoError(t, err, "apply migration %s", filename)
}

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	userID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "user@test.com", tenantID, "Test User")
	require.NoError(t, err)
	return tenantID, schoolID, userID
}

func newRepo(pool *pgxpool.Pool) *PgRepository {
	return NewRepository(&database.Pools{PG: pool})
}

func d(year int, month time.Month, day int) DateOnly {
	return DateOnly{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func TestPgRepository_CreateAndListYears(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Name:      "2026",
		StartDate: d(2026, 1, 1),
		EndDate:   d(2026, 12, 31),
		CreatedBy: userID,
		UpdatedBy: userID,
	}

	id, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	years, err := repo.ListYears(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, years, 1)
	require.Equal(t, "2026", years[0].Name)
	require.False(t, years[0].IsCurrent)

	empty, err := repo.ListYears(ctx, tenantID, uuid.New().String())
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestPgRepository_GetYearByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Name:      "2026",
		StartDate: d(2026, 1, 1),
		EndDate:   d(2026, 12, 31),
		CreatedBy: userID,
		UpdatedBy: userID,
	}

	id, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)

	found, err := repo.GetYearByID(ctx, id, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, id, found.ID)
	require.Equal(t, "2026", found.Name)

	_, err = repo.GetYearByID(ctx, id, uuid.New().String(), schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_SetCurrentYear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year1 := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2025",
		StartDate: d(2025, 1, 1), EndDate: d(2025, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	id1, err := repo.CreateYear(ctx, year1)
	require.NoError(t, err)

	year2 := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	id2, err := repo.CreateYear(ctx, year2)
	require.NoError(t, err)

	changed, err := repo.SetCurrentYear(ctx, id2, tenantID, schoolID, userID)
	require.NoError(t, err)
	require.True(t, changed)

	var isCurrent bool
	err = pool.QueryRow(ctx, `SELECT is_current FROM academic_years WHERE id = $1`, id1).Scan(&isCurrent)
	require.NoError(t, err)
	require.False(t, isCurrent)

	err = pool.QueryRow(ctx, `SELECT is_current FROM academic_years WHERE id = $1`, id2).Scan(&isCurrent)
	require.NoError(t, err)
	require.True(t, isCurrent)
}

func TestPgRepository_CreateAndListTerms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	yearID, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)

	term1 := &AcademicTerm{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		Name:           "Term 1",
		TermNumber:     1,
		StartDate:      d(2026, 1, 1),
		EndDate:        d(2026, 4, 30),
		CreatedBy:      userID,
		UpdatedBy:      userID,
	}
	termID, err := repo.CreateTerm(ctx, term1)
	require.NoError(t, err)
	require.NotEmpty(t, termID)

	terms, err := repo.ListTerms(ctx, tenantID, schoolID, nil)
	require.NoError(t, err)
	require.Len(t, terms, 1)
	require.Equal(t, "Term 1", terms[0].Name)
	require.Equal(t, 1, terms[0].TermNumber)
}

func TestPgRepository_UpdateYear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	id, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)

	update := &AcademicYear{ID: id, Name: "2026-Updated", Version: 1,
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		UpdatedBy: userID}
	err = repo.UpdateYear(ctx, update)
	require.NoError(t, err)

	found, err := repo.GetYearByID(ctx, id, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "2026-Updated", found.Name)
	require.Equal(t, 2, found.Version)
}

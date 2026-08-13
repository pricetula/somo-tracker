package health

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

func seedStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, schoolID string) string {
	studentID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, admission_number, full_name, gender, date_of_birth) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		studentID, tenantID, schoolID, "ADM001", "Test Student", "M", time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return studentID
}

func TestPgRepository_CreateAndGetIncident(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)
	studentID := seedStudent(t, ctx, pool, tenantID, schoolID)

	params := CreateIncidentParams{
		TenantID:          tenantID,
		StudentID:         studentID,
		IncidentTimestamp: time.Now(),
		Symptoms:          "Headache and fever",
		ActionTaken:       "Given paracetamol, sent home",
		LoggedBy:          userID,
	}

	id, err := repo.CreateIncident(ctx, params)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	incident, err := repo.GetIncidentByID(ctx, id, tenantID)
	require.NoError(t, err)
	require.Equal(t, "Headache and fever", incident.Symptoms)
	require.Equal(t, "Given paracetamol, sent home", incident.ActionTaken)
}

func TestPgRepository_GetIncident_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	repo := newRepo(pool)
	_, err := repo.GetIncidentByID(ctx, uuid.New().String(), uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpdateIncident(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)
	studentID := seedStudent(t, ctx, pool, tenantID, schoolID)

	id, err := repo.CreateIncident(ctx, CreateIncidentParams{
		TenantID: tenantID, StudentID: studentID,
		IncidentTimestamp: time.Now(), Symptoms: "Fever",
		ActionTaken: "Rested", LoggedBy: userID,
	})
	require.NoError(t, err)

	newSymptoms := "High fever"
	err = repo.UpdateIncident(ctx, id, tenantID, UpdateMedicalIncidentPayload{Symptoms: &newSymptoms})
	require.NoError(t, err)

	incident, err := repo.GetIncidentByID(ctx, id, tenantID)
	require.NoError(t, err)
	require.Equal(t, "High fever", incident.Symptoms)
}

func TestPgRepository_DeleteIncident(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)
	studentID := seedStudent(t, ctx, pool, tenantID, schoolID)

	id, err := repo.CreateIncident(ctx, CreateIncidentParams{
		TenantID: tenantID, StudentID: studentID,
		IncidentTimestamp: time.Now(), Symptoms: "Fever",
		ActionTaken: "Rested", LoggedBy: userID,
	})
	require.NoError(t, err)

	err = repo.DeleteIncident(ctx, id, tenantID)
	require.NoError(t, err)

	_, err = repo.GetIncidentByID(ctx, id, tenantID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_UpsertAndGetProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)
	studentID := seedStudent(t, ctx, pool, tenantID, schoolID)

	bg := "O+"
	allergies := []string{"Peanuts", "Pollen"}
	chronic := []string{"Asthma"}
	ei := "Call 911 if difficulty breathing"

	profile, err := repo.UpsertProfile(ctx, UpsertProfileParams{
		TenantID:              tenantID,
		StudentID:             studentID,
		BloodGroup:            &bg,
		Allergies:             allergies,
		ChronicConditions:     chronic,
		EmergencyInstructions: &ei,
		LoggedBy:              userID,
	})
	require.NoError(t, err)
	require.NotNil(t, profile)
	require.Equal(t, "O+", *profile.BloodGroup)

	// Get by student
	fetched, err := repo.GetProfileByStudent(ctx, studentID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "O+", *fetched.BloodGroup)
	require.Contains(t, fetched.Allergies, "Peanuts")
}

func TestPgRepository_GetProfile_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	repo := newRepo(pool)
	_, err := repo.GetProfileByStudent(ctx, uuid.New().String(), uuid.New().String())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_ListIncidentsByStudent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)
	studentID := seedStudent(t, ctx, pool, tenantID, schoolID)

	_, err := repo.CreateIncident(ctx, CreateIncidentParams{
		TenantID: tenantID, StudentID: studentID,
		IncidentTimestamp: time.Now(), Symptoms: "Fever",
		ActionTaken: "Rested", LoggedBy: userID,
	})
	require.NoError(t, err)

	items, err := repo.ListIncidentsByStudent(ctx, studentID, tenantID)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

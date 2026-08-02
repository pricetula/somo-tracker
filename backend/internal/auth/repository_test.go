package auth

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
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
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

// newRepo constructs a SqlcRepository directly. The fx-provided constructor
// (NewSqlcRepository) wires lifecycle hooks; in tests we mirror the pattern
// used by integration_suite_test.go and build the struct literally so no
// fx.Lifecycle is required.
func newRepo(pool *pgxpool.Pool) *SqlcRepository {
	return &SqlcRepository{
		pool:   pool,
		logger: zap.NewNop(),
		cfg:    config.Config{},
	}
}

var _ = (*database.Pools)(nil) // keep the import meaningful for future seed helpers

func TestPgRepository_TenantExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	repo := newRepo(pool)

	exists, err := repo.TenantExists(ctx, "stytch-"+tenantID[:8])
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = repo.TenantExists(ctx, "non-existent")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestPgRepository_GetTenantByName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID := uuid.New().String()
	expectedName := "Test School Name"
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, expectedName, "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)

	repo := newRepo(pool)

	id, orgID, err := repo.GetTenantByName(ctx, expectedName)
	require.NoError(t, err)
	require.Equal(t, tenantID, id)
	require.Equal(t, "stytch-"+tenantID[:8], orgID)

	// not found
	_, _, err = repo.GetTenantByName(ctx, "Unknown School")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_CreateUserSessionAndGetSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, _, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// Create a user session
	sessionToken := "session-token-123"
	userParams := CreateUserParams{
		Email:          "user2@test.com",
		TenantID:       tenantID,
		FullName:       "User Two",
		ExternalAuthID: "external-auth-id",
	}
	sessionParams := CreateSessionParams{
		Token:              sessionToken,
		TenantID:           tenantID,
		UserID:             userID,
		StytchMemberID:     "stytch-member-1",
		StytchOrgID:        "stytch-org-1",
		StytchSessionToken: "stytch-session-token",
		DeviceFingerprint:  "device-fingerprint",
		ExpiresAt:          time.Now().Add(time.Hour),
	}
	userID2, err := repo.CreateUserSession(ctx, userParams, sessionParams)
	require.NoError(t, err)
	require.NotEmpty(t, userID2)

	// Retrieve session by token
	session, err := repo.GetSessionByToken(ctx, sessionToken)
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, userID, session.UserID)
	require.Equal(t, tenantID, session.TenantID)
	require.Equal(t, sessionToken, session.Token)
}

func TestPgRepository_GetSessionByToken_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	repo := newRepo(pool)

	_, err := repo.GetSessionByToken(ctx, "non-existent-token")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

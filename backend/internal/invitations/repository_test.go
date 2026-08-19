package invitations

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
	// Cleanup order matters: terminate the container FIRST so any in-flight
	// pgx connections are severed (this unblocks pool.Close's waitgroup even
	// if a test forgot to roll back a transaction). Then close the pool.
	// Without this, a leaked tx hangs pool.Close until the test timeout.
	cleanup := func() {
		if termErr := c.Terminate(ctx); termErr != nil {
			t.Logf("terminate postgres container: %v", termErr)
		}
		pool.Close()
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

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tID, sID, uID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tID = uuid.New()
	sID = uuid.New()
	uID = uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tID, "Test", "slug-"+tID.String()[:8], "stytch-"+tID.String()[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		sID, tID, "Test School")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		uID, "inviter@test.com", tID, "Inviter")
	require.NoError(t, err)
	return tID, sID, uID
}

func newRepo(pool *pgxpool.Pool) *PgRepository {
	return NewRepository(&database.Pools{PG: pool})
}

func TestPgRepository_InsertAndListInvitations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	// Deferred rollback per backend AGENTS.md §6. Safe to call after Commit
	// (Commit returns ErrTxClosed which Rollback ignores); prevents a leaked
	// tx from blocking pool.Close if the test fails before committing.
	defer func() { _ = tx.Rollback(ctx) }()

	err = repo.InsertInvitation(ctx, tx, InsertInvitationParams{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Email:     "newuser@test.com",
		FullName:  "New User",
		InvitedBy: userID,
		Role:      "TEACHER",
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	tID := tenantID.String()
	sID := schoolID.String()
	invitations, total, err := repo.ListInvitations(ctx, tID, sID, ListInvitationsFilter{Offset: 0, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, invitations, 1)
	require.Equal(t, "newuser@test.com", invitations[0].Email)
}

func TestPgRepository_CheckExistingEmails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	err = repo.InsertInvitation(ctx, tx, InsertInvitationParams{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Email:     "existing@test.com",
		FullName:  "Existing User",
		InvitedBy: userID,
		Role:      "TEACHER",
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	existingUsers, existingInvites, err := repo.CheckExistingEmails(ctx, tenantID.String(), schoolID.String(), []string{"existing@test.com", "new@test.com"})
	require.NoError(t, err)
	// existingUsers: emails that already have a user account for this tenant.
	// The test only inserted an invitation (not a user) for existing@test.com,
	// so this list must be empty.
	require.Empty(t, existingUsers)
	// existingInvites: emails that already have a pending invitation for this school.
	// existing@test.com was inserted above; new@test.com was not.
	require.Contains(t, existingInvites, "existing@test.com")
	require.NotContains(t, existingInvites, "new@test.com")
}

func TestPgRepository_RevokeInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	err = repo.InsertInvitation(ctx, tx, InsertInvitationParams{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Email:     "revoke@test.com",
		FullName:  "Revoke User",
		InvitedBy: userID,
		Role:      "TEACHER",
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	invitations, total, err := repo.ListInvitations(ctx, tenantID.String(), schoolID.String(), ListInvitationsFilter{Offset: 0, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, invitations, 1)
	require.Equal(t, "pending", invitations[0].Status)

	err = repo.RevokeInvitation(ctx, invitations[0].ID, schoolID.String())
	require.NoError(t, err)

	// Default ListInvitationsFilter keeps non-expired rows regardless of status,
	// so the revoked invitation still appears in the list — but with status=revoked.
	invitations2, total2, err := repo.ListInvitations(ctx, tenantID.String(), schoolID.String(), ListInvitationsFilter{Offset: 0, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, 1, total2)
	require.Len(t, invitations2, 1)
	require.Equal(t, "revoked", invitations2[0].Status)
}

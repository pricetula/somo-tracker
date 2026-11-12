//go:build integration

package service_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"

	"somotracker/internal/identity/service"
	identityStore "somotracker/internal/identity/store"
	storegen "somotracker/internal/identity/store/gen"
)

// testEnv holds a containerized Postgres seeded with the minimum fixtures
// required to exercise session+membership resolution end to end.
type testEnv struct {
	db        *sql.DB
	countryID uuid.UUID
	systemsID uuid.UUID
}

func setupIntegrationDB(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	pgC, err := postgres.RunContainer(ctx, testcontainers.WithImage("postgres:18"),
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pgC.Terminate(ctx) })

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	goose.SetTableName("goose_db_version")
	if err := goose.UpContext(ctx, db, "../../../migrations"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	env := &testEnv{db: db}
	env.countryID = uuid.New()
	env.systemsID = uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES ($1, $2, $3)`,
		env.countryID, "Kenya", "KE")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES ($1, $2, $3)`,
		env.systemsID, env.countryID, "CBE")
	require.NoError(t, err)
	return env
}

func (e *testEnv) insertTenant(t *testing.T, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := e.db.ExecContext(context.Background(), `INSERT INTO tenants (id, name) VALUES ($1, $2)`, id, name)
	require.NoError(t, err)
	return id
}

func (e *testEnv) insertUser(t *testing.T, tenantID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := e.db.ExecContext(context.Background(),
		`INSERT INTO users (id, tenant_id, email_normalized, name) VALUES ($1, $2, $3, $4)`,
		id, tenantID, fmt.Sprintf("%s@example.com", name), name)
	require.NoError(t, err)
	return id
}

func (e *testEnv) insertSchool(t *testing.T, tenantID uuid.UUID, name, code string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := e.db.ExecContext(context.Background(),
		`INSERT INTO schools (id, tenant_id, name, code, country_id, education_system_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, tenantID, name, code, e.countryID, e.systemsID)
	require.NoError(t, err)
	return id
}

func (e *testEnv) insertMembership(t *testing.T, userID, schoolID uuid.UUID, role string, active bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := e.db.ExecContext(context.Background(),
		`INSERT INTO memberships (id, user_id, school_id, role, is_active) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, schoolID, role, active)
	require.NoError(t, err)
	return id
}

func (e *testEnv) insertSession(t *testing.T, userID, tenantID uuid.UUID, expiresAt time.Time, revoked bool) ([]byte, uuid.UUID) {
	t.Helper()
	id := uuid.New()
	token := uuid.NewString()
	hash := sha256.Sum256([]byte(token))
	if revoked {
		_, err := e.db.ExecContext(context.Background(),
			`INSERT INTO app_session (id, user_id, tenant_id, token_hash, expires_at, revoked_at) VALUES ($1, $2, $3, $4, $5, NOW())`,
			id, userID, tenantID, hash[:], expiresAt)
		require.NoError(t, err)
		return hash[:], id
	}
	_, err := e.db.ExecContext(context.Background(),
		`INSERT INTO app_session (id, user_id, tenant_id, token_hash, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, tenantID, hash[:], expiresAt)
	require.NoError(t, err)
	return hash[:], id
}

func newIntegrationService(t *testing.T, db *sql.DB) *service.Service {
	t.Helper()
	queries := storegen.New(db)
	repo := identityStore.New(queries)
	svc, err := service.New(repo, nil, db, "")
	require.NoError(t, err)
	return svc
}

// TestValidateSessionWithMemberships_Integration covers AC-1, AC-3, AC-4 and AC-5
// against a real Postgres database: single joined query, live revocation/expiry,
// and tenant isolation of the membership join.
func TestValidateSessionWithMemberships_Integration(t *testing.T) {
	env := setupIntegrationDB(t)
	ctx := context.Background()

	tenantA := env.insertTenant(t, "Tenant A")
	tenantB := env.insertTenant(t, "Tenant B")
	schoolA := env.insertSchool(t, tenantA, "School A", "SCH-A")
	schoolB := env.insertSchool(t, tenantB, "School B", "SCH-B")

	svc := newIntegrationService(t, env.db)

	t.Run("happy path: session + active membership in single query", func(t *testing.T) {
		userA := env.insertUser(t, tenantA, "teacher-a")
		env.insertMembership(t, userA, schoolA, "TEACHER", true)
		hash, _ := env.insertSession(t, userA, tenantA, time.Now().Add(1*time.Hour), false)

		sess, mems, err := svc.ValidateSessionWithMemberships(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, userA, sess.UserID)
		require.Equal(t, tenantA, sess.TenantID)
		require.Len(t, mems, 1)
		require.Equal(t, schoolA, mems[0].SchoolID)
		require.Equal(t, "TEACHER", mems[0].Role)
		require.True(t, mems[0].IsActive)
	})

	t.Run("revoked session returns ErrSessionNotFound even with active membership", func(t *testing.T) {
		userR := env.insertUser(t, tenantA, "revoked")
		env.insertMembership(t, userR, schoolA, "TEACHER", true)
		hash, _ := env.insertSession(t, userR, tenantA, time.Now().Add(1*time.Hour), true)

		sess, mems, err := svc.ValidateSessionWithMemberships(ctx, hash)
		require.ErrorIs(t, err, service.ErrSessionNotFound)
		require.Nil(t, sess)
		require.Nil(t, mems)
	})

	t.Run("expired session returns ErrSessionNotFound", func(t *testing.T) {
		userE := env.insertUser(t, tenantA, "expired")
		hash, _ := env.insertSession(t, userE, tenantA, time.Now().Add(-1*time.Hour), false)

		sess, mems, err := svc.ValidateSessionWithMemberships(ctx, hash)
		require.ErrorIs(t, err, service.ErrSessionNotFound)
		require.Nil(t, sess)
		require.Nil(t, mems)
	})

	t.Run("cross-tenant school membership is not returned", func(t *testing.T) {
		userX := env.insertUser(t, tenantA, "cross-tenant")
		// Membership points at a school in tenant B; the query must exclude it.
		env.insertMembership(t, userX, schoolB, "TEACHER", true)
		hash, _ := env.insertSession(t, userX, tenantA, time.Now().Add(1*time.Hour), false)

		sess, mems, err := svc.ValidateSessionWithMemberships(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, userX, sess.UserID)
		require.Empty(t, mems)
	})

	t.Run("no membership returns session with empty memberships", func(t *testing.T) {
		userN := env.insertUser(t, tenantA, "no-membership")
		hash, _ := env.insertSession(t, userN, tenantA, time.Now().Add(1*time.Hour), false)

		sess, mems, err := svc.ValidateSessionWithMemberships(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, userN, sess.UserID)
		require.Empty(t, mems)
	})

	t.Run("session tenant not matching user tenant returns ErrSessionNotFound", func(t *testing.T) {
		userY := env.insertUser(t, tenantB, "mismatched")
		// Session claims tenant A but user belongs to tenant B -> inner join fails closed.
		hash, _ := env.insertSession(t, userY, tenantA, time.Now().Add(1*time.Hour), false)

		sess, mems, err := svc.ValidateSessionWithMemberships(ctx, hash)
		require.ErrorIs(t, err, service.ErrSessionNotFound)
		require.Nil(t, sess)
		require.Nil(t, mems)
	})
}

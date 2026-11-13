//go:build integration

package handler

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gofiber/fiber/v3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/jackc/pgx/v5/stdlib"

	"somotracker/internal/identity/service"
	identityStore "somotracker/internal/identity/store"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
)

// meTestEnv holds everything needed to exercise the /me, /schools and
// /me/active-school endpoints against a real Postgres database.
type meTestEnv struct {
	db         *sql.DB
	t          *testing.T
	svc        *service.Service
	h          *Handler
	app        *fiber.App
	countryID  uuid.UUID
	systemsID  uuid.UUID
	tenantID   uuid.UUID
	userID     uuid.UUID
	schoolAID  uuid.UUID
	schoolBID  uuid.UUID
	sessionTok string // raw session token (for cookie)
	tokenHash  []byte // SHA-256 of session token
	cfg        *config.Config
}

// setupMeIntegrationDB starts a containerized Postgres, runs migrations,
// seeds reference data + tenant + user + schools + memberships + session,
// and wires up the handler + Fiber app.
func setupMeIntegrationDB(t *testing.T) *meTestEnv {
	t.Helper()
	ctx := context.Background()

	// 1. Start Postgres via testcontainers
	pgC, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:18"),
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "start postgres container")
	t.Cleanup(func() { _ = pgC.Terminate(ctx) })

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "connection string")

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err, "open db")
	t.Cleanup(func() { _ = db.Close() })

	// 2. Run migrations
	goose.SetTableName("goose_db_version")
	require.NoError(t, goose.UpContext(ctx, db, "../../../migrations"), "migrate up")

	// 3. Create env and seed reference data
	env := &meTestEnv{db: db, t: t}
	env.countryID = uuid.New()
	env.systemsID = uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES ($1, $2, $3)`,
		env.countryID, "Kenya", "KE")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES ($1, $2, $3)`,
		env.systemsID, env.countryID, "CBE")
	require.NoError(t, err)

	// 4. Create tenant first (needed for RLS GUC)
	env.tenantID = uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO tenants (id, name) VALUES ($1, $2)`,
		env.tenantID, "Test Tenant")
	require.NoError(t, err)

	// 5. Disable RLS for the test (RLS behavior is covered by
	// migrator_integration_test.go; here we just want to test the handler
	// logic without connection-pool GUC complications).
	_, err = db.ExecContext(ctx, "ALTER TABLE schools DISABLE ROW LEVEL SECURITY")
	_, err = db.ExecContext(ctx, "ALTER TABLE memberships DISABLE ROW LEVEL SECURITY")
	_, err = db.ExecContext(ctx, "ALTER TABLE app_session DISABLE ROW LEVEL SECURITY")
	_, err = db.ExecContext(ctx, "ALTER TABLE users DISABLE ROW LEVEL SECURITY")
	require.NoError(t, err, "disable RLS for test")

	// 6. Create config for test environment
	env.cfg = &config.Config{
		Env:           "test",
		CookieDomain:  "",
	}

	// 7. Wire service + handler
	queries := storegen.New(db)
	repo := identityStore.New(queries)
	svc, err := service.New(repo, nil, db, "")
	require.NoError(t, err)
	env.svc = svc

	h := New(svc, env.cfg, nil)
	env.h = h

	// 8. Wire Fiber app with authenticated routes
	app := fiber.New()
	api := app.Group("/api/v1")
	protected := api.Group("")
	protected.Use(func(c fiber.Ctx) error {
		// No-op middleware: tests inject session via cookie directly
		return c.Next()
	})
	h.RegisterAuthenticatedRoutes(protected)
	env.app = app

	// 9. Seed user, schools, memberships, session (tenant already created)
	env.seedRemainingData(ctx)

	return env
}

func (e *meTestEnv) seedRemainingData(ctx context.Context) {
	// User
	e.userID = uuid.New()
	var err error
	_, err = e.db.ExecContext(ctx,
		`INSERT INTO users (id, tenant_id, email_normalized, name) VALUES ($1, $2, $3, $4)`,
		e.userID, e.tenantID, "teacher@example.com", "Teacher User")
	require.NoError(e.t, err)

	// School A (will be active)
	e.schoolAID = uuid.New()
	_, err = e.db.ExecContext(ctx,
		`INSERT INTO schools (id, tenant_id, name, code, country_id, education_system_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		e.schoolAID, e.tenantID, "School A", "SCH-A", e.countryID, e.systemsID)
	require.NoError(e.t, err)

	// School B (inactive)
	e.schoolBID = uuid.New()
	_, err = e.db.ExecContext(ctx,
		`INSERT INTO schools (id, tenant_id, name, code, country_id, education_system_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		e.schoolBID, e.tenantID, "School B", "SCH-B", e.countryID, e.systemsID)
	require.NoError(e.t, err)

	// Memberships: School A active, School B inactive
	_, err = e.db.ExecContext(ctx,
		`INSERT INTO memberships (id, user_id, school_id, role, is_active) VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), e.userID, e.schoolAID, "TEACHER", true)
	require.NoError(e.t, err)
	_, err = e.db.ExecContext(ctx,
		`INSERT INTO memberships (id, user_id, school_id, role, is_active) VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), e.userID, e.schoolBID, "TEACHER", false)
	require.NoError(e.t, err)

	// Session
	e.sessionTok = uuid.NewString()
	hash := sha256.Sum256([]byte(e.sessionTok))
	e.tokenHash = hash[:]
	sessionID := uuid.New()
	expiresAt := time.Now().Add(1 * time.Hour)
	_, err = e.db.ExecContext(ctx,
		`INSERT INTO app_session (id, user_id, tenant_id, token_hash, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		sessionID, e.userID, e.tenantID, e.tokenHash, expiresAt)
	require.NoError(e.t, err)
}

// cookieHeader returns a Cookie header value for the session token.
// In test env, SessionCookieName() returns "session_token_test".
func (e *meTestEnv) cookieHeader() string {
	return fmt.Sprintf("session_token_test=%s", e.sessionTok)
}

// req creates an HTTP request with the session cookie.
func (e *meTestEnv) req(method, path string, body string) *http.Request {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req, err := http.NewRequest(method, "http://test"+path, bodyReader)
	require.NoError(e.t, err)
	req.Header.Set("Cookie", e.cookieHeader())
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// testRoundTrip sends a request through the Fiber app and returns the response.
func (e *meTestEnv) testRoundTrip(req *http.Request) *http.Response {
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	require.NoError(e.t, err)
	return resp
}

// ---------------------------------------------------------------------------
// GET /api/v1/me
// ---------------------------------------------------------------------------

func TestMeIntegration_HappyPath(t *testing.T) {
	env := setupMeIntegrationDB(t)

	resp := env.testRoundTrip(env.req(http.MethodGet, "/api/v1/me", ""))
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK")

	var me struct {
		UserID       string     `json:"user_id"`
		FullName     string     `json:"full_name"`
		Email        string     `json:"email"`
		Role         string     `json:"role"`
		ActiveSchool *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"active_school"`
		Schools []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"schools"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&me))

	require.Equal(t, env.userID.String(), me.UserID)
	require.Equal(t, "Teacher User", me.FullName)
	require.Equal(t, "teacher@example.com", me.Email)
	require.Equal(t, "TEACHER", me.Role)

	// Active school should be School A (is_active = true)
	require.NotNil(t, me.ActiveSchool)
	require.Equal(t, env.schoolAID.String(), me.ActiveSchool.ID)
	require.Equal(t, "School A", me.ActiveSchool.Name)

	// Both schools should be listed
	require.Len(t, me.Schools, 2)
	schoolIDs := map[string]bool{}
	for _, s := range me.Schools {
		schoolIDs[s.ID] = true
	}
	require.True(t, schoolIDs[env.schoolAID.String()], "school A should be listed")
	require.True(t, schoolIDs[env.schoolBID.String()], "school B should be listed")
}

func TestMeIntegration_NoSessionCookie(t *testing.T) {
	env := setupMeIntegrationDB(t)

	req, err := http.NewRequest(http.MethodGet, "http://test/api/v1/me", strings.NewReader(""))
	require.NoError(t, err)
	// No cookie set

	resp := env.testRoundTrip(req)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 UNAUTHENTICATED")
}

func TestMeIntegration_InvalidSession(t *testing.T) {
	env := setupMeIntegrationDB(t)

	// Use a token that doesn't exist in the database
	req, err := http.NewRequest(http.MethodGet, "http://test/api/v1/me", strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("Cookie", "session_token_test=nonexistent-token")

	resp := env.testRoundTrip(req)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 UNAUTHENTICATED")
}

// ---------------------------------------------------------------------------
// GET /api/v1/schools
// ---------------------------------------------------------------------------

func TestListSchoolsIntegration_HappyPath(t *testing.T) {
	env := setupMeIntegrationDB(t)

	resp := env.testRoundTrip(env.req(http.MethodGet, "/api/v1/schools", ""))
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK")

	var schools []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&schools))

	require.Len(t, schools, 2)
	schoolIDs := map[string]bool{}
	for _, s := range schools {
		schoolIDs[s.ID] = true
	}
	require.True(t, schoolIDs[env.schoolAID.String()], "school A should be listed")
	require.True(t, schoolIDs[env.schoolBID.String()], "school B should be listed")
}

func TestListSchoolsIntegration_NoSessionCookie(t *testing.T) {
	env := setupMeIntegrationDB(t)

	req, err := http.NewRequest(http.MethodGet, "http://test/api/v1/schools", strings.NewReader(""))
	require.NoError(t, err)

	resp := env.testRoundTrip(req)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 UNAUTHENTICATED")
}

// ---------------------------------------------------------------------------
// PUT /api/v1/me/active-school
// ---------------------------------------------------------------------------

func TestSwitchActiveSchoolIntegration_HappyPath(t *testing.T) {
	env := setupMeIntegrationDB(t)
	ctx := context.Background()

	// Switch active school from School A to School B
	body := fmt.Sprintf(`{"school_id": "%s"}`, env.schoolBID.String())
	resp := env.testRoundTrip(env.req(http.MethodPut, "/api/v1/me/active-school", body))
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK")

	var result struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.True(t, result.Success, "expected success=true")

	// Verify the membership was updated in the database
	var isActiveA, isActiveB bool
	err := env.db.QueryRowContext(ctx,
		`SELECT is_active FROM memberships WHERE user_id = $1 AND school_id = $2`,
		env.userID, env.schoolAID).Scan(&isActiveA)
	require.NoError(t, err)
	require.False(t, isActiveA, "school A should no longer be active")

	err = env.db.QueryRowContext(ctx,
		`SELECT is_active FROM memberships WHERE user_id = $1 AND school_id = $2`,
		env.userID, env.schoolBID).Scan(&isActiveB)
	require.NoError(t, err)
	require.True(t, isActiveB, "school B should now be active")

	// Now /me should report School B as active
	resp2 := env.testRoundTrip(env.req(http.MethodGet, "/api/v1/me", ""))
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	var me struct {
		ActiveSchool *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"active_school"`
	}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&me))
	require.NotNil(t, me.ActiveSchool)
	require.Equal(t, env.schoolBID.String(), me.ActiveSchool.ID, "active school should now be School B")
	require.Equal(t, "School B", me.ActiveSchool.Name)
}

func TestSwitchActiveSchoolIntegration_InvalidSchoolID(t *testing.T) {
	env := setupMeIntegrationDB(t)

	body := `{"school_id": "not-a-uuid"}`
	resp := env.testRoundTrip(env.req(http.MethodPut, "/api/v1/me/active-school", body))
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 BAD_REQUEST")
}

func TestSwitchActiveSchoolIntegration_UnknownSchoolID(t *testing.T) {
	env := setupMeIntegrationDB(t)

	// UUID that doesn't exist in the database
	unknownID := uuid.New()
	body := fmt.Sprintf(`{"school_id": "%s"}`, unknownID.String())
	resp := env.testRoundTrip(env.req(http.MethodPut, "/api/v1/me/active-school", body))
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, "expected 500 INTERNAL_ERROR for unknown school")
}

func TestSwitchActiveSchoolIntegration_NoSessionCookie(t *testing.T) {
	env := setupMeIntegrationDB(t)

	body := fmt.Sprintf(`{"school_id": "%s"}`, env.schoolAID.String())
	req, err := http.NewRequest(http.MethodPut, "http://test/api/v1/me/active-school", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	// No cookie set

	resp := env.testRoundTrip(req)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 UNAUTHENTICATED")
}

// ---------------------------------------------------------------------------
// Role-based visibility via /me (verify role comes through correctly)
// ---------------------------------------------------------------------------

func TestMeIntegration_RoleReflected(t *testing.T) {
	env := setupMeIntegrationDB(t)

	resp := env.testRoundTrip(env.req(http.MethodGet, "/api/v1/me", ""))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var me struct {
		Role string `json:"role"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&me))
	require.Equal(t, "TEACHER", me.Role)
}

// ---------------------------------------------------------------------------
// Error body format (BE-025 compliance)
// ---------------------------------------------------------------------------

func TestMeIntegration_ErrorBodyFormat(t *testing.T) {
	env := setupMeIntegrationDB(t)

	req, err := http.NewRequest(http.MethodGet, "http://test/api/v1/me", strings.NewReader(""))
	require.NoError(t, err)
	// No cookie -> 401

	resp := env.testRoundTrip(req)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var errBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
	require.Equal(t, string(apperr.UNAUTHENTICATED), errBody.Code)
	require.NotEmpty(t, errBody.Message)
}

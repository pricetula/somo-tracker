package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// Helper to resolve migrations path matching repository pattern
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

// Spuns up Postgres container and miniredis server to build concrete *database.Pools
func setupTestPools(t *testing.T) (*database.Pools, *miniredis.Miniredis, func()) {
	t.Helper()
	ctx := context.Background()

	// 1. Miniredis for real *redis.Client
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 2. Testcontainers Postgres for real *pgxpool.Pool
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

	pools := &database.Pools{
		PG:    pool,
		Redis: redisClient,
	}

	cleanup := func() {
		pool.Close()
		mr.Close()
		_ = c.Terminate(ctx)
	}

	return pools, mr, cleanup
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

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// testResolverCfg returns a non-production config so C5 device-fingerprint
// enforcement stays disabled for the base subtests; the C5 cases construct
// their own production config.
func testResolverCfg() config.Config {
	return config.Config{AppEnv: "test"}
}

func TestSessionResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pools, mr, cleanup := setupTestPools(t)
	defer cleanup()

	// Apply database schema
	applyAllMigrations(t, pools.PG)

	t.Run("Ignore non-api routes", func(t *testing.T) {
		app := fiber.New()
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/health", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Cookie", "somo_sid=any-token")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Valid session from Redis cache hit", func(t *testing.T) {
		rawToken := "valid-session-token"
		tokenHash := hashToken(rawToken)
		cacheKey := "session:" + tokenHash

		expectedSession := SessionInfo{
			UserID:   "user-123",
			TenantID: "tenant-456",
			Role:     "TEACHER",
		}
		sessBytes, _ := json.Marshal(expectedSession)
		_ = mr.Set(cacheKey, string(sessBytes))

		app := fiber.New()
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/api/v1/user", func(c *fiber.Ctx) error {
			sess, ok := c.Locals("session").(*SessionInfo)
			if !ok || sess == nil {
				return c.SendStatus(http.StatusUnauthorized)
			}
			return c.JSON(sess)
		})

		req := httptest.NewRequest("GET", "/api/v1/user", nil)
		req.Header.Set("Cookie", "somo_sid="+rawToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Redis cache miss, DB hit (caches result in Redis)", func(t *testing.T) {
		ctx := context.Background()
		rawToken := "uncached-token"
		tokenHash := hashToken(rawToken)
		cacheKey := "session:" + tokenHash

		now := time.Now()
		userID := uuid.New().String()
		tenantID := uuid.New().String()
		schoolID := uuid.New().String()

		// FK chain: tenants → cbc_schools → users → sessions/memberships.
		_, err := pools.PG.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, stytch_org_id)
			VALUES ($1, $2, $3, $4)
		`, tenantID, "Cache Miss Tenant", "cache-miss-"+tenantID, "StytchOrgID-"+tenantID)
		require.NoError(t, err)

		_, err = pools.PG.Exec(ctx, `
			INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
			VALUES ($1, $2, 'Cache Miss School', 'Default County', 'Default Sub-County', 'Public')
		`, schoolID, tenantID)
		require.NoError(t, err)

		_, err = pools.PG.Exec(ctx, `
			INSERT INTO users (id, email, tenant_id, full_name)
			VALUES ($1, $2, $3, 'Cache Miss User')
		`, userID, "uncached@example.com", tenantID)
		require.NoError(t, err)

		// token is kept in the seed INSERT for readability; the column is now
		// nullable and lookups go through token_hash only (C1).
		_, err = pools.PG.Exec(ctx, `
			INSERT INTO sessions (token, token_hash, user_id, tenant_id, stytch_member_id, stytch_org_id, expires_at)
			VALUES($1, $2, $3, $4, $5, $6, $7)
		`, rawToken, tokenHash, userID, tenantID, "StytchMemberID", "StytchOrgID", now.Add(24*time.Hour))
		require.NoError(t, err)

		_, err = pools.PG.Exec(ctx, `
				INSERT INTO memberships (tenant_id, user_id, school_id, role, is_active)
				VALUES ($1, $2, $3, 'SCHOOL_ADMIN', true);
			`, tenantID, userID, schoolID)
		require.NoError(t, err)

		app := fiber.New()
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/api/v1/test", func(c *fiber.Ctx) error {
			sess := c.Locals("session").(*SessionInfo)
			return c.SendString(sess.Role)
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Cookie", "somo_sid="+rawToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify write-through cache: session key now present in miniredis
		assert.Eventually(t, func() bool {
			return mr.Exists(cacheKey)
		}, 2*time.Second, 50*time.Millisecond)
	})

	t.Run("DB miss triggers Negative Caching in Redis", func(t *testing.T) {
		rawToken := "non-existent-token"
		tokenHash := hashToken(rawToken)
		cacheKey := "session:" + tokenHash

		app := fiber.New()
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/api/v1/test", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Cookie", "somo_sid="+rawToken)

		_, err := app.Test(req)
		require.NoError(t, err)

		// Verify negative cache write ("INVALID") to miniredis
		assert.Eventually(t, func() bool {
			val, _ := mr.Get(cacheKey)
			return val == "INVALID"
		}, 2*time.Second, 50*time.Millisecond)
	})

	t.Run("Populates active_school_id and school_id locals from cookie", func(t *testing.T) {
		app := fiber.New()
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/api/v1/school", func(c *fiber.Ctx) error {
			activeSchool := c.Locals("active_school_id").(string)
			school := c.Locals("school_id").(string)
			return c.SendString(activeSchool + ":" + school)
		})

		req := httptest.NewRequest("GET", "/api/v1/school", nil)
		req.Header.Set("Cookie", "somo_school_id=school-999")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Session without membership is forbidden (B3)", func(t *testing.T) {
		ctx := context.Background()
		rawToken := "no-membership-token"
		tokenHash := hashToken(rawToken)

		now := time.Now()
		userID := uuid.New().String()
		tenantID := uuid.New().String()

		_, err := pools.PG.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, stytch_org_id)
			VALUES ($1, $2, $3, $4)
		`, tenantID, "No Membership Tenant", "no-membership-"+tenantID, "StytchOrgID-"+tenantID)
		require.NoError(t, err)

		_, err = pools.PG.Exec(ctx, `
			INSERT INTO users (id, email, tenant_id, full_name)
			VALUES ($1, $2, $3, 'No Membership User')
		`, userID, "nomember@example.com", tenantID)
		require.NoError(t, err)

		_, err = pools.PG.Exec(ctx, `
			INSERT INTO sessions (token, token_hash, user_id, tenant_id, stytch_member_id, stytch_org_id, expires_at)
			VALUES($1, $2, $3, $4, $5, $6, $7)
		`, rawToken, tokenHash, userID, tenantID, "StytchMemberID", "StytchOrgID", now.Add(24*time.Hour))
		require.NoError(t, err)
		// NOTE: deliberately no memberships row.

		// Mirror the canonical error handler wired in cmd/api/main.go.
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				if errors.Is(err, fiber.ErrNotFound) || errors.Is(err, fiber.ErrMethodNotAllowed) {
					return fiber.DefaultErrorHandler(c, err)
				}
				return HTTPError(c, err)
			},
		})
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/api/v1/teacher-only", func(c *fiber.Ctx) error {
			return c.SendString("should not reach here")
		})

		req := httptest.NewRequest("GET", "/api/v1/teacher-only", nil)
		req.Header.Set("Cookie", "somo_sid="+rawToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode,
			"a session with zero memberships must not be granted TEACHER; it must be forbidden")
	})

	t.Run("Missing session is unauthorized (401)", func(t *testing.T) {
		// Same canonical error handler as production.
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				if errors.Is(err, fiber.ErrNotFound) || errors.Is(err, fiber.ErrMethodNotAllowed) {
					return fiber.DefaultErrorHandler(c, err)
				}
				return HTTPError(c, err)
			},
		})
		app.Use(NewSessionResolver(pools, testResolverCfg()))
		app.Get("/api/v1/whatever", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/api/v1/whatever", nil)
		req.Header.Set("Cookie", "somo_sid=token-that-does-not-exist")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"an unknown/expired session must map to 401, not 404 or 500")
	})

	// seedSchoolContext inserts a tenant, two schools (A, B), a user, a session
	// row (device fingerprint = fp-ctx), and memberships for the labelled
	// schools listed in memberOf. Returns the two school IDs.
	seedSchoolContext := func(t *testing.T, rawToken string, memberOf ...string) (schoolA, schoolB string) {
		t.Helper()
		ctx := context.Background()
		userID := uuid.New().String()
		tenantID := uuid.New().String()
		schoolA = uuid.New().String()
		schoolB = uuid.New().String()

		_, err := pools.PG.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, stytch_org_id)
			VALUES ($1, $2, $3, $4)
		`, tenantID, "School Context Tenant", "school-ctx-"+tenantID, "StytchOrgID-"+tenantID)
		require.NoError(t, err)

		for _, schoolID := range []string{schoolA, schoolB} {
			_, err = pools.PG.Exec(ctx, `
				INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
				VALUES ($1, $2, $3, 'Default County', 'Default Sub-County', 'Public')
			`, schoolID, tenantID, schoolID[:8])
			require.NoError(t, err)
		}

		_, err = pools.PG.Exec(ctx, `
			INSERT INTO users (id, email, tenant_id, full_name)
			VALUES ($1, $2, $3, 'School Context User')
		`, userID, "schoolctx@example.com", tenantID)
		require.NoError(t, err)

		_, err = pools.PG.Exec(ctx, `
			INSERT INTO sessions (token, token_hash, user_id, tenant_id, stytch_member_id, stytch_org_id, device_fingerprint, expires_at)
			VALUES ($1, $2, $3, $4, 'StytchMemberID', 'StytchOrgID', 'fp-ctx', $5)
		`, rawToken, hashToken(rawToken), userID, tenantID, time.Now().Add(24*time.Hour))
		require.NoError(t, err)

		schoolByLabel := map[string]string{"A": schoolA, "B": schoolB}
		for _, label := range memberOf {
			_, err = pools.PG.Exec(ctx, `
				INSERT INTO memberships (tenant_id, user_id, school_id, role, is_active)
				VALUES ($1, $2, $3, 'SCHOOL_ADMIN', true)
			`, tenantID, userID, schoolByLabel[label])
			require.NoError(t, err)
		}
		return schoolA, schoolB
	}

	// schoolContextApp resolves a session and echoes the resolved school ID.
	schoolContextApp := func(cfg config.Config) *fiber.App {
		app := fiber.New()
		app.Use(NewSessionResolver(pools, cfg))
		app.Get("/api/v1/ctx", func(c *fiber.Ctx) error {
			school, _ := c.Locals("active_school_id").(string)
			return c.SendString(school)
		})
		return app
	}

	// requestTo sends a GET to the school-context app with optional cookies.
	requestTo := func(t *testing.T, app *fiber.App, cookies ...string) string {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/v1/ctx", nil)
		if len(cookies) > 0 {
			req.Header.Set("Cookie", strings.Join(cookies, "; "))
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(body)
	}

	t.Run("C2: forged school cookie falls back to membership school", func(t *testing.T) {
		rawToken := "c2-forged-cookie-token"
		schoolA, schoolB := seedSchoolContext(t, rawToken, "A") // member of A only
		app := schoolContextApp(testResolverCfg())

		got := requestTo(t, app, "somo_sid="+rawToken, SchoolIDCookieName+"="+schoolB)
		assert.Equal(t, schoolA, got,
			"a forged cookie naming an unowned school must NOT scope the request")
	})

	t.Run("C2: no cookie uses the DB active school", func(t *testing.T) {
		rawToken := "c2-no-cookie-token"
		schoolA, _ := seedSchoolContext(t, rawToken, "A")
		app := schoolContextApp(testResolverCfg())

		got := requestTo(t, app, "somo_sid="+rawToken)
		assert.Equal(t, schoolA, got, "without a cookie the DB active school must be used")
	})

	t.Run("C2: authorized school cookie is honored", func(t *testing.T) {
		rawToken := "c2-authorized-cookie-token"
		schoolA, schoolB := seedSchoolContext(t, rawToken, "A", "B")
		// Pin the DB active school to A so honoring cookie B proves the cookie
		// selection (not just the DB default) is what wins.
		_, err := pools.PG.Exec(context.Background(), `
			INSERT INTO member_active_school (user_id, tenant_id, school_id)
			VALUES ((SELECT user_id FROM sessions WHERE token_hash = $1),
			        (SELECT tenant_id FROM sessions WHERE token_hash = $1), $2)
		`, hashToken(rawToken), schoolA)
		require.NoError(t, err)

		app := schoolContextApp(testResolverCfg())
		got := requestTo(t, app, "somo_sid="+rawToken, SchoolIDCookieName+"="+schoolB)
		assert.Equal(t, schoolB, got,
			"a cookie naming an authorized membership school must be honored")
	})

	// deviceApp builds an app that presents a fixed device fingerprint and
	// resolves the session with the canonical error handler (so sentinels map
	// to real status codes).
	deviceApp := func(cfg config.Config, presentedFP string) *fiber.App {
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				if errors.Is(err, fiber.ErrNotFound) || errors.Is(err, fiber.ErrMethodNotAllowed) {
					return fiber.DefaultErrorHandler(c, err)
				}
				return HTTPError(c, err)
			},
		})
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("device_fingerprint", presentedFP)
			return c.Next()
		})
		app.Use(NewSessionResolver(pools, cfg))
		app.Get("/api/v1/ctx", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})
		return app
	}

	requestStatus := func(t *testing.T, app *fiber.App, rawToken string) int {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/v1/ctx", nil)
		req.Header.Set("Cookie", "somo_sid="+rawToken)
		resp, err := app.Test(req)
		require.NoError(t, err)
		return resp.StatusCode
	}

	t.Run("C5 production: mismatched device fingerprint is rejected", func(t *testing.T) {
		rawToken := "c5-mismatch-token"
		seedSchoolContext(t, rawToken, "A") // session stores fingerprint fp-ctx

		app := deviceApp(config.Config{AppEnv: "production", EnforceDeviceFingerprint: true}, "fp-evil")
		assert.Equal(t, http.StatusUnauthorized, requestStatus(t, app, rawToken),
			"a session resumed from a different device must be rejected when enforcement is enabled")
	})

	t.Run("C5 production: matching fingerprint is allowed", func(t *testing.T) {
		rawToken := "c5-match-token"
		seedSchoolContext(t, rawToken, "A")

		app := deviceApp(config.Config{AppEnv: "production", EnforceDeviceFingerprint: true}, "fp-ctx")
		assert.Equal(t, http.StatusOK, requestStatus(t, app, rawToken))
	})

	t.Run("C5 production: legacy session without stored fingerprint is allowed", func(t *testing.T) {
		rawToken := "c5-legacy-token"
		seedSchoolContext(t, rawToken, "A")
		// Legacy sessions were created before fingerprint collection; blank the
		// stored fingerprint and confirm they are not force-logged-out.
		_, err := pools.PG.Exec(context.Background(), `
			UPDATE sessions SET device_fingerprint = '' WHERE token_hash = $1
		`, hashToken(rawToken))
		require.NoError(t, err)

		app := deviceApp(config.Config{AppEnv: "production"}, "fp-whatever")
		assert.Equal(t, http.StatusOK, requestStatus(t, app, rawToken))
	})

	t.Run("C5 production: mismatched fingerprint is logged but allowed when enforcement disabled", func(t *testing.T) {
		rawToken := "c5-prod-mismatch-allowed"
		seedSchoolContext(t, rawToken, "A") // session stores fingerprint fp-ctx

		app := deviceApp(config.Config{AppEnv: "production"}, "fp-evil")
		assert.Equal(t, http.StatusOK, requestStatus(t, app, rawToken),
			"with enforcement disabled, a fingerprint mismatch should be logged but allowed")
	})

	t.Run("C5 development: mismatched fingerprint is allowed", func(t *testing.T) {
		rawToken := "c5-dev-token"
		seedSchoolContext(t, rawToken, "A")

		// Development must allow one device to act as many users — no
		// fingerprint enforcement at all.
		app := deviceApp(config.Config{AppEnv: "development"}, "fp-evil")
		assert.Equal(t, http.StatusOK, requestStatus(t, app, rawToken))
	})
}

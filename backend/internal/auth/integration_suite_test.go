package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"somotracker/backend/internal/config"
)

// ============================================================================
// Package-level test suite
// ============================================================================

var (
	testSuite     *IntegrationSuite
	suiteSetupOnce sync.Once
	suiteErr      error
)

// IntegrationSuite holds all the shared infrastructure for integration tests.
type IntegrationSuite struct {
	ctx            context.Context
	cancel         context.CancelFunc
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	pgHostPort     string // "host:port" for pg connection
	redisAddr      string // "host:port" for redis connection
	pgPool         *pgxpool.Pool
	rdb            *redis.Client
	svc            *Service
	hnd            *Handler
	cfg            config.Config
	logger         *zap.Logger
	observedLogs   *observer.ObservedLogs
	stytchURL      string // base URL of the mock Stytch server
	stytchServer   *httptest.Server

	// Stytch mock controls — test functions set these to control behavior
	stytchMu                  sync.Mutex
	stytchDiscoverySendFn     func(email string) (int, any)
	stytchDiscoveryAuthFn     func(token string) (int, any)
	stytchCreateOrgFn         func(name string) (int, any)
	stytchExchangeISTFn       func(ist, orgID string) (int, any)
	stytchCreateOrgCallCount  int
	stytchExchangeISTCallCount int
}

// ============================================================================
// TestMain — starts containers once for the entire package
// ============================================================================

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	suite, err := setupSuite(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration suite setup failed: %v\n", err)
		os.Exit(1)
	}

	testSuite = suite

	code := m.Run()

	suite.cleanup()

	os.Exit(code)
}

// ============================================================================
// Suite setup
// ============================================================================

func setupSuite(ctx context.Context) (*IntegrationSuite, error) {
	// ---------- 1. Start PostgreSQL container ----------
	fmt.Println("=== Starting PostgreSQL container...")
	pgC, pgHostPort, err := startPostgres(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres container: %w", err)
	}
	fmt.Printf("=== PostgreSQL ready at %s\n", pgHostPort)

	// ---------- 2. Start Redis container ----------
	fmt.Println("=== Starting Redis container...")
	redisC, redisAddr, err := startRedis(ctx)
	if err != nil {
		pgC.Terminate(ctx)
		return nil, fmt.Errorf("redis container: %w", err)
	}
	fmt.Printf("=== Redis ready at %s\n", redisAddr)

	// ---------- 3. Build config ----------
	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", pgHostPort)
	cfg := config.Config{
		DatabaseURL:         dbURL,
		RedisURL:            redisAddr,
		AppEnv:              "test",
		Port:                "0",
		AllowedOrigins:      "http://localhost:3000",
		CookieDomain:        "localhost",
		StytchProjectID:     "test-project-id",
		StytchSecret:        "test-secret",
		StytchEnv:           "test",
		StytchRedirectURL:   "http://localhost:3030/api/auth/callback",
		StytchBaseURL:       "", // will be set after mock server starts
	}

	// ---------- 4. Connect to Postgres and run migrations ----------
	pgCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		pgC.Terminate(ctx)
		redisC.Terminate(ctx)
		return nil, fmt.Errorf("parse pg config: %w", err)
	}
	pgCfg.MaxConns = 5
	pgCfg.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		pgC.Terminate(ctx)
		redisC.Terminate(ctx)
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		pgC.Terminate(ctx)
		redisC.Terminate(ctx)
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	fmt.Println("=== Migrations applied")

	// ---------- 5. Connect to Redis ----------
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		rdb.Close()
		pgC.Terminate(ctx)
		redisC.Terminate(ctx)
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	// ---------- 6. Create logger ----------
	observedCore, observedLogs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)

	// ---------- 7. Build suite ----------
	suite := &IntegrationSuite{
		ctx:          ctx,
		pgContainer:  pgC,
		redisContainer: redisC,
		pgHostPort:   pgHostPort,
		redisAddr:    redisAddr,
		pgPool:       pool,
		rdb:          rdb,
		cfg:          cfg,
		logger:       logger,
		observedLogs: observedLogs,

		// Default Stytch mock behaviors
		stytchDiscoverySendFn: func(email string) (int, any) {
			return http.StatusOK, map[string]any{
				"request_id":  "req-discovery-send-ok",
				"status_code": 200,
			}
		},
		stytchDiscoveryAuthFn: func(token string) (int, any) {
			return http.StatusOK, map[string]any{
				"request_id":                "req-discovery-auth-ok",
				"status_code":               200,
				"intermediate_session_token": "ist_test_" + token,
				"email_address":             "test@example.com",
				"discovered_organizations":  []any{},
			}
		},
		stytchCreateOrgFn: func(name string) (int, any) {
			orgID := fmt.Sprintf("org_test_%08x", rand.Uint32())
			return http.StatusOK, map[string]any{
				"request_id":  "req-create-org-ok",
				"status_code": 200,
				"organization": map[string]any{
					"organization_id":   orgID,
					"organization_name": name,
					"organization_slug": strings.ToLower(strings.ReplaceAll(name, " ", "-")),
				},
			}
		},
		stytchExchangeISTFn: func(ist, orgID string) (int, any) {
			return http.StatusOK, map[string]any{
				"request_id":             "req-exchange-ist-ok",
				"status_code":            200,
				"member_id":              "member_test_" + orgID,
				"session_token":          "sess_test_" + ist,
				"session_jwt":            "",
				"member_authenticated":   true,
				"intermediate_session_token": ist,
				"member": map[string]any{
					"member_id":   "member_test_" + orgID,
					"email_address": "test@example.com",
					"status":      "active",
				},
				"organization": map[string]any{
					"organization_id": orgID,
				},
			}
		},
	}

	// ---------- 8. Start mock Stytch server ----------
	suite.startMockStytchServer()

	// Update config with the mock server URL
	suite.cfg.StytchBaseURL = suite.stytchURL

	// ---------- 9. Build Service and Handler ----------
	repo := &SqlcRepository{
		pool:   pool,
		logger: logger,
		cfg:    suite.cfg,
	}
	idp, err := NewStytchAdapter(suite.cfg, logger)
	if err != nil {
		suite.cleanup()
		return nil, fmt.Errorf("create stytch adapter: %w", err)
	}

	// Manually create service without fx lifecycle
	svc := &Service{
		idp:    idp,
		repo:   repo,
		rdb:    rdb,
		logger: logger,
		cfg:    suite.cfg,
	}
	suite.svc = svc

	hnd := NewHandler(svc, logger, suite.cfg)
	suite.hnd = hnd

	return suite, nil
}

// ============================================================================
// Container helpers
// ============================================================================

func startPostgres(ctx context.Context) (testcontainers.Container, string, error) {
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
	if err != nil {
		return nil, "", err
	}

	host, err := c.Host(ctx)
	if err != nil {
		c.Terminate(ctx)
		return nil, "", err
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		c.Terminate(ctx)
		return nil, "", err
	}

	return c, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func startRedis(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("* Ready to accept connections"),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := c.Host(ctx)
	if err != nil {
		c.Terminate(ctx)
		return nil, "", err
	}

	port, err := c.MappedPort(ctx, "6379")
	if err != nil {
		c.Terminate(ctx)
		return nil, "", err
	}

	return c, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

// ============================================================================
// Migration runner
// ============================================================================

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationFiles := []string{
		"000001_initial_schema.up.sql",
	}

	// Find the migrations directory relative to the test file
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "database", "migrations")

	for _, f := range migrationFiles {
		path := filepath.Join(migrationsDir, f)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("execute migration %s: %w", f, err)
		}
	}

	return nil
}

// ============================================================================
// Mock Stytch HTTP server
// ============================================================================

func (s *IntegrationSuite) startMockStytchServer() {
	mux := http.NewServeMux()

	// POST /v1/b2b/magic_links/email/discovery/send
	mux.HandleFunc("/v1/b2b/magic_links/email/discovery/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeStytchError(w, http.StatusMethodNotAllowed, "method_not_allowed", "expected POST")
			return
		}
		s.stytchMu.Lock()
		fn := s.stytchDiscoverySendFn
		s.stytchMu.Unlock()

		status, body := fn("")
		writeStytchJSON(w, status, body)
	})

	// POST /v1/b2b/magic_links/discovery/authenticate
	mux.HandleFunc("/v1/b2b/magic_links/discovery/authenticate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeStytchError(w, http.StatusMethodNotAllowed, "method_not_allowed", "expected POST")
			return
		}

		var req struct {
			Token string `json:"discovery_magic_links_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeStytchError(w, http.StatusBadRequest, "invalid_request", "bad JSON")
			return
		}

		s.stytchMu.Lock()
		fn := s.stytchDiscoveryAuthFn
		s.stytchMu.Unlock()

		status, body := fn(req.Token)
		writeStytchJSON(w, status, body)
	})

	// POST /v1/b2b/organizations
	mux.HandleFunc("/v1/b2b/organizations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeStytchError(w, http.StatusMethodNotAllowed, "method_not_allowed", "expected POST")
			return
		}

		var req struct {
			Name string `json:"organization_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeStytchError(w, http.StatusBadRequest, "invalid_request", "bad JSON")
			return
		}

		s.stytchMu.Lock()
		fn := s.stytchCreateOrgFn
		s.stytchCreateOrgCallCount++
		s.stytchMu.Unlock()

		status, body := fn(req.Name)
		writeStytchJSON(w, status, body)
	})

	// POST /v1/b2b/discovery/intermediate_sessions/exchange
	mux.HandleFunc("/v1/b2b/discovery/intermediate_sessions/exchange", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeStytchError(w, http.StatusMethodNotAllowed, "method_not_allowed", "expected POST")
			return
		}

		var req struct {
			IST   string `json:"intermediate_session_token"`
			OrgID string `json:"organization_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeStytchError(w, http.StatusBadRequest, "invalid_request", "bad JSON")
			return
		}

		s.stytchMu.Lock()
		fn := s.stytchExchangeISTFn
		s.stytchExchangeISTCallCount++
		s.stytchMu.Unlock()

		status, body := fn(req.IST, req.OrgID)
		writeStytchJSON(w, status, body)
	})

	server := httptest.NewServer(mux)
	s.stytchServer = server
	s.stytchURL = server.URL
}

// setStytchHandlers atomically replaces the mock Stytch handler functions.
func (s *IntegrationSuite) setStytchHandlers(h StytchMockHandlers) {
	s.stytchMu.Lock()
	defer s.stytchMu.Unlock()
	if h.DiscoverySendFn != nil {
		s.stytchDiscoverySendFn = h.DiscoverySendFn
	}
	if h.DiscoveryAuthFn != nil {
		s.stytchDiscoveryAuthFn = h.DiscoveryAuthFn
	}
	if h.CreateOrgFn != nil {
		s.stytchCreateOrgFn = h.CreateOrgFn
	}
	if h.ExchangeISTFn != nil {
		s.stytchExchangeISTFn = h.ExchangeISTFn
	}
}

// resetStytchHandlers resets all mock handlers to their default success behaviors.
func (s *IntegrationSuite) resetStytchHandlers() {
	s.stytchMu.Lock()
	defer s.stytchMu.Unlock()
	s.stytchDiscoverySendFn = func(email string) (int, any) {
		return http.StatusOK, map[string]any{
			"request_id":  "req-discovery-send-ok",
			"status_code": 200,
		}
	}
	s.stytchDiscoveryAuthFn = func(token string) (int, any) {
		return http.StatusOK, map[string]any{
			"request_id":                  "req-discovery-auth-ok",
			"status_code":                 200,
			"intermediate_session_token":  "ist_test_" + token,
			"email_address":               "test@example.com",
			"discovered_organizations":    []any{},
		}
	}
	s.stytchCreateOrgFn = func(name string) (int, any) {
		orgID := fmt.Sprintf("org_test_%08x", rand.Uint32())
		return http.StatusOK, map[string]any{
			"request_id":  "req-create-org-ok",
			"status_code": 200,
			"organization": map[string]any{
				"organization_id":   orgID,
				"organization_name": name,
				"organization_slug": strings.ToLower(strings.ReplaceAll(name, " ", "-")),
			},
		}
	}
	s.stytchExchangeISTFn = func(ist, orgID string) (int, any) {
		return http.StatusOK, map[string]any{
			"request_id":                "req-exchange-ist-ok",
			"status_code":               200,
			"member_id":                 "member_test_" + orgID,
			"session_token":             "sess_test_" + ist,
			"member_authenticated":      true,
			"intermediate_session_token": ist,
			"member": map[string]any{
				"member_id":     "member_test_" + orgID,
				"email_address": "test@example.com",
				"status":        "active",
			},
			"organization": map[string]any{
				"organization_id": orgID,
			},
		}
	}
	s.stytchCreateOrgCallCount = 0
	s.stytchExchangeISTCallCount = 0
}

// StytchMockHandlers holds optional handler overrides for the mock Stytch server.
type StytchMockHandlers struct {
	DiscoverySendFn func(email string) (statusCode int, response any)
	DiscoveryAuthFn func(token string) (statusCode int, response any)
	CreateOrgFn     func(name string) (statusCode int, response any)
	ExchangeISTFn   func(ist, orgID string) (statusCode int, response any)
}

// ============================================================================
// Stytch JSON response helpers
// ============================================================================

func writeStytchJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeStytchError(w http.ResponseWriter, status int, errType, message string) {
	writeStytchJSON(w, status, map[string]any{
		"status_code":   status,
		"error_type":    errType,
		"error_message": message,
		"request_id":    "req-error-" + fmt.Sprintf("%08x", rand.Uint32()),
	})
}

// ============================================================================
// Suite cleanup
// ============================================================================

func (s *IntegrationSuite) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if s.pgPool != nil {
		s.pgPool.Close()
	}
	if s.rdb != nil {
		s.rdb.Close()
	}
	if s.stytchServer != nil {
		s.stytchServer.Close()
	}
	if s.redisContainer != nil {
		s.redisContainer.Terminate(ctx)
	}
	if s.pgContainer != nil {
		s.pgContainer.Terminate(ctx)
	}
}

// ============================================================================
// Test helpers
// ============================================================================

// freshDB cleans the sessions, users, and tenants tables between test cases.
func (s *IntegrationSuite) freshDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.pgPool.Exec(ctx, "DELETE FROM sessions"); err != nil {
		t.Fatalf("clean sessions: %v", err)
	}
	if _, err := s.pgPool.Exec(ctx, "DELETE FROM users"); err != nil {
		t.Fatalf("clean users: %v", err)
	}
	if _, err := s.pgPool.Exec(ctx, "DELETE FROM tenants"); err != nil {
		t.Fatalf("clean tenants: %v", err)
	}
}

// freshRedis flushes all keys from Redis between test cases.
func (s *IntegrationSuite) freshRedis(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := s.rdb.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

// insertSession inserts a session directly into Postgres (bypassing the service layer).
func (s *IntegrationSuite) insertSession(t *testing.T, session UserSession) {
	t.Helper()
	ctx := context.Background()
	_, err := s.pgPool.Exec(ctx, `
		INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id, stytch_org_id, stytch_session_token, device_fingerprint, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, session.Token, session.UserID, session.TenantID, session.StytchMemberID,
		session.StytchOrgID, session.StytchSessionToken, session.DeviceFingerprint,
		session.ExpiresAt, session.CreatedAt)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

// insertTenant inserts a tenant directly into Postgres.
func (s *IntegrationSuite) insertTenant(t *testing.T, id, name, slug, stytchOrgID string) {
	t.Helper()
	ctx := context.Background()
	_, err := s.pgPool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, stytch_org_id)
		VALUES ($1, $2, $3, $4)
	`, id, name, slug, stytchOrgID)
	if err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
}

// insertUser inserts a user directly into Postgres.
func (s *IntegrationSuite) insertUser(t *testing.T, id, email, tenantID, externalAuthID string) {
	t.Helper()
	ctx := context.Background()
	_, err := s.pgPool.Exec(ctx, `
		INSERT INTO users (id, email, tenant_id, first_name, last_name, external_auth_id)
		VALUES ($1, $2, $3, '', '', $4)
	`, id, email, tenantID, externalAuthID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

// setRedisSession sets a session token mapping in Redis (as the service would).
func (s *IntegrationSuite) setRedisSession(t *testing.T, token, stytchSessionToken string) {
	t.Helper()
	ctx := context.Background()
	err := s.rdb.Set(ctx, "session:"+token, stytchSessionToken, 30*24*time.Hour).Err()
	if err != nil {
		t.Fatalf("set redis session: %v", err)
	}
}

// verifyRedisSession checks if a session exists in Redis.
func (s *IntegrationSuite) verifyRedisSession(t *testing.T, token string) bool {
	t.Helper()
	ctx := context.Background()
	exists, err := s.rdb.Exists(ctx, "session:"+token).Result()
	if err != nil {
		t.Fatalf("check redis session: %v", err)
	}
	return exists == 1
}

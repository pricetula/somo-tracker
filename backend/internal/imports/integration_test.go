package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"somotracker/backend/internal/database"
)

// ============================================================================
// Integration suite — shared Postgres + Redis containers
// ============================================================================

type IntegrationSuite struct {
	ctx      context.Context
	pgC      testcontainers.Container
	redisC   testcontainers.Container
	pgPool   *pgxpool.Pool
	rdb      *redis.Client
	repo     *PgRepository
	svc      *Service
	asynqSrv *asynq.Server
	asynqCl  *asynq.Client
	asynqMux *asynq.ServeMux
	cancel   context.CancelFunc
}

var (
	integrationSuite *IntegrationSuite
	suiteOnce        sync.Once
)

func getIntegrationSuite(t *testing.T) *IntegrationSuite {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	var initErr error
	suiteOnce.Do(func() {
		suite, err := setupSuite()
		if err != nil {
			initErr = fmt.Errorf("setup integration suite: %w", err)
			return
		}
		integrationSuite = suite
	})

	if initErr != nil {
		t.Fatalf("integration suite setup failed: %v", initErr)
	}

	return integrationSuite
}

// ============================================================================
// Suite setup / teardown
// ============================================================================

func setupSuite() (*IntegrationSuite, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	// ── PostgreSQL ──────────────────────────────────────────────────────
	pgC, pgHostPort, err := startPostgres(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("postgres: %w", err)
	}

	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", pgHostPort)

	pgCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		_ = pgC.Terminate(ctx)
		cancel()
		return nil, fmt.Errorf("parse pg config: %w", err)
	}
	pgCfg.MaxConns = 10
	pgCfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		_ = pgC.Terminate(ctx)
		cancel()
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		_ = pgC.Terminate(ctx)
		cancel()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	// ── Redis ───────────────────────────────────────────────────────────
	redisC, redisAddr, err := startRedis(ctx)
	if err != nil {
		pool.Close()
		_ = pgC.Terminate(ctx)
		cancel()
		return nil, fmt.Errorf("redis: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		pool.Close()
		_ = redisC.Terminate(ctx)
		_ = pgC.Terminate(ctx)
		cancel()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	// ── Build service ───────────────────────────────────────────────────
	pools := &database.Pools{PG: pool, Redis: rdb}
	repo := NewRepository(pools)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	svc := NewService(repo, pools, asynqClient)

	// ── Asynq server for processing ────────────────────────────────────
	asynqSrv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 3,
			Queues:      map[string]int{"imports": 10},
		},
	)

	// Register the process_chunk handler on a mux
	mux := asynq.NewServeMux()
	mux.HandleFunc("imports:process_chunk", func(cctx context.Context, t *asynq.Task) error {
		var payload ChunkTaskPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}
		return svc.ProcessChunk(cctx, payload)
	})

	// Start the Asynq server in background
	go func() {
		if err := asynqSrv.Start(mux); err != nil {
			fmt.Fprintf(os.Stderr, "asynq server error: %v\n", err)
		}
	}()

	return &IntegrationSuite{
		ctx:      ctx,
		pgC:      pgC,
		redisC:   redisC,
		pgPool:   pool,
		rdb:      rdb,
		repo:     repo,
		svc:      svc,
		asynqSrv: asynqSrv,
		asynqCl:  asynqClient,
		asynqMux: mux,
		cancel:   cancel,
	}, nil
}

func (s *IntegrationSuite) TearDown() {
	s.asynqSrv.Shutdown()
	_ = s.asynqCl.Close()
	_ = s.rdb.Close()
	s.pgPool.Close()
	_ = s.pgC.Terminate(s.ctx)
	_ = s.redisC.Terminate(s.ctx)
	s.cancel()
}

// ============================================================================
// Per-test cleanup
// ============================================================================

func (s *IntegrationSuite) freshDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Order matters (FK constraints)
	tables := []string{
		"import_job_failures",
		"import_job_chunks",
		"import_job_staging",
		"import_jobs",
		"cbc_student_enrollments",
		"cbc_students",
		"cbc_classes",
		"cbc_streams",
		"academic_terms",
		"academic_years",
		"cbc_schools",
		"users",
		"tenants",
	}
	for _, tName := range tables {
		if _, err := s.pgPool.Exec(ctx, "DELETE FROM "+tName); err != nil {
			t.Fatalf("clean %s: %v", tName, err)
		}
	}
}

func (s *IntegrationSuite) freshRedis(t *testing.T) {
	t.Helper()
	if err := s.rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

// ============================================================================
// DB helpers — insert seed data
// ============================================================================

func (s *IntegrationSuite) insertTenant(t *testing.T, id, name string) {
	t.Helper()
	_, err := s.pgPool.Exec(context.Background(),
		"INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)",
		id, name, name+"-slug", "org_"+id[:8])
	if err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
}

func (s *IntegrationSuite) insertUser(t *testing.T, id, tenantID, email string) {
	t.Helper()
	_, err := s.pgPool.Exec(context.Background(),
		"INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)",
		id, email, tenantID, "Test User")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func (s *IntegrationSuite) insertSchool(t *testing.T, id, tenantID, name string) {
	t.Helper()
	_, err := s.pgPool.Exec(context.Background(),
		"INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, $4, $5, $6)",
		id, tenantID, name, "County", "SubCounty", "Public")
	if err != nil {
		t.Fatalf("insert school: %v", err)
	}
}

func (s *IntegrationSuite) insertAcademicYear(t *testing.T, id, tenantID, schoolID, name string, isCurrent bool, createdBy string) {
	t.Helper()
	_, err := s.pgPool.Exec(context.Background(),
		"INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)",
		id, tenantID, schoolID, name, "2026-01-01", "2026-12-31", isCurrent, createdBy)
	if err != nil {
		t.Fatalf("insert academic year: %v", err)
	}
}

func (s *IntegrationSuite) insertAcademicTerm(t *testing.T, id, tenantID, schoolID, yearID, name string, termNum int, isCurrent bool, createdBy string) {
	t.Helper()
	_, err := s.pgPool.Exec(context.Background(),
		"INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)",
		id, tenantID, schoolID, yearID, name, termNum, "2026-01-01", "2026-04-30", isCurrent, createdBy)
	if err != nil {
		t.Fatalf("insert academic term: %v", err)
	}
}

// ============================================================================
// Assert helpers
// ============================================================================

func (s *IntegrationSuite) queryInt(t *testing.T, query string, args ...any) int {
	t.Helper()
	var v int
	err := s.pgPool.QueryRow(context.Background(), query, args...).Scan(&v)
	if err != nil {
		t.Fatalf("query int (%s): %v", query, err)
	}
	return v
}

func (s *IntegrationSuite) queryString(t *testing.T, query string, args ...any) string {
	t.Helper()
	var v string
	err := s.pgPool.QueryRow(context.Background(), query, args...).Scan(&v)
	if err != nil {
		t.Fatalf("query string (%s): %v", query, err)
	}
	return v
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
		_ = c.Terminate(ctx)
		return nil, "", err
	}
	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		_ = c.Terminate(ctx)
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
		_ = c.Terminate(ctx)
		return nil, "", err
	}
	port, err := c.MappedPort(ctx, "6379")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}
	return c, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "database", "migrations")
	sql, err := os.ReadFile(filepath.Join(migrationsDir, "000001_initial_schema.up.sql"))
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	return nil
}

// ============================================================================
// Importer registration helper
// ============================================================================

// registerImporterForTest registers a mock importer for STUDENT_IMPORT.
// Returns a cleanup function.
func registerImporterForTest(t *testing.T, imp Importer) func() {
	t.Helper()
	existing, had := ImporterRegistry[ImportJobTypeStudentImport]
	ImporterRegistry[ImportJobTypeStudentImport] = imp
	return func() {
		if had {
			ImporterRegistry[ImportJobTypeStudentImport] = existing
		} else {
			delete(ImporterRegistry, ImportJobTypeStudentImport)
		}
	}
}

// ============================================================================
// Minimal mock importer for integration tests
// ============================================================================

// integrationImporter is a simplified StudentImporter that works against
// the real database for integration tests.
type integrationImporter struct {
	jobType       ImportJobType
	validateFn    func(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]ValidatedRow, []RowFailure)
	resolveRefsFn func(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []ValidatedRow) ([]ValidatedRow, []RowFailure)
	bulkInsertFn  func(ctx context.Context, tx pgx.Tx, rows []ValidatedRow) (int, error)
	insertOneFn   func(ctx context.Context, tx pgx.Tx, row ValidatedRow) error
}

func (m *integrationImporter) JobType() ImportJobType { return m.jobType }
func (m *integrationImporter) Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]ValidatedRow, []RowFailure) {
	if m.validateFn != nil {
		return m.validateFn(ctx, tenantID, schoolID, raw)
	}
	result := make([]ValidatedRow, len(raw))
	for i, r := range raw {
		result[i] = ValidatedRow{RawData: r}
	}
	return result, nil
}
func (m *integrationImporter) ResolveReferences(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []ValidatedRow) ([]ValidatedRow, []RowFailure) {
	if m.resolveRefsFn != nil {
		return m.resolveRefsFn(ctx, tenantID, schoolID, metadata, rows)
	}
	return rows, nil
}
func (m *integrationImporter) BulkInsert(ctx context.Context, tx pgx.Tx, rows []ValidatedRow) (int, error) {
	if m.bulkInsertFn != nil {
		return m.bulkInsertFn(ctx, tx, rows)
	}
	return len(rows), nil
}
func (m *integrationImporter) InsertOne(ctx context.Context, tx pgx.Tx, row ValidatedRow) error {
	if m.insertOneFn != nil {
		return m.insertOneFn(ctx, tx, row)
	}
	return nil
}

// ============================================================================
// Real DB-based student importer for integration
// ============================================================================

// realStudentImporter inserts a student row into cbc_students with staging_row_id.
type realStudentImporter struct {
	*integrationImporter
}

func newRealStudentImporter() *realStudentImporter {
	base := &integrationImporter{jobType: ImportJobTypeStudentImport}
	// ResolveReferences injects tenant/school/staging_row_id context into each row
	base.resolveRefsFn = func(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []ValidatedRow) ([]ValidatedRow, []RowFailure) {
		var meta struct {
			AcademicTermID string `json:"academic_term_id"`
			AcademicYearID string `json:"academic_year_id"`
		}
		if err := json.Unmarshal(metadata, &meta); err != nil {
			return nil, allIntegrationFail(rows, "bad metadata")
		}
		resolved := make([]ValidatedRow, len(rows))
		for i, row := range rows {
			var raw struct {
				FullName string `json:"full_name"`
				Gender   string `json:"gender"`
			}
			if err := json.Unmarshal(row.RawData, &raw); err != nil {
				return nil, allIntegrationFail(rows, "bad row")
			}
			aug, _ := json.Marshal(map[string]any{
				"full_name":        raw.FullName,
				"gender":           raw.Gender,
				"tenant_id":        tenantID.String(),
				"school_id":        schoolID.String(),
				"academic_term_id": meta.AcademicTermID,
				"academic_year_id": meta.AcademicYearID,
				"staging_row_id":   row.StagingRowID.String(),
			})
			resolved[i] = ValidatedRow{RawData: aug, StagingRowID: row.StagingRowID}
		}
		return resolved, nil
	}
	base.bulkInsertFn = func(ctx context.Context, tx pgx.Tx, rows []ValidatedRow) (int, error) {
		return 0, fmt.Errorf("force savepoint fallback")
	}
	base.insertOneFn = func(ctx context.Context, tx pgx.Tx, row ValidatedRow) error {
		var aug struct {
			FullName     string `json:"full_name"`
			Gender       string `json:"gender"`
			TenantID     string `json:"tenant_id"`
			SchoolID     string `json:"school_id"`
			StagingRowID string `json:"staging_row_id"`
		}
		if err := json.Unmarshal(row.RawData, &aug); err != nil {
			return err
		}

		// Insert the student with staging_row_id, using ON CONFLICT for idempotency
		_, err := tx.Exec(ctx, `
			INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, staging_row_id, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL
			DO UPDATE SET full_name = EXCLUDED.full_name
		`, aug.TenantID, aug.SchoolID, aug.FullName, aug.Gender, aug.StagingRowID)
		return err
	}
	return &realStudentImporter{integrationImporter: base}
}

func allIntegrationFail(rows []ValidatedRow, msg string) []RowFailure {
	fails := make([]RowFailure, len(rows))
	for i, r := range rows {
		fails[i] = RowFailure{RowNumber: i, RawPayload: r.RawData, ErrorMessage: msg, ErrorType: ImportFailureBusinessRule}
	}
	return fails
}

// ============================================================================
// Tests
// ============================================================================

// TestIntegration_HappyPath_Import50Students creates a job with 50 students,
// processes the chunk, and verifies all 50 end up in cbc_students.
func TestIntegration_HappyPath_Import50Students(t *testing.T) {
	suite := getIntegrationSuite(t)
	suite.freshDB(t)
	suite.freshRedis(t)
	defer func() { suite.freshRedis(t); suite.freshDB(t) }()

	ctx := context.Background()

	// ── Seed data ───────────────────────────────────────────────────────
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()
	suite.insertTenant(t, tenantID, "Test Tenant")
	suite.insertSchool(t, schoolID, tenantID, "Test School")
	suite.insertUser(t, userID, tenantID, "admin@test.com")

	yearID := uuid.New().String()
	termID := uuid.New().String()
	suite.insertAcademicYear(t, yearID, tenantID, schoolID, "2026", true, userID)
	suite.insertAcademicTerm(t, termID, tenantID, schoolID, yearID, "Term 1", 1, true, userID)

	// ── Register importer ───────────────────────────────────────────────
	cleanup := registerImporterForTest(t, newRealStudentImporter())
	defer cleanup()

	// ── Create import job ───────────────────────────────────────────────
	rows := make([]json.RawMessage, 50)
	for i := 0; i < 50; i++ {
		rows[i] = json.RawMessage(`{"full_name":"Student ` + itoa(i) + `","gender":"M"}`)
	}

	metadata, _ := json.Marshal(map[string]string{
		"academic_term_id": termID,
		"academic_year_id": yearID,
	})

	resp, err := suite.svc.CreateJob(ctx, CreateJobRequest{
		TenantID:  uuid.MustParse(tenantID),
		SchoolID:  uuid.MustParse(schoolID),
		JobType:   ImportJobTypeStudentImport,
		CreatedBy: uuid.MustParse(userID),
		Rows:      rows,
		Metadata:  metadata,
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if resp.TotalRecords != 50 {
		t.Fatalf("expected TotalRecords=50, got %d", resp.TotalRecords)
	}
	if resp.TotalChunks != 1 {
		t.Fatalf("expected TotalChunks=1, got %d", resp.TotalChunks)
	}
	if resp.Status != ImportJobStatusProcessing {
		t.Fatalf("expected status=processing, got %s", resp.Status)
	}

	// ── Process the chunk directly (bypass Asynq for reliability) ──────
	err = suite.svc.ProcessChunk(ctx, ChunkTaskPayload{
		JobID:          resp.JobID.String(),
		ChunkIndex:     0,
		RowNumberStart: 0,
		RowNumberEnd:   50,
	})
	if err != nil {
		t.Fatalf("ProcessChunk failed: %v", err)
	}

	// ── Verify job state ────────────────────────────────────────────────
	job, err := suite.svc.GetJob(ctx, resp.JobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if job.Status != ImportJobStatusCompleted {
		t.Fatalf("expected job status=completed, got %s", job.Status)
	}
	if job.ProcessedRecords != 50 {
		t.Fatalf("expected processed_records=50, got %d", job.ProcessedRecords)
	}
	if job.SuccessCount != 50 {
		t.Fatalf("expected success_count=50, got %d", job.SuccessCount)
	}
	if job.FailedCount != 0 {
		t.Fatalf("expected failed_count=0, got %d", job.FailedCount)
	}
	if job.ProcessedChunks != 1 {
		t.Fatalf("expected processed_chunks=1, got %d", job.ProcessedChunks)
	}

	// ── Verify students in DB ───────────────────────────────────────────
	studentCount := suite.queryInt(t, "SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)
	if studentCount != 50 {
		t.Fatalf("expected 50 students in cbc_students, got %d", studentCount)
	}

	// ── Verify chunk status ─────────────────────────────────────────────
	chunkStatus := suite.queryString(t,
		"SELECT status FROM import_job_chunks WHERE job_id = $1 AND chunk_index = 0", resp.JobID)
	if chunkStatus != "completed" {
		t.Fatalf("expected chunk status=completed, got %s", chunkStatus)
	}

	// ── Verify all staging rows marked succeeded ────────────────────────
	pendingCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM import_job_staging WHERE job_id = $1 AND status = 'pending'", resp.JobID)
	if pendingCount != 0 {
		t.Fatalf("expected 0 pending staging rows, got %d", pendingCount)
	}

	succeededCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM import_job_staging WHERE job_id = $1 AND status = 'succeeded'", resp.JobID)
	if succeededCount != 50 {
		t.Fatalf("expected 50 succeeded staging rows, got %d", succeededCount)
	}

	// ── Verify staging_row_id is set on students ────────────────────────
	studentsWithStagingRow := suite.queryInt(t,
		"SELECT COUNT(*) FROM cbc_students WHERE staging_row_id IS NOT NULL AND school_id = $1", schoolID)
	if studentsWithStagingRow != 50 {
		t.Fatalf("expected 50 students with staging_row_id, got %d", studentsWithStagingRow)
	}
}

// TestIntegration_RedeliveryAfterCrash simulates a worker crash after 2
// rows were inserted+marked, then redelivers the same chunk.
func TestIntegration_RedeliveryAfterCrash(t *testing.T) {
	suite := getIntegrationSuite(t)
	suite.freshDB(t)
	suite.freshRedis(t)
	defer func() { suite.freshRedis(t); suite.freshDB(t) }()

	ctx := context.Background()

	// ── Seed data ───────────────────────────────────────────────────────
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()
	suite.insertTenant(t, tenantID, "Crash Tenant")
	suite.insertSchool(t, schoolID, tenantID, "Crash School")
	suite.insertUser(t, userID, tenantID, "admin@crash.com")

	yearID := uuid.New().String()
	termID := uuid.New().String()
	suite.insertAcademicYear(t, yearID, tenantID, schoolID, "2026", true, userID)
	suite.insertAcademicTerm(t, termID, tenantID, schoolID, yearID, "Term 1", 1, true, userID)

	cleanup := registerImporterForTest(t, newRealStudentImporter())
	defer cleanup()

	// ── Create job with 5 students ──────────────────────────────────────
	rows := make([]json.RawMessage, 5)
	for i := 0; i < 5; i++ {
		rows[i] = json.RawMessage(`{"full_name":"Student ` + itoa(i) + `","gender":"M"}`)
	}

	metadata, _ := json.Marshal(map[string]string{
		"academic_term_id": termID,
		"academic_year_id": yearID,
	})

	resp, err := suite.svc.CreateJob(ctx, CreateJobRequest{
		TenantID:  uuid.MustParse(tenantID),
		SchoolID:  uuid.MustParse(schoolID),
		JobType:   ImportJobTypeStudentImport,
		CreatedBy: uuid.MustParse(userID),
		Rows:      rows,
		Metadata:  metadata,
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// ── Simulate first attempt: insert 2 rows directly, mark staging ───
	stagingRows, err := suite.repo.GetStagingRows(ctx, resp.JobID, 0, 5)
	if err != nil {
		t.Fatalf("GetStagingRows: %v", err)
	}
	if len(stagingRows) != 5 {
		t.Fatalf("expected 5 pending staging rows, got %d", len(stagingRows))
	}

	// Insert first 2 students + mark staging as succeeded (simulating crash after commit)
	for i := 0; i < 2; i++ {
		_, err := suite.pgPool.Exec(ctx, `
			INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, staging_row_id, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
		`, tenantID, schoolID, "Student "+itoa(i), "M", stagingRows[i].ID)
		if err != nil {
			t.Fatalf("insert student %d: %v", i, err)
		}
		_, err = suite.pgPool.Exec(ctx, `
			UPDATE import_job_staging SET status = 'succeeded', processed_at = NOW() WHERE id = $1
		`, stagingRows[i].ID)
		if err != nil {
			t.Fatalf("mark staging row %d: %v", i, err)
		}
	}

	// Assert: 2 students in DB, 3 pending, 2 succeeded
	if suite.queryInt(t, "SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID) != 2 {
		t.Fatal("expected 2 students after crash simulation")
	}
	if suite.queryInt(t, "SELECT COUNT(*) FROM import_job_staging WHERE job_id = $1 AND status = 'pending'", resp.JobID) != 3 {
		t.Fatal("expected 3 pending staging rows after crash simulation")
	}

	// ── Redeliver: process the same chunk ───────────────────────────────
	err = suite.svc.ProcessChunk(ctx, ChunkTaskPayload{
		JobID:          resp.JobID.String(),
		ChunkIndex:     0,
		RowNumberStart: 0,
		RowNumberEnd:   5,
	})
	if err != nil {
		t.Fatalf("ProcessChunk (redelivery) failed: %v", err)
	}

	// ── Assertions ──────────────────────────────────────────────────────
	// Exactly 5 students total (no duplicates from the 2 already inserted)
	studentCount := suite.queryInt(t, "SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)
	if studentCount != 5 {
		t.Fatalf("expected exactly 5 students (no duplicates), got %d", studentCount)
	}

	// All staging rows succeeded
	succeededCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM import_job_staging WHERE job_id = $1 AND status = 'succeeded'", resp.JobID)
	if succeededCount != 5 {
		t.Fatalf("expected 5 succeeded staging rows, got %d", succeededCount)
	}

	// Job counters: processed=5, success=3 (only the 3 from this attempt)
	job, err := suite.svc.GetJob(ctx, resp.JobID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if job.ProcessedRecords != 5 {
		t.Fatalf("expected processed_records=5, got %d", job.ProcessedRecords)
	}
	if job.SuccessCount != 3 {
		t.Fatalf("expected success_count=3 (3 new inserts), got %d", job.SuccessCount)
	}
	if job.FailedCount != 0 {
		t.Fatalf("expected failed_count=0, got %d", job.FailedCount)
	}
	if job.ProcessedChunks != 1 {
		t.Fatalf("expected processed_chunks=1, got %d", job.ProcessedChunks)
	}

	// Chunk is completed
	chunkStatus := suite.queryString(t,
		"SELECT status FROM import_job_chunks WHERE job_id = $1 AND chunk_index = 0", resp.JobID)
	if chunkStatus != "completed" {
		t.Fatalf("expected chunk status=completed, got %s", chunkStatus)
	}
}

// TestIntegration_ConcurrentClaim verifies two workers racing to claim
// the same chunk — exactly one proceeds.
func TestIntegration_ConcurrentClaim(t *testing.T) {
	suite := getIntegrationSuite(t)
	suite.freshDB(t)
	suite.freshRedis(t)
	defer func() { suite.freshRedis(t); suite.freshDB(t) }()

	ctx := context.Background()

	// ── Seed data ───────────────────────────────────────────────────────
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()
	suite.insertTenant(t, tenantID, "Race Tenant")
	suite.insertSchool(t, schoolID, tenantID, "Race School")
	suite.insertUser(t, userID, tenantID, "race@test.com")

	yearID := uuid.New().String()
	termID := uuid.New().String()
	suite.insertAcademicYear(t, yearID, tenantID, schoolID, "2026", true, userID)
	suite.insertAcademicTerm(t, termID, tenantID, schoolID, yearID, "Term 1", 1, true, userID)

	cleanup := registerImporterForTest(t, newRealStudentImporter())
	defer cleanup()

	// ── Create a job with 10 students ───────────────────────────────────
	rows := make([]json.RawMessage, 10)
	for i := 0; i < 10; i++ {
		rows[i] = json.RawMessage(`{"full_name":"S` + itoa(i) + `","gender":"F"}`)
	}

	metadata, _ := json.Marshal(map[string]string{
		"academic_term_id": termID,
		"academic_year_id": yearID,
	})

	resp, err := suite.svc.CreateJob(ctx, CreateJobRequest{
		TenantID:  uuid.MustParse(tenantID),
		SchoolID:  uuid.MustParse(schoolID),
		JobType:   ImportJobTypeStudentImport,
		CreatedBy: uuid.MustParse(userID),
		Rows:      rows,
		Metadata:  metadata,
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// ── Create two services sharing the same repo (simulating workers) ──
	svc1 := &Service{
		repo:  suite.repo,
		pool:  suite.pgPool,
		asynq: suite.asynqCl,
	}
	svc2 := &Service{
		repo:  suite.repo,
		pool:  suite.pgPool,
		asynq: suite.asynqCl,
	}
	// Both use real transactions
	svc1.beginTx = func(c context.Context) (pgx.Tx, error) { return suite.pgPool.Begin(c) }
	svc2.beginTx = func(c context.Context) (pgx.Tx, error) { return suite.pgPool.Begin(c) }

	payload := ChunkTaskPayload{
		JobID:          resp.JobID.String(),
		ChunkIndex:     0,
		RowNumberStart: 0,
		RowNumberEnd:   10,
	}

	// ── Race both workers ──────────────────────────────────────────────
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	var studentCountAfterRace atomic.Int64

	wg.Add(2)
	go func() {
		defer wg.Done()
		if e := svc1.ProcessChunk(ctx, payload); e != nil {
			errs <- e
		}
		studentCountAfterRace.Add(int64(suite.queryInt(t,
			"SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)))
	}()
	go func() {
		defer wg.Done()
		if e := svc2.ProcessChunk(ctx, payload); e != nil {
			errs <- e
		}
		studentCountAfterRace.Add(int64(suite.queryInt(t,
			"SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)))
	}()
	wg.Wait()
	close(errs)

	for e := range errs {
		if e != nil {
			t.Errorf("ProcessChunk returned error: %v", e)
		}
	}

	// Exactly 10 students (the claim race prevented duplicate processing)
	finalCount := suite.queryInt(t, "SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)
	if finalCount != 10 {
		t.Fatalf("expected exactly 10 students after concurrent claim race, got %d", finalCount)
	}

	// Chunk completed exactly once
	chunkStatus := suite.queryString(t,
		"SELECT status FROM import_job_chunks WHERE job_id = $1 AND chunk_index = 0", resp.JobID)
	if chunkStatus != "completed" {
		t.Fatalf("expected chunk status=completed, got %s", chunkStatus)
	}

	claimedAtCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM import_job_chunks WHERE job_id = $1 AND claimed_at IS NOT NULL", resp.JobID)
	if claimedAtCount != 1 {
		t.Fatalf("expected exactly 1 claimed_at, got %d", claimedAtCount)
	}
}

// TestIntegration_DoubleChunkCompletion verifies that calling
// AtomicChunkCompletion twice for the same chunk does not double-count.
func TestIntegration_DoubleChunkCompletion(t *testing.T) {
	suite := getIntegrationSuite(t)
	suite.freshDB(t)
	suite.freshRedis(t)
	defer func() { suite.freshRedis(t); suite.freshDB(t) }()

	ctx := context.Background()

	// ── Seed data ───────────────────────────────────────────────────────
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()
	suite.insertTenant(t, tenantID, "Double Tenant")
	suite.insertSchool(t, schoolID, tenantID, "Double School")
	suite.insertUser(t, userID, tenantID, "double@test.com")

	yearID := uuid.New().String()
	termID := uuid.New().String()
	suite.insertAcademicYear(t, yearID, tenantID, schoolID, "2026", true, userID)
	suite.insertAcademicTerm(t, termID, tenantID, schoolID, yearID, "Term 1", 1, true, userID)

	cleanup := registerImporterForTest(t, newRealStudentImporter())
	defer cleanup()

	// ── Create job ──────────────────────────────────────────────────────
	rows := make([]json.RawMessage, 10)
	for i := 0; i < 10; i++ {
		rows[i] = json.RawMessage(`{"full_name":"S` + itoa(i) + `","gender":"F"}`)
	}

	metadata, _ := json.Marshal(map[string]string{
		"academic_term_id": termID,
		"academic_year_id": yearID,
	})

	resp, _ := suite.svc.CreateJob(ctx, CreateJobRequest{
		TenantID:  uuid.MustParse(tenantID),
		SchoolID:  uuid.MustParse(schoolID),
		JobType:   ImportJobTypeStudentImport,
		CreatedBy: uuid.MustParse(userID),
		Rows:      rows,
		Metadata:  metadata,
	})

	// Process the chunk normally (first completion)
	err := suite.svc.ProcessChunk(ctx, ChunkTaskPayload{
		JobID:          resp.JobID.String(),
		ChunkIndex:     0,
		RowNumberStart: 0,
		RowNumberEnd:   10,
	})
	if err != nil {
		t.Fatalf("first ProcessChunk: %v", err)
	}

	// Record job state after first completion
	jobAfterFirst, _ := suite.svc.GetJob(ctx, resp.JobID)
	processedAfterFirst := jobAfterFirst.ProcessedRecords
	successAfterFirst := jobAfterFirst.SuccessCount
	failedAfterFirst := jobAfterFirst.FailedCount
	chunksAfterFirst := jobAfterFirst.ProcessedChunks

	// Get the chunk ID
	var chunkID string
	err = suite.pgPool.QueryRow(ctx,
		"SELECT id FROM import_job_chunks WHERE job_id = $1 AND chunk_index = 0", resp.JobID).Scan(&chunkID)
	if err != nil {
		t.Fatalf("get chunk id: %v", err)
	}

	// Call AtomicChunkCompletion a second time with the same params
	_, _, err = suite.repo.AtomicChunkCompletion(ctx, resp.JobID, uuid.MustParse(chunkID), 10, 10, 0)
	if err != nil {
		t.Fatalf("second AtomicChunkCompletion: %v", err)
	}

	// Verify counters are unchanged
	jobAfterSecond, _ := suite.svc.GetJob(ctx, resp.JobID)
	if jobAfterSecond.ProcessedRecords != processedAfterFirst {
		t.Fatalf("processed_records changed from %d to %d after second completion",
			processedAfterFirst, jobAfterSecond.ProcessedRecords)
	}
	if jobAfterSecond.SuccessCount != successAfterFirst {
		t.Fatalf("success_count changed from %d to %d after second completion",
			successAfterFirst, jobAfterSecond.SuccessCount)
	}
	if jobAfterSecond.FailedCount != failedAfterFirst {
		t.Fatalf("failed_count changed from %d to %d after second completion",
			failedAfterFirst, jobAfterSecond.FailedCount)
	}
	if jobAfterSecond.ProcessedChunks != chunksAfterFirst {
		t.Fatalf("processed_chunks changed from %d to %d after second completion",
			chunksAfterFirst, jobAfterSecond.ProcessedChunks)
	}

	// Chunk still completed
	chunkStatus := suite.queryString(t,
		"SELECT status FROM import_job_chunks WHERE id = $1", chunkID)
	if chunkStatus != "completed" {
		t.Fatalf("expected chunk status=completed, got %s", chunkStatus)
	}
}

// TestIntegration_UniqueConstraintOnStagingRow verifies that inserting
// a student with a staging_row_id that already exists (ON CONFLICT) is
// treated as success, not a duplicate error.
func TestIntegration_UniqueConstraintOnStagingRow(t *testing.T) {
	suite := getIntegrationSuite(t)
	suite.freshDB(t)
	suite.freshRedis(t)
	defer func() { suite.freshRedis(t); suite.freshDB(t) }()

	ctx := context.Background()

	// ── Seed data ───────────────────────────────────────────────────────
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()
	suite.insertTenant(t, tenantID, "UC Tenant")
	suite.insertSchool(t, schoolID, tenantID, "UC School")
	suite.insertUser(t, userID, tenantID, "uc@test.com")

	yearID := uuid.New().String()
	termID := uuid.New().String()
	suite.insertAcademicYear(t, yearID, tenantID, schoolID, "2026", true, userID)
	suite.insertAcademicTerm(t, termID, tenantID, schoolID, yearID, "Term 1", 1, true, userID)

	// ResolveReferences helper (same as realStudentImporter)
	resolveForUC := func(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []ValidatedRow) ([]ValidatedRow, []RowFailure) {
		resolved := make([]ValidatedRow, len(rows))
		for i, row := range rows {
			var raw struct {
				FullName string `json:"full_name"`
				Gender   string `json:"gender"`
			}
			if err := json.Unmarshal(row.RawData, &raw); err != nil {
				return nil, allIntegrationFail(rows, "bad row")
			}
			aug, _ := json.Marshal(map[string]any{
				"full_name":      raw.FullName,
				"gender":         raw.Gender,
				"tenant_id":      tenantID.String(),
				"school_id":      schoolID.String(),
				"staging_row_id": row.StagingRowID.String(),
			})
			resolved[i] = ValidatedRow{RawData: aug, StagingRowID: row.StagingRowID}
		}
		return resolved, nil
	}

	// ── Create a custom importer that simulates the ON CONFLICT path ────
	imp := &integrationImporter{
		jobType:       ImportJobTypeStudentImport,
		resolveRefsFn: resolveForUC,
		bulkInsertFn: func(c context.Context, tx pgx.Tx, rows []ValidatedRow) (int, error) {
			return 0, fmt.Errorf("force savepoint fallback")
		},
		insertOneFn: func(c context.Context, tx pgx.Tx, row ValidatedRow) error {
			var aug struct {
				FullName     string `json:"full_name"`
				Gender       string `json:"gender"`
				TenantID     string `json:"tenant_id"`
				SchoolID     string `json:"school_id"`
				StagingRowID string `json:"staging_row_id"`
			}
			if err := json.Unmarshal(row.RawData, &aug); err != nil {
				return err
			}

			_, err := tx.Exec(ctx, `
				INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, staging_row_id, is_active)
				VALUES ($1, $2, $3, $4, $5, true)
				ON CONFLICT (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL
				DO UPDATE SET full_name = EXCLUDED.full_name
			`, aug.TenantID, aug.SchoolID, aug.FullName, aug.Gender, aug.StagingRowID)
			return err
		},
	}
	cleanup := registerImporterForTest(t, imp)
	defer cleanup()

	// ── Create job with 3 students ──────────────────────────────────────
	rows := make([]json.RawMessage, 3)
	for i := 0; i < 3; i++ {
		rows[i] = json.RawMessage(`{"full_name":"S` + itoa(i) + `","gender":"M"}`)
	}

	metadata, _ := json.Marshal(map[string]string{
		"academic_term_id": termID,
		"academic_year_id": yearID,
	})

	resp, err := suite.svc.CreateJob(ctx, CreateJobRequest{
		TenantID:  uuid.MustParse(tenantID),
		SchoolID:  uuid.MustParse(schoolID),
		JobType:   ImportJobTypeStudentImport,
		CreatedBy: uuid.MustParse(userID),
		Rows:      rows,
		Metadata:  metadata,
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// ── Get staging row IDs and pre-insert one student ──────────────────
	stagingRows, _ := suite.repo.GetStagingRows(ctx, resp.JobID, 0, 3)
	if len(stagingRows) < 3 {
		t.Fatalf("expected at least 3 staging rows, got %d", len(stagingRows))
	}

	// Pre-insert student for staging row 0 (simulates partial crash)
	// This will cause a unique constraint conflict during ProcessChunk
	_, err = suite.pgPool.Exec(ctx, `
		INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, staging_row_id, is_active)
		VALUES ($1, $2, $3, $4, $5, true)`,
		tenantID, schoolID, "Pre-inserted", "M", stagingRows[0].ID)
	if err != nil {
		t.Fatalf("pre-insert student for conflict test: %v", err)
	}

	// ── Process chunk — the pre-inserted row should NOT cause an error ──
	err = suite.svc.ProcessChunk(ctx, ChunkTaskPayload{
		JobID:          resp.JobID.String(),
		ChunkIndex:     0,
		RowNumberStart: 0,
		RowNumberEnd:   3,
	})
	if err != nil {
		t.Fatalf("ProcessChunk failed when unique constraint expected to be handled: %v", err)
	}

	// ── Assertions ──────────────────────────────────────────────────────
	// Exactly 3 students (not 4 — the ON CONFLICT prevented a duplicate)
	studentCount := suite.queryInt(t, "SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)
	if studentCount != 3 {
		t.Fatalf("expected exactly 3 students (ON CONFLICT prevented duplicate), got %d", studentCount)
	}

	// No failures recorded
	failureCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM import_job_failures WHERE import_job_id = $1", resp.JobID)
	if failureCount != 0 {
		t.Fatalf("expected 0 failures, got %d", failureCount)
	}

	// All staging rows succeeded
	succeededCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM import_job_staging WHERE job_id = $1 AND status = 'succeeded'", resp.JobID)
	if succeededCount != 3 {
		t.Fatalf("expected 3 succeeded staging rows, got %d", succeededCount)
	}

	// Job counters reflect 3 processed
	job, _ := suite.svc.GetJob(ctx, resp.JobID)
	if job.ProcessedRecords != 3 {
		t.Fatalf("expected processed_records=3, got %d", job.ProcessedRecords)
	}
	if job.SuccessCount != 3 {
		t.Fatalf("expected success_count=3 (all treated as successes), got %d", job.SuccessCount)
	}
}

// TestIntegration_ChunkAlreadyClaimed_NoProcessing verifies that a
// redelivery where the chunk is already claimed does no processing.
func TestIntegration_ChunkAlreadyClaimed_NoProcessing(t *testing.T) {
	suite := getIntegrationSuite(t)
	suite.freshDB(t)
	suite.freshRedis(t)
	defer func() { suite.freshRedis(t); suite.freshDB(t) }()

	ctx := context.Background()

	// ── Seed data ───────────────────────────────────────────────────────
	tenantID := uuid.New().String()
	schoolID := uuid.New().String()
	userID := uuid.New().String()
	suite.insertTenant(t, tenantID, "Claimed Tenant")
	suite.insertSchool(t, schoolID, tenantID, "Claimed School")
	suite.insertUser(t, userID, tenantID, "claimed@test.com")

	yearID := uuid.New().String()
	termID := uuid.New().String()
	suite.insertAcademicYear(t, yearID, tenantID, schoolID, "2026", true, userID)
	suite.insertAcademicTerm(t, termID, tenantID, schoolID, yearID, "Term 1", 1, true, userID)

	cleanup := registerImporterForTest(t, newRealStudentImporter())
	defer cleanup()

	// ── Create job ──────────────────────────────────────────────────────
	rows := make([]json.RawMessage, 5)
	for i := 0; i < 5; i++ {
		rows[i] = json.RawMessage(`{"full_name":"S` + itoa(i) + `","gender":"F"}`)
	}

	metadata, _ := json.Marshal(map[string]string{
		"academic_term_id": termID,
		"academic_year_id": yearID,
	})

	resp, _ := suite.svc.CreateJob(ctx, CreateJobRequest{
		TenantID:  uuid.MustParse(tenantID),
		SchoolID:  uuid.MustParse(schoolID),
		JobType:   ImportJobTypeStudentImport,
		CreatedBy: uuid.MustParse(userID),
		Rows:      rows,
		Metadata:  metadata,
	})

	// Claim the chunk manually (simulate another worker)
	claimedID, err := suite.repo.ClaimChunk(ctx, resp.JobID, 0)
	if err != nil {
		t.Fatalf("ClaimChunk: %v", err)
	}
	if claimedID == uuid.Nil {
		t.Fatal("expected chunk to be claimed successfully")
	}

	// Now process the chunk — should skip processing since already claimed
	preStudentCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)

	err = suite.svc.ProcessChunk(ctx, ChunkTaskPayload{
		JobID:          resp.JobID.String(),
		ChunkIndex:     0,
		RowNumberStart: 0,
		RowNumberEnd:   5,
	})
	if err != nil {
		t.Fatalf("ProcessChunk on already-claimed chunk should not error: %v", err)
	}

	// No new students added
	postStudentCount := suite.queryInt(t,
		"SELECT COUNT(*) FROM cbc_students WHERE school_id = $1", schoolID)
	if postStudentCount != preStudentCount {
		t.Fatalf("student count changed from %d to %d — should not have processed", preStudentCount, postStudentCount)
	}
}

// ============================================================================
// helpers
// ============================================================================
// itoa is declared in service_test.go

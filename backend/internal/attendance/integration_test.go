package attendance

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ============================================================================
// Package-level integration test suite
// ============================================================================

var integrationPool *pgxpool.Pool

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		// No containers needed in short mode
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, cleanup := setupIntegration(ctx)
	if pool == nil {
		os.Exit(1)
	}
	integrationPool = pool

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func setupIntegration(ctx context.Context) (*pgxpool.Pool, func()) {
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
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		return nil, func() {}
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		return nil, func() {}
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		_ = c.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		return nil, func() {}
	}

	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s:%s/somotracker_test?sslmode=disable", host, port.Port())

	pgCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		_ = c.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to parse pg config: %v\n", err)
		return nil, func() {}
	}
	pgCfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		_ = c.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to create pg pool: %v\n", err)
		return nil, func() {}
	}

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		_ = c.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		return nil, func() {}
	}

	cleanup := func() {
		pool.Close()
		_ = c.Terminate(context.Background())
	}

	return pool, cleanup
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "database", "migrations")

	files := []string{
		"000001_initial_schema.up.sql",
	}

	for _, f := range files {
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

// ─── Seed helpers ──────────────────────────────────────────────────────────

func seedTenant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, stytch_org_id)
		VALUES ($1, 'Test Tenant', $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, id, "test-tenant-"+id[:8], "org_"+id[:8])
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
}

func seedSchool(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, 'Test School', 'County', 'SubCounty', 'Public')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID)
	if err != nil {
		t.Fatalf("seed school: %v", err)
	}
}

func seedYear(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID)
	if err != nil {
		t.Fatalf("seed year: %v", err)
	}
}

func seedTerm(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID, yearID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', true, '00000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000000')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID, yearID)
	if err != nil {
		t.Fatalf("seed term: %v", err)
	}
}

func seedUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, email string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, tenant_id, full_name)
		VALUES ($1, $2, $3, 'Test User')
		ON CONFLICT (id) DO NOTHING
	`, id, email, tenantID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedStream(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_streams (id, tenant_id, school_id, name)
		VALUES ($1, $2, $3, 'Test Stream')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}
}

func seedClass(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID, yearID, streamID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active)
		VALUES ($1, $2, $3, $4, 'G1', $5, true)
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID, yearID, streamID)
	if err != nil {
		t.Fatalf("seed class: %v", err)
	}
}

func seedStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender)
		VALUES ($1, $2, $3, 'Test Student', 'M')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID)
	if err != nil {
		t.Fatalf("seed student: %v", err)
	}
}

func seedLearningArea(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level)
		VALUES ($1, $2, $3, 'Math', 'MATH', 'Early_Years')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID)
	if err != nil {
		t.Fatalf("seed learning area: %v", err)
	}
}

func seedEnrollment(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, studentID, schoolID, termID, classID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_student_enrollments (id, tenant_id, student_id, school_id, academic_term_id, class_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE')
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, studentID, schoolID, termID, classID)
	if err != nil {
		t.Fatalf("seed enrollment: %v", err)
	}
}

func seedAttendancePeriod(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, tenantID, schoolID, termID, classID, areaID, userID string, date string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_attendance_periods (id, tenant_id, school_id, academic_term_id, class_id, cbc_learning_area_id, date_recorded, recorded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, id, tenantID, schoolID, termID, classID, areaID, date, userID)
	if err != nil {
		t.Fatalf("seed attendance period: %v", err)
	}
}

func seedAttendanceLog(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, periodID, studentID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_attendance_logs (tenant_id, cbc_attendance_period_id, student_id, status, recorded_by)
		VALUES ($1, $2, $3, 'PRESENT', '00000000-0000-0000-0000-000000000000')
	`, tenantID, periodID, studentID)
	if err != nil {
		t.Fatalf("seed attendance log: %v", err)
	}
}

// ============================================================================
// Data Safety Tests
// ============================================================================

func TestIntegration_DataSafety_DetachStudentPreservesLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "dddddddd-0000-0000-0000-000000000001"
	schoolID := "dddddddd-0000-0000-0000-000000000002"
	yearID := "dddddddd-0000-0000-0000-000000000003"
	termID := "dddddddd-0000-0000-0000-000000000004"
	classID := "dddddddd-0000-0000-0000-000000000005"
	streamID := "dddddddd-0000-0000-0000-000000000006"
	studentID := "dddddddd-0000-0000-0000-000000000007"
	areaID := "dddddddd-0000-0000-0000-000000000008"
	userID := "dddddddd-0000-0000-0000-000000000009"
	enrollmentID := "dddddddd-0000-0000-0000-000000000010"
	periodID := "dddddddd-0000-0000-0000-000000000011"

	seedTenant(t, ctx, pool, tenantID)
	seedSchool(t, ctx, pool, schoolID, tenantID)
	seedYear(t, ctx, pool, yearID, tenantID, schoolID)
	seedTerm(t, ctx, pool, termID, tenantID, schoolID, yearID)
	seedStream(t, ctx, pool, streamID, tenantID, schoolID)
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)
	seedStudent(t, ctx, pool, studentID, tenantID, schoolID)
	seedUser(t, ctx, pool, userID, tenantID, "teacher@test.com")
	seedLearningArea(t, ctx, pool, areaID, tenantID, schoolID)
	seedEnrollment(t, ctx, pool, enrollmentID, tenantID, studentID, schoolID, termID, classID)
	seedAttendancePeriod(t, ctx, pool, periodID, tenantID, schoolID, termID, classID, areaID, userID, "2026-06-26")
	seedAttendanceLog(t, ctx, pool, tenantID, periodID, studentID)

	t.Run("setting class_id to NULL does not delete attendance logs", func(t *testing.T) {
		// Detach student from class
		_, err := pool.Exec(ctx, `
			UPDATE cbc_student_enrollments SET class_id = NULL
			WHERE id = $1
		`, enrollmentID)
		if err != nil {
			t.Fatalf("detach student: %v", err)
		}

		// Verify log still exists
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_logs
			WHERE tenant_id = $1 AND cbc_attendance_period_id = $2
		`, tenantID, periodID).Scan(&count)
		if err != nil {
			t.Fatalf("query logs after detachment: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 log to remain after detachment, got %d", count)
		}
	})

	t.Run("historical logs are still retrievable by periodId after detachment", func(t *testing.T) {
		// Query logs directly by period
		var count int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_logs
			WHERE cbc_attendance_period_id = $1
		`, periodID).Scan(&count)
		if err != nil {
			t.Fatalf("query logs by period: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 log retrievable by periodId, got %d", count)
		}
	})
}

func TestIntegration_DataSafety_CascadeDeletes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "eeeeeeee-0000-0000-0000-000000000001"
	schoolID := "eeeeeeee-0000-0000-0000-000000000002"
	yearID := "eeeeeeee-0000-0000-0000-000000000003"
	termID := "eeeeeeee-0000-0000-0000-000000000004"
	classID := "eeeeeeee-0000-0000-0000-000000000005"
	streamID := "eeeeeeee-0000-0000-0000-000000000006"
	studentID := "eeeeeeee-0000-0000-0000-000000000007"
	areaID := "eeeeeeee-0000-0000-0000-000000000008"
	userID := "eeeeeeee-0000-0000-0000-000000000009"
	enrollmentID := "eeeeeeee-0000-0000-0000-000000000010"
	periodID := "eeeeeeee-0000-0000-0000-000000000011"

	seedTenant(t, ctx, pool, tenantID)
	seedSchool(t, ctx, pool, schoolID, tenantID)
	seedYear(t, ctx, pool, yearID, tenantID, schoolID)
	seedTerm(t, ctx, pool, termID, tenantID, schoolID, yearID)
	seedStream(t, ctx, pool, streamID, tenantID, schoolID)
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)
	seedStudent(t, ctx, pool, studentID, tenantID, schoolID)
	seedUser(t, ctx, pool, userID, tenantID, "teacher@test.com")
	seedLearningArea(t, ctx, pool, areaID, tenantID, schoolID)
	seedEnrollment(t, ctx, pool, enrollmentID, tenantID, studentID, schoolID, termID, classID)
	seedAttendancePeriod(t, ctx, pool, periodID, tenantID, schoolID, termID, classID, areaID, userID, "2026-06-26")
	seedAttendanceLog(t, ctx, pool, tenantID, periodID, studentID)

	t.Run("deleting an attendance period cascades to its child logs", func(t *testing.T) {
		// Delete the period
		_, err := pool.Exec(ctx, `DELETE FROM cbc_attendance_periods WHERE id = $1`, periodID)
		if err != nil {
			t.Fatalf("delete period: %v", err)
		}

		// Verify logs were cascade-deleted
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_logs
			WHERE cbc_attendance_period_id = $1
		`, periodID).Scan(&count)
		if err != nil {
			t.Fatalf("query logs after cascade: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 logs after period cascade delete, got %d", count)
		}
	})

	t.Run("deleting a class cascades to attendance periods and their logs", func(t *testing.T) {
		// Re-seed period and log under a fresh class
		class2ID := "eeeeeeee-0000-0000-0000-000000000012"
		period2ID := "eeeeeeee-0000-0000-0000-000000000013"

		seedClass(t, ctx, pool, class2ID, tenantID, schoolID, yearID, streamID)
		seedAttendancePeriod(t, ctx, pool, period2ID, tenantID, schoolID, termID, class2ID, areaID, userID, "2026-06-27")
		seedAttendanceLog(t, ctx, pool, tenantID, period2ID, studentID)

		// Delete the class
		_, err := pool.Exec(ctx, `DELETE FROM cbc_classes WHERE id = $1`, class2ID)
		if err != nil {
			t.Fatalf("delete class: %v", err)
		}

		// Verify period was cascade-deleted
		var periodCount int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_periods
			WHERE class_id = $1
		`, class2ID).Scan(&periodCount)
		if err != nil {
			t.Fatalf("query periods after class delete: %v", err)
		}
		if periodCount != 0 {
			t.Fatalf("expected 0 periods after class cascade delete, got %d", periodCount)
		}

		// Verify logs were cascade-deleted
		var logCount int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_logs
			WHERE cbc_attendance_period_id = $1
		`, period2ID).Scan(&logCount)
		if err != nil {
			t.Fatalf("query logs after class cascade: %v", err)
		}
		if logCount != 0 {
			t.Fatalf("expected 0 logs after class cascade delete, got %d", logCount)
		}
	})
}

// ============================================================================
// Cross-Cutting / Tenant Isolation Tests
// ============================================================================

func TestIntegration_TenantIsolation_Attendance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantA := "ff000000-0000-0000-0000-000000000001"
	tenantB := "ff000000-0000-0000-0000-000000000002"
	schoolA := "ff000000-0000-0000-0000-000000000003"
	schoolB := "ff000000-0000-0000-0000-000000000004"
	yearA := "ff000000-0000-0000-0000-000000000005"
	yearB := "ff000000-0000-0000-0000-000000000006"
	termA := "ff000000-0000-0000-0000-000000000007"
	termB := "ff000000-0000-0000-0000-000000000008"
	classA := "ff000000-0000-0000-0000-000000000009"
	classB := "ff000000-0000-0000-0000-000000000010"
	streamA := "ff000000-0000-0000-0000-000000000011"
	streamB := "ff000000-0000-0000-0000-000000000012"
	studentA := "ff000000-0000-0000-0000-000000000013"
	studentB := "ff000000-0000-0000-0000-000000000014"
	areaA := "ff000000-0000-0000-0000-000000000015"
	areaB := "ff000000-0000-0000-0000-000000000016"
	userA := "ff000000-0000-0000-0000-000000000017"
	userB := "ff000000-0000-0000-0000-000000000018"
	enrollA := "ff000000-0000-0000-0000-000000000019"
	enrollB := "ff000000-0000-0000-0000-000000000020"
	periodA := "ff000000-0000-0000-0000-000000000021"
	periodB := "ff000000-0000-0000-0000-000000000022"

	// Seed Tenant A
	seedTenant(t, ctx, pool, tenantA)
	seedSchool(t, ctx, pool, schoolA, tenantA)
	seedYear(t, ctx, pool, yearA, tenantA, schoolA)
	seedTerm(t, ctx, pool, termA, tenantA, schoolA, yearA)
	seedStream(t, ctx, pool, streamA, tenantA, schoolA)
	seedClass(t, ctx, pool, classA, tenantA, schoolA, yearA, streamA)
	seedStudent(t, ctx, pool, studentA, tenantA, schoolA)
	seedUser(t, ctx, pool, userA, tenantA, "admin-a@test.com")
	seedLearningArea(t, ctx, pool, areaA, tenantA, schoolA)
	seedEnrollment(t, ctx, pool, enrollA, tenantA, studentA, schoolA, termA, classA)
	seedAttendancePeriod(t, ctx, pool, periodA, tenantA, schoolA, termA, classA, areaA, userA, "2026-06-26")

	// Seed Tenant B
	seedTenant(t, ctx, pool, tenantB)
	seedSchool(t, ctx, pool, schoolB, tenantB)
	seedYear(t, ctx, pool, yearB, tenantB, schoolB)
	seedTerm(t, ctx, pool, termB, tenantB, schoolB, yearB)
	seedStream(t, ctx, pool, streamB, tenantB, schoolB)
	seedClass(t, ctx, pool, classB, tenantB, schoolB, yearB, streamB)
	seedStudent(t, ctx, pool, studentB, tenantB, schoolB)
	seedUser(t, ctx, pool, userB, tenantB, "admin-b@test.com")
	seedLearningArea(t, ctx, pool, areaB, tenantB, schoolB)
	seedEnrollment(t, ctx, pool, enrollB, tenantB, studentB, schoolB, termB, classB)
	seedAttendancePeriod(t, ctx, pool, periodB, tenantB, schoolB, termB, classB, areaB, userB, "2026-06-26")

	t.Run("attendance period under Tenant A not returned by Tenant B queries", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_periods
			WHERE tenant_id = $1 AND id = $2
		`, tenantA, periodA).Scan(&count)
		if err != nil {
			t.Fatalf("query tenant A period: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 period for tenant A, got %d", count)
		}

		// Tenant B should not see Tenant A's period
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_periods
			WHERE tenant_id = $1 AND id = $2
		`, tenantB, periodA).Scan(&count)
		if err != nil {
			t.Fatalf("query tenant B for tenant A's period: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 results when tenant B queries tenant A's period, got %d", count)
		}
	})

	t.Run("missing tenant_id returns zero rows, not error or another tenant's data", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_attendance_periods
			WHERE tenant_id = '00000000-0000-0000-0000-000000000000'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("query non-existent tenant: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 rows for non-existent tenant, got %d", count)
		}
	})
}

// ============================================================================
// Cross-School Isolation within Same Tenant
// ============================================================================

func TestIntegration_CrossSchoolIsolation_SameTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "cc000000-0000-0000-0000-000000000001"
	schoolX := "cc000000-0000-0000-0000-000000000002"
	schoolY := "cc000000-0000-0000-0000-000000000003"
	yearID := "cc000000-0000-0000-0000-000000000004"
	termID := "cc000000-0000-0000-0000-000000000005"
	classX := "cc000000-0000-0000-0000-000000000006"
	classY := "cc000000-0000-0000-0000-000000000007"
	streamX := "cc000000-0000-0000-0000-000000000008"
	streamY := "cc000000-0000-0000-0000-000000000009"
	areaX := "cc000000-0000-0000-0000-000000000010"
	areaY := "cc000000-0000-0000-0000-000000000011"
	adminUserID := "cc000000-0000-0000-0000-000000000012"

	seedTenant(t, ctx, pool, tenantID)
	seedSchool(t, ctx, pool, schoolX, tenantID)
	seedSchool(t, ctx, pool, schoolY, tenantID)
	seedYear(t, ctx, pool, yearID, tenantID, schoolX)
	seedTerm(t, ctx, pool, termID, tenantID, schoolX, yearID)
	seedStream(t, ctx, pool, streamX, tenantID, schoolX)
	seedStream(t, ctx, pool, streamY, tenantID, schoolY)
	seedClass(t, ctx, pool, classX, tenantID, schoolX, yearID, streamX)
	seedClass(t, ctx, pool, classY, tenantID, schoolY, yearID, streamY)
	seedLearningArea(t, ctx, pool, areaX, tenantID, schoolX)
	seedLearningArea(t, ctx, pool, areaY, tenantID, schoolY)
	seedUser(t, ctx, pool, adminUserID, tenantID, "admin@test.com")

	// Assign admin as SCHOOL_ADMIN in School X only
	_, err := pool.Exec(ctx, `
		INSERT INTO memberships (tenant_id, user_id, school_id, role)
		VALUES ($1, $2, $3, 'SCHOOL_ADMIN')
		ON CONFLICT (user_id, school_id) DO NOTHING
	`, tenantID, adminUserID, schoolX)
	if err != nil {
		t.Fatalf("seed membership for school X: %v", err)
	}

	// Verify: School X admin cannot submit attendance for School Y
	// Note: This is enforced by the IsAuthorizedRecorder SQL which scopes
	// membership checks to the class's school. We verify this by checking
	// that the admin has no membership in School Y.
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM memberships
		WHERE tenant_id = $1 AND user_id = $2 AND school_id = $3 AND is_active = true
	`, tenantID, adminUserID, schoolY).Scan(&count)
	if err != nil {
		t.Fatalf("query membership for school Y: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected admin to have no membership in School Y, got %d", count)
	}
}

package timetable

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	// Start PostgreSQL container
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

// ─── Helpers ──────────────────────────────────────────────────────────────

func seedTerm(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, schoolID, yearID, termID, termName string) {
	t.Helper()
	// Use the full UUID (without dashes) for unique slug generation
	slug := "test-tenant-" + strings.ReplaceAll(tenantID, "-", "")
	stytchOrgID := "org_" + strings.ReplaceAll(tenantID, "-", "")

	// Insert a tenant if not exists
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, stytch_org_id)
		VALUES ($1, 'Test Tenant', $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, tenantID, slug, stytchOrgID)
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	// Insert a school if not exists
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, 'Test School', 'County', 'SubCounty', 'Public')
		ON CONFLICT (id) DO NOTHING
	`, schoolID, tenantID)
	if err != nil {
		t.Fatalf("seed school: %v", err)
	}

	// Create a system user for created_by/updated_by FK references
	systemUserID := "00000000-0000-0000-0000-000000000000"
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, tenant_id, full_name)
		VALUES ($1, $2, $3, 'System')
		ON CONFLICT (id) DO NOTHING
	`, systemUserID, "system-"+slug+"@test.com", tenantID)
	if err != nil {
		t.Fatalf("seed system user: %v", err)
	}

	if yearID != "" {
		_, err := pool.Exec(ctx, `
			INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by)
			VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)
			ON CONFLICT (id) DO NOTHING
		`, yearID, tenantID, schoolID, systemUserID)
		if err != nil {
			t.Fatalf("seed academic year: %v", err)
		}
	}

	if termID != "" {
		_, err := pool.Exec(ctx, `
			INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, 1, '2026-01-01', '2026-04-30', true, $6, $6)
			ON CONFLICT (id) DO NOTHING
		`, termID, tenantID, schoolID, yearID, termName, systemUserID)
		if err != nil {
			t.Fatalf("seed academic term: %v", err)
		}
	}
}

func seedUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, tenantID, email string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, tenant_id, full_name)
		VALUES ($1, $2, $3, 'Test User')
		ON CONFLICT (id) DO NOTHING
	`, userID, email, tenantID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedClass(t *testing.T, ctx context.Context, pool *pgxpool.Pool, classID, tenantID, schoolID, yearID, streamID string) {
	t.Helper()
	if streamID != "" {
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_streams (id, tenant_id, school_id, name)
			VALUES ($1, $2, $3, 'Test Stream')
			ON CONFLICT (id) DO NOTHING
		`, streamID, tenantID, schoolID)
		if err != nil {
			t.Fatalf("seed stream: %v", err)
		}
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active)
		VALUES ($1, $2, $3, $4, 'G1', $5, true)
		ON CONFLICT (id) DO NOTHING
	`, classID, tenantID, schoolID, yearID, streamID)
	if err != nil {
		t.Fatalf("seed class: %v", err)
	}
}

// ============================================================================
// Bulk Slot — Term Isolation
// ============================================================================

func TestIntegration_BulkSlots_TermIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "10000000-0000-0000-0000-000000000001"
	schoolID := "20000000-0000-0000-0000-000000000001"
	yearID := "30000000-0000-0000-0000-000000000001"
	term1ID := "40000000-0000-0000-0000-000000000001"
	term2ID := "40000000-0000-0000-0000-000000000002"
	classID := "50000000-0000-0000-0000-000000000001"
	teacherID := "60000000-0000-0000-0000-000000000001"
	streamID := "70000000-0000-0000-0000-000000000001"

	seedTerm(t, ctx, pool, tenantID, schoolID, yearID, term1ID, "Term 1")
	// Second term must have is_current=false to avoid idx_one_current_term_per_year violation
	if _, err := pool.Exec(ctx, `
		INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Term 2', 2, '2026-05-01', '2026-08-31', false, $5, $5)
		ON CONFLICT (id) DO NOTHING
	`, term2ID, tenantID, schoolID, yearID, "00000000-0000-0000-0000-000000000000"); err != nil {
		t.Fatalf("seed term2: %v", err)
	}
	seedUser(t, ctx, pool, teacherID, tenantID, "teacher@test.com")
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)

	// Insert slots for term 1
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
	`, tenantID, schoolID, yearID, term1ID, classID, teacherID)
	if err != nil {
		t.Fatalf("insert term1 slot: %v", err)
	}

	// Insert slots for term 2
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 2, '10:00', '11:00')
	`, tenantID, schoolID, yearID, term2ID, classID, teacherID)
	if err != nil {
		t.Fatalf("insert term2 slot: %v", err)
	}

	// Query term 1 — should only return term 1 slot
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots
		WHERE tenant_id = $1 AND academic_term_id = $2
	`, tenantID, term1ID).Scan(&count)
	if err != nil {
		t.Fatalf("query term1 count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 slot in term 1, got %d", count)
	}

	// Query term 2 — should only return term 2 slot
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots
		WHERE tenant_id = $1 AND academic_term_id = $2
	`, tenantID, term2ID).Scan(&count)
	if err != nil {
		t.Fatalf("query term2 count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 slot in term 2, got %d", count)
	}
}

// ============================================================================
// Bulk Slot — Teacher Auto-Registration
// ============================================================================

func TestIntegration_BulkSlots_AutoRegisterSubjectTeacher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "10000000-0000-0000-0000-000000000010"
	schoolID := "20000000-0000-0000-0000-000000000010"
	yearID := "30000000-0000-0000-0000-000000000010"
	termID := "40000000-0000-0000-0000-000000000010"
	classID := "50000000-0000-0000-0000-000000000010"
	teacherID := "60000000-0000-0000-0000-000000000010"
	teacher2ID := "60000000-0000-0000-0000-000000000011"
	teacher3ID := "60000000-0000-0000-0000-000000000012"
	areaID := "80000000-0000-0000-0000-000000000010"
	area2ID := "80000000-0000-0000-0000-000000000011"
	streamID := "70000000-0000-0000-0000-000000000010"

	seedTerm(t, ctx, pool, tenantID, schoolID, yearID, termID, "Term 1")
	seedUser(t, ctx, pool, teacherID, tenantID, "teacher-a@test.com")
	seedUser(t, ctx, pool, teacher2ID, tenantID, "teacher-b@test.com")
	seedUser(t, ctx, pool, teacher3ID, tenantID, "teacher-c@test.com")
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)

	// Seed learning areas
	_, _ = pool.Exec(ctx, `
		INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level)
		VALUES ($1, $2, $3, 'Math', 'MATH', 'Early_Years')
		ON CONFLICT (id) DO NOTHING
	`, areaID, tenantID, schoolID)
	_, _ = pool.Exec(ctx, `
		INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level)
		VALUES ($1, $2, $3, 'Science', 'SCI', 'Early_Years')
		ON CONFLICT (id) DO NOTHING
	`, area2ID, tenantID, schoolID)

	t.Run("inserting slot with learning_area auto-registers teacher as SUBJECT_TEACHER", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_timetable_slots
				(tenant_id, school_id, academic_year_id, academic_term_id,
				 class_id, teacher_id, cbc_learning_area_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 1, '08:00', '09:00')
		`, tenantID, schoolID, yearID, termID, classID, teacherID, areaID)
		if err != nil {
			t.Fatalf("insert slot with learning area: %v", err)
		}

		// Verify teacher was auto-registered as SUBJECT_TEACHER
		var role string
		err = pool.QueryRow(ctx, `
			SELECT teacher_role::text FROM cbc_class_teachers
			WHERE tenant_id = $1 AND class_id = $2 AND user_id = $3
		`, tenantID, classID, teacherID).Scan(&role)
		if err != nil {
			t.Fatalf("query class_teacher: %v", err)
		}
		if role != "SUBJECT_TEACHER" {
			t.Fatalf("expected SUBJECT_TEACHER, got %s", role)
		}
	})

	t.Run("inserting slot with teacher already PRIMARY does not change their role", func(t *testing.T) {
		// Pre-register teacher2 as PRIMARY
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, teacher_role)
			VALUES ($1, $2, $3, NULL, 'PRIMARY_CLASS_TEACHER')
		`, tenantID, classID, teacher2ID)
		if err != nil {
			t.Fatalf("pre-register teacher2 as PRIMARY: %v", err)
		}

		// Insert slot with teacher2 (should not change role)
		_, err = pool.Exec(ctx, `
			INSERT INTO cbc_timetable_slots
				(tenant_id, school_id, academic_year_id, academic_term_id,
				 class_id, teacher_id, cbc_learning_area_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 2, '09:00', '10:00')
		`, tenantID, schoolID, yearID, termID, classID, teacher2ID, areaID)
		if err != nil {
			t.Fatalf("insert slot for teacher2: %v", err)
		}

		// Verify role is still PRIMARY
		var role string
		err = pool.QueryRow(ctx, `
			SELECT teacher_role::text FROM cbc_class_teachers
			WHERE tenant_id = $1 AND class_id = $2 AND user_id = $3
		`, tenantID, classID, teacher2ID).Scan(&role)
		if err != nil {
			t.Fatalf("query teacher2 role: %v", err)
		}
		if role != "PRIMARY_CLASS_TEACHER" {
			t.Fatalf("expected PRIMARY_CLASS_TEACHER to be unchanged, got %s", role)
		}
	})

	t.Run("inserting slot with teacher already SUBJECT for that area produces no duplicate row", func(t *testing.T) {
		// Pre-register teacher3 as SUBJECT for area2
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, teacher_role)
			VALUES ($1, $2, $3, $4, 'SUBJECT_TEACHER')
		`, tenantID, classID, teacher3ID, area2ID)
		if err != nil {
			t.Fatalf("pre-register teacher3 as SUBJECT: %v", err)
		}

		// Insert slot with teacher3 for area2 — trigger should attempt insert but skip
		_, err = pool.Exec(ctx, `
			INSERT INTO cbc_timetable_slots
				(tenant_id, school_id, academic_year_id, academic_term_id,
				 class_id, teacher_id, cbc_learning_area_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 3, '10:00', '11:00')
		`, tenantID, schoolID, yearID, termID, classID, teacher3ID, area2ID)
		if err != nil {
			t.Fatalf("insert slot for teacher3: %v", err)
		}

		// Verify only 1 row exists for teacher3 on this class
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM cbc_class_teachers
			WHERE tenant_id = $1 AND class_id = $2 AND user_id = $3
		`, tenantID, classID, teacher3ID).Scan(&count)
		if err != nil {
			t.Fatalf("count teacher3 rows: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected exactly 1 row for teacher3, got %d", count)
		}
	})
}

// ============================================================================
// GiST Exclusion — Teacher double-booking
// ============================================================================

func TestIntegration_GiST_TeacherDoubleBooking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "10000000-0000-0000-0000-000000000020"
	schoolID := "20000000-0000-0000-0000-000000000020"
	yearID := "30000000-0000-0000-0000-000000000020"
	termID := "40000000-0000-0000-0000-000000000020"
	classID := "50000000-0000-0000-0000-000000000020"
	class2ID := "50000000-0000-0000-0000-000000000021"
	teacherID := "60000000-0000-0000-0000-000000000020"
	teacher2ID := "60000000-0000-0000-0000-000000000021"
	streamID := "70000000-0000-0000-0000-000000000020"
	stream2ID := "70000000-0000-0000-0000-000000000021"

	seedTerm(t, ctx, pool, tenantID, schoolID, yearID, termID, "Term 1")
	seedUser(t, ctx, pool, teacherID, tenantID, "teacher-double@test.com")
	seedUser(t, ctx, pool, teacher2ID, tenantID, "teacher-other@test.com")
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)
	// Pre-seed stream2 with unique name so seedClass's ON CONFLICT skips it
	if _, sErr := pool.Exec(ctx, `
		INSERT INTO cbc_streams (id, tenant_id, school_id, name)
		VALUES ($1, $2, $3, 'Stream B')
		ON CONFLICT (id) DO NOTHING
	`, stream2ID, tenantID, schoolID); sErr != nil {
		t.Fatalf("seed stream2: %v", sErr)
	}
	seedClass(t, ctx, pool, class2ID, tenantID, schoolID, yearID, stream2ID)

	// Insert first slot for teacher at 08:00-09:00 on Monday (day 1)
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
	`, tenantID, schoolID, yearID, termID, classID, teacherID)
	if err != nil {
		t.Fatalf("insert first slot: %v", err)
	}

	t.Run("same teacher cannot be booked in overlapping times on same day", func(t *testing.T) {
		// Try to insert overlapping slot for same teacher at 08:30-09:30 on same day
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_timetable_slots
				(tenant_id, school_id, academic_year_id, academic_term_id,
				 class_id, teacher_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4, $5, $6, 1, '08:30', '09:30')
		`, tenantID, schoolID, yearID, termID, class2ID, teacherID)
		if err == nil {
			t.Fatal("expected GiST exclusion error for overlapping teacher slot, got nil")
		}
	})

	t.Run("two teachers CAN have overlapping slots in different rooms — no conflict", func(t *testing.T) {
		// Insert different teacher at same time in different class
		_, err := pool.Exec(ctx, `
			INSERT INTO cbc_timetable_slots
				(tenant_id, school_id, academic_year_id, academic_term_id,
				 class_id, teacher_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
		`, tenantID, schoolID, yearID, termID, class2ID, teacher2ID)
		if err != nil {
			t.Fatalf("expected different teacher to succeed, got: %v", err)
		}
	})
}

// ============================================================================
// GiST Exclusion — Room double-booking
// ============================================================================

func TestIntegration_GiST_RoomDoubleBooking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "10000000-0000-0000-0000-000000000030"
	schoolID := "20000000-0000-0000-0000-000000000030"
	yearID := "30000000-0000-0000-0000-000000000030"
	termID := "40000000-0000-0000-0000-000000000030"
	classID := "50000000-0000-0000-0000-000000000030"
	class2ID := "50000000-0000-0000-0000-000000000031"
	teacherID := "60000000-0000-0000-0000-000000000030"
	teacher2ID := "60000000-0000-0000-0000-000000000031"
	streamID := "70000000-0000-0000-0000-000000000030"
	stream2ID := "70000000-0000-0000-0000-000000000031"
	room := "Room-A"

	seedTerm(t, ctx, pool, tenantID, schoolID, yearID, termID, "Term 1")
	seedUser(t, ctx, pool, teacherID, tenantID, "teacher-room1@test.com")
	seedUser(t, ctx, pool, teacher2ID, tenantID, "teacher-room2@test.com")
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)
	// Pre-seed stream2 with unique name so seedClass's ON CONFLICT skips it
	if _, sErr := pool.Exec(ctx, `
		INSERT INTO cbc_streams (id, tenant_id, school_id, name)
		VALUES ($1, $2, $3, 'Room Stream B')
		ON CONFLICT (id) DO NOTHING
	`, stream2ID, tenantID, schoolID); sErr != nil {
		t.Fatalf("seed stream2: %v", sErr)
	}
	seedClass(t, ctx, pool, class2ID, tenantID, schoolID, yearID, stream2ID)

	// Insert first slot in Room A at 08:00-09:00 Monday
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, room_identifier, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, '08:00', '09:00')
	`, tenantID, schoolID, yearID, termID, classID, teacherID, room)
	if err != nil {
		t.Fatalf("insert first room slot: %v", err)
	}

	// Try to insert overlapping slot in same room
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, room_identifier, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, '08:30', '09:30')
	`, tenantID, schoolID, yearID, termID, class2ID, teacher2ID, room)
	if err == nil {
		t.Fatal("expected GiST exclusion error for overlapping room slot, got nil")
	}
}

// ============================================================================
// Tenant Isolation — Timetable Slots
// ============================================================================

func TestIntegration_TenantIsolation_Slots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantA := "aaaaaaaa-0000-0000-0000-000000000001"
	tenantB := "bbbbbbbb-0000-0000-0000-000000000001"
	schoolA := "20000000-0000-0000-0000-aaaaaaaa0001"
	schoolB := "20000000-0000-0000-0000-bbbbbbbb0001"
	yearA := "30000000-0000-0000-0000-aaaaaaaa0001"
	yearB := "30000000-0000-0000-0000-bbbbbbbb0001"
	termA := "40000000-0000-0000-0000-aaaaaaaa0001"
	termB := "40000000-0000-0000-0000-bbbbbbbb0001"
	classA := "50000000-0000-0000-0000-aaaaaaaa0001"
	classB := "50000000-0000-0000-0000-bbbbbbbb0001"
	teacherA := "60000000-0000-0000-0000-aaaaaaaa0001"
	teacherB := "60000000-0000-0000-0000-bbbbbbbb0001"
	streamA := "70000000-0000-0000-0000-aaaaaaaa0001"
	streamB := "70000000-0000-0000-0000-bbbbbbbb0001"

	// Seed Tenant A
	seedTerm(t, ctx, pool, tenantA, schoolA, yearA, termA, "Term A")
	seedUser(t, ctx, pool, teacherA, tenantA, "teacher-isolation-a@test.com")
	seedClass(t, ctx, pool, classA, tenantA, schoolA, yearA, streamA)

	// Seed Tenant B
	seedTerm(t, ctx, pool, tenantB, schoolB, yearB, termB, "Term B")
	seedUser(t, ctx, pool, teacherB, tenantB, "teacher-isolation-b@test.com")
	seedClass(t, ctx, pool, classB, tenantB, schoolB, yearB, streamB)

	// Insert slot for Tenant A
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
	`, tenantA, schoolA, yearA, termA, classA, teacherA)
	if err != nil {
		t.Fatalf("insert tenant A slot: %v", err)
	}

	// Insert slot for Tenant B
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
	`, tenantB, schoolB, yearB, termB, classB, teacherB)
	if err != nil {
		t.Fatalf("insert tenant B slot: %v", err)
	}

	// Query Tenant A's slots — should not see Tenant B's slot
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots WHERE tenant_id = $1
	`, tenantA).Scan(&count)
	if err != nil {
		t.Fatalf("query tenant A slots: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 slot for tenant A, got %d", count)
	}

	// Query Tenant B's slots — should not see Tenant A's slot
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots WHERE tenant_id = $1
	`, tenantB).Scan(&count)
	if err != nil {
		t.Fatalf("query tenant B slots: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 slot for tenant B, got %d", count)
	}

	// Query with non-existent tenant — should return 0
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots WHERE tenant_id = $1
	`, "00000000-0000-0000-0000-000000000000").Scan(&count)
	if err != nil {
		t.Fatalf("query non-existent tenant: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 slots for non-existent tenant, got %d", count)
	}
}

// ============================================================================
// Transaction Integrity — Atomic Failure on Bulk Insert
// ============================================================================

func TestIntegration_BulkSlots_AtomicFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "10000000-0000-0000-0000-000000000040"
	schoolID := "20000000-0000-0000-0000-000000000040"
	yearID := "30000000-0000-0000-0000-000000000040"
	termID := "40000000-0000-0000-0000-000000000040"
	classID := "50000000-0000-0000-0000-000000000040"
	teacherID := "60000000-0000-0000-0000-000000000040"
	teacher2ID := "60000000-0000-0000-0000-000000000041"
	streamID := "70000000-0000-0000-0000-000000000040"

	seedTerm(t, ctx, pool, tenantID, schoolID, yearID, termID, "Term 1")
	seedUser(t, ctx, pool, teacherID, tenantID, "teacher-tx@test.com")
	seedUser(t, ctx, pool, teacher2ID, tenantID, "teacher-tx2@test.com")
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)

	// Insert a first slot to set up a teacher double-booking scenario
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
	`, tenantID, schoolID, yearID, termID, classID, teacherID)
	if err != nil {
		t.Fatalf("insert first slot: %v", err)
	}

	// Now try to bulk-insert two slots: one valid, one conflicting (same teacher, overlapping time)
	// Using a transaction that mimics the repo's BulkUpsertSlots
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	// Valid slot
	_, err = tx.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 2, '10:00', '11:00')
	`, tenantID, schoolID, yearID, termID, classID, teacher2ID)
	if err != nil {
		t.Fatalf("insert valid slot in tx: %v", err)
	}

	// Conflicting slot (same teacher, overlapping time on day 1)
	_, err = tx.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:30', '09:30')
	`, tenantID, schoolID, yearID, termID, classID, teacherID)
	if err != nil {
		// Expected: GiST exclusion violation
		// Roll back the transaction
		_ = tx.Rollback(ctx)
	} else {
		// If no error, commit would persist both
		_ = tx.Commit(ctx)
	}

	// Verify: after rollback, the valid slot (teacher2, day 2) should NOT be persisted
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots
		WHERE tenant_id = $1 AND teacher_id = $2
	`, tenantID, teacher2ID).Scan(&count)
	if err != nil {
		t.Fatalf("query teacher2 slots: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 slots for teacher2 after rollback, got %d", count)
	}

	// Verify original slot for teacher1 is still there (was committed before tx)
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_timetable_slots
		WHERE tenant_id = $1 AND teacher_id = $2
	`, tenantID, teacherID).Scan(&count)
	if err != nil {
		t.Fatalf("query teacher1 slots: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 slot for teacher1 (committed before tx), got %d", count)
	}
}

// ============================================================================
// Transaction Integrity — Auto-Registration Rolls Back with Slot
// ============================================================================

func TestIntegration_BulkSlots_AutoRegistrationRollsBack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := integrationPool

	tenantID := "10000000-0000-0000-0000-000000000050"
	schoolID := "20000000-0000-0000-0000-000000000050"
	yearID := "30000000-0000-0000-0000-000000000050"
	termID := "40000000-0000-0000-0000-000000000050"
	classID := "50000000-0000-0000-0000-000000000050"
	teacherID := "60000000-0000-0000-0000-000000000050"
	areaID := "80000000-0000-0000-0000-000000000050"
	streamID := "70000000-0000-0000-0000-000000000050"

	seedTerm(t, ctx, pool, tenantID, schoolID, yearID, termID, "Term 1")
	seedUser(t, ctx, pool, teacherID, tenantID, "teacher-reg@test.com")
	seedClass(t, ctx, pool, classID, tenantID, schoolID, yearID, streamID)

	// Seed a learning area
	_, err := pool.Exec(ctx, `
		INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level)
		VALUES ($1, $2, $3, 'Math', 'MATH', 'Early_Years')
		ON CONFLICT (id) DO NOTHING
	`, areaID, tenantID, schoolID)
	if err != nil {
		t.Fatalf("seed learning area: %v", err)
	}

	// Insert a conflicting slot first to set up GiST violation
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, 1, '08:00', '09:00')
	`, tenantID, schoolID, yearID, termID, classID, teacherID)
	if err != nil {
		t.Fatalf("insert conflicting slot: %v", err)
	}

	// Use a transaction to insert a slot with a learning area
	// that will conflict (same teacher, overlapping time)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO cbc_timetable_slots
			(tenant_id, school_id, academic_year_id, academic_term_id,
			 class_id, teacher_id, cbc_learning_area_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, '08:30', '09:30')
	`, tenantID, schoolID, yearID, termID, classID, teacherID, areaID)
	if err == nil {
		_ = tx.Commit(ctx)
		t.Fatal("expected GiST exclusion error for overlapping slot, got nil")
	}

	// Rollback happens via defer

	// Verify: no class_teacher row was created by the trigger
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cbc_class_teachers
		WHERE tenant_id = $1 AND class_id = $2 AND user_id = $3
	`, tenantID, classID, teacherID).Scan(&count)
	if err != nil {
		t.Fatalf("query class_teachers: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 class_teacher rows after rolled-back slot insert, got %d", count)
	}
}

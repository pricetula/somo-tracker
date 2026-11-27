//go:build integration

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"somotracker/internal/platform/db"
)

// TestMigrations_ApplyAllAgainstRealDB applies every SQL migration in order
// against a real PostgreSQL database (testcontainers, postgres:18) and verifies
// the resulting schema via catalog/information_schema queries.
//
// postgres:18 is the latest major release and matches the project standard
// (docker-compose.yml + AGENTS.md). No mocks are used.
func TestMigrations_ApplyAllAgainstRealDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration integration test in short mode")
	}

	ctx := context.Background()

	// --- PostgreSQL (latest = postgres:18, matches docker-compose.yml) ---
	pgContainer, err := postgres.Run(ctx,
		"postgres:18",
		postgres.WithDatabase("somotracker"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "failed to start PostgreSQL container")
	t.Cleanup(func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	})

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get PostgreSQL connection string")

	pool, err := db.New(ctx, pgDSN)
	require.NoError(t, err, "failed to connect to test database")
	defer pool.Close()

	// --- Apply all migrations in order ---
	require.NoError(t, runMigrations(ctx, t, pool.StdlibDB), "failed to apply migrations")

	// --- Verify schema integrity via catalog queries ---
	verifySchemaIntegrity(t, ctx, pool.StdlibDB)

	// --- Functional end-to-end: foreign key relationships work ---
	testForeignKeyIntegrity(t, ctx, pool.StdlibDB)
}

// runMigrations reads every *.sql file in the migrations directory (sorted by
// name) and executes only the "-- +goose Up" portion against the database.
func runMigrations(ctx context.Context, t *testing.T, stdlibDB *sql.DB) error {
	t.Helper()

	migrationsDir := migrationsDir()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	require.NotEmpty(t, files, "no migration files found in %s", migrationsDir)

	for _, name := range files {
		path := filepath.Join(migrationsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		sqlContent := string(content)
		// Keep only the "Up" portion (everything before "-- +goose Down").
		if idx := strings.Index(sqlContent, "-- +goose Down"); idx != -1 {
			sqlContent = sqlContent[:idx]
		}
		// Strip goose directive lines.
		var lines []string
		for _, line := range strings.Split(sqlContent, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "-- +goose") {
				continue
			}
			lines = append(lines, line)
		}
		sqlContent = strings.TrimSpace(strings.Join(lines, "\n"))
		if sqlContent == "" {
			continue
		}

		t.Logf("Applying migration: %s", name)
		if _, err := stdlibDB.ExecContext(ctx, sqlContent); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

// migrationsDir resolves the backend/migrations directory relative to this test
// file so the test works regardless of the current working directory.
func migrationsDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "./migrations"
	}
	dir := filepath.Dir(filename)
	// backend/internal/database -> backend/migrations
	candidate := filepath.Join(dir, "..", "..", "migrations")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return "./migrations"
}

// intScalar runs a query returning a single integer.
func intScalar(ctx context.Context, stdlibDB *sql.DB, query string, args ...interface{}) (int, error) {
	var n int
	err := stdlibDB.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

func verifySchemaIntegrity(t *testing.T, ctx context.Context, stdlibDB *sql.DB) {
	t.Helper()

	// --- Tables ---
	expectedTables := []string{
		// core domain
		"organizations", "users", "schools", "user_schools",
		// auth domain
		"login_challenges", "email_outbox", "sessions",
		// countries / education system
		"countries", "education_systems", "education_levels",
		"grading_scales", "grading_bands",
		// subjects / terms
		"subjects", "topics", "sub_topics",
		// school config
		"school_subjects", "academic_years", "academic_terms", "grade_years", "streams", "classes",
		// domain (10)
		"teachers", "students", "enrollments", "assessment_types", "assessments", "assessment_results",
		"guardians", "student_guardians", "attendance_records", "teacher_qualifications", "consent_records",
	}
	for _, table := range expectedTables {
		n, err := intScalar(ctx, stdlibDB, `
			SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1`, table)
		require.NoError(t, err, "query table %s", table)
		require.Equal(t, 1, n, "expected table %s to exist", table)
	}

	// --- Enums ---
	expectedEnums := []string{"organization_status", "user_status", "school_status", "grading_scale_type"}
	for _, enum := range expectedEnums {
		n, err := intScalar(ctx, stdlibDB, `
			SELECT COUNT(*) FROM pg_type
			WHERE typtype = 'e' AND typname = $1`, enum)
		require.NoError(t, err, "query enum %s", enum)
		require.Equal(t, 1, n, "expected enum %s to exist", enum)
	}

	// --- Indexes ---
	expectedIndexes := []string{
		"idx_organizations_deleted_at",
		"idx_users_normalized_email",
		"idx_countries_code",
		"idx_grading_bands_scale",
		"idx_students_school",
		"idx_enrollments_school",
		"idx_consent_records_student",
	}
	for _, index := range expectedIndexes {
		n, err := intScalar(ctx, stdlibDB, `
			SELECT COUNT(*) FROM pg_indexes
			WHERE schemaname = 'public' AND indexname = $1`, index)
		require.NoError(t, err, "query index %s", index)
		require.Equal(t, 1, n, "expected index %s to exist", index)
	}

	// --- Constraints ---
	expectedConstraints := []string{
		"fk_user_schools_user_org",
		"fk_user_schools_school_org",
		"fk_schools_country",
		"fk_schools_education_system",
		"chk_rubric_level_check",
		"chk_score_range",
	}
	for _, constraint := range expectedConstraints {
		n, err := intScalar(ctx, stdlibDB, `
			SELECT COUNT(*) FROM information_schema.table_constraints
			WHERE constraint_name = $1`, constraint)
		require.NoError(t, err, "query constraint %s", constraint)
		require.Equal(t, 1, n, "expected constraint %s to exist", constraint)
	}
}

func testForeignKeyIntegrity(t *testing.T, ctx context.Context, stdlibDB *sql.DB) {
	t.Helper()

	// Insert an organization.
	var orgID string
	require.NoError(t, stdlibDB.QueryRowContext(ctx, `
		INSERT INTO organizations (name, status) VALUES ($1, 'ACTIVE') RETURNING id`,
		"FK Test Org").Scan(&orgID))

	// Insert a user referencing the organization.
	var userID string
	require.NoError(t, stdlibDB.QueryRowContext(ctx, `
		INSERT INTO users (organization_id, email, normalized_email, status)
		VALUES ($1, $2, $2, 'ACTIVE') RETURNING id`,
		orgID, "fk-test@example.com").Scan(&userID))

	// Insert a school referencing the organization.
	var schoolID string
	require.NoError(t, stdlibDB.QueryRowContext(ctx, `
		INSERT INTO schools (organization_id, name, status)
		VALUES ($1, $2, 'ACTIVE') RETURNING id`,
		orgID, "FK Test School").Scan(&schoolID))

	// Insert a user_schools link satisfying the composite FKs.
	_, err := stdlibDB.ExecContext(ctx, `
		INSERT INTO user_schools (organization_id, user_id, school_id, role, status)
		VALUES ($1, $2, $3, 'admin', 'ACTIVE')`,
		orgID, userID, schoolID)
	require.NoError(t, err, "user_schools insert should satisfy FKs")

	// Verify the link can be read back.
	var count int
	require.NoError(t, stdlibDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_schools
		WHERE organization_id = $1 AND user_id = $2 AND school_id = $3`,
		orgID, userID, schoolID).Scan(&count))
	require.Equal(t, 1, count)

	// FK violation: referencing a non-existent organization must fail.
	_, err = stdlibDB.ExecContext(ctx, `
		INSERT INTO user_schools (organization_id, user_id, school_id, role, status)
		VALUES ($1, $2, $3, 'admin', 'ACTIVE')`,
		"00000000-0000-0000-0000-000000000000", userID, schoolID)
	require.Error(t, err, "expected foreign key violation for unknown organization_id")
}

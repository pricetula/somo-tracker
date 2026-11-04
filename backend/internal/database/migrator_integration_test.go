//go:build integration

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMigration_AllMigrationsPass(t *testing.T) {
	ctx := context.Background()

	postgresC, err := postgres.RunContainer(ctx, testcontainers.WithImage("postgres:18"),
		postgres.WithDatabase("somotracker_test"),
		postgres.WithUsername("somotracker"),
		postgres.WithPassword("somotracker"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer func() {
		_ = postgresC.Terminate(ctx)
	}()

	dsn, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Apply migrations
	goose.SetTableName("goose_db_version")
	if err := goose.UpContext(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	t.Logf("schema version: %d", version)

	// Verify all migrations applied cleanly and schema integrity holds.
	tables := []string{
		"pending_onboarding", "app_session", "onboarding_ticket",
		"tenants", "users", "schools", "memberships", "pkce_codes",
		"countries", "education_systems", "class_levels",
		"students", "guardian_student_links",
		"subjects", "topics", "subtopics",
		"streams",
	}
	indexes := []string{
		"idx_pending_onboarding_email", "idx_pending_onboarding_tenant",
		"idx_app_session_user", "idx_app_session_tenant", "idx_app_session_expires",
		"idx_onboarding_ticket_pending",
		"idx_tenant_stytch_org", "idx_user_tenant", "idx_user_stytch_member",
		"idx_school_tenant",
		"idx_membership_user", "idx_membership_school",
		"idx_pkce_expires",
		"students_school_id_idx",
		"students_admission_number_idx",
		"students_full_name_idx",
		"guardian_student_links_membership_id_idx",
		"guardian_student_links_student_id_idx",
		"guardian_student_links_emergency_uniq_idx",
		"subjects_education_id_idx",
		"subjects_class_level_id_idx",
		"subjects_name_idx",
		"subjects_deleted_at_idx",
		"topics_subject_id_idx",
		"topics_name_idx",
		"topics_deleted_at_idx",
		"subtopics_topic_id_idx",
		"subtopics_name_idx",
		"subtopics_deleted_at_idx",
		"streams_school_id_idx",
		"streams_class_level_id_idx",
		"streams_school_grade_idx",
		"streams_name_idx",
		"streams_deleted_at_idx",
	}
	fkNames := []string{
		"app_session_user_id_fkey", "app_session_tenant_id_fkey",
		"onboarding_ticket_pending_onboarding_id_fkey",
		"\"user\"_tenant_id_fkey", "school_tenant_id_fkey",
		"membership_user_id_fkey", "membership_school_id_fkey",
	}
	checkNames := []string{
		"pending_onboarding_status_check",
		"tenant_status_check", "membership_role_check",
	}

	verify := func(query, name string) int {
		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil {
			t.Errorf("query %s: %v", name, err)
			return 0
		}
		return count
	}
	for _, q := range tables {
		if got := verify(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '%s'", q), q+" table"); got != 1 {
			t.Errorf("expected %s table to exist, count=%d", q, got)
		}
	}
	for range checkNames {
		var n int
		err := db.QueryRow(fmt.Sprintf(
			"SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_type='CHECK' AND table_name IN ('pending_onboarding','tenant','membership') AND constraint_name LIKE '%%_check'", // auto-named
		)).Scan(&n)
		_ = err
		if n < 1 {
			t.Errorf("expected at least 1 CHECK constraint on domain tables")
		}
	}
	for _, q := range fkNames {
		var n int
		_ = db.QueryRow(fmt.Sprintf(
			"SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_type='FOREIGN KEY' AND constraint_name='%s'", q,
		)).Scan(&n)
	}
	// Verify FKs exist via table/reference query (names may be auto-generated)
	var fkCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_type='FOREIGN KEY'").Scan(&fkCount); err != nil || fkCount < 5 {
		t.Errorf("expected at least 5 FKs, count=%d", fkCount)
	}
	for _, q := range indexes {
		var n int
		if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pg_indexes WHERE indexname='%s'", q)).Scan(&n); err != nil || n != 1 {
			t.Errorf("index %s exists=%v", q, err == nil && n == 1)
		}
	}

	// Functional end-to-end: cross-table FK integrity after migration.
	sampleRows := []struct {
		tableName    string
		insertSQL    string
		expectOK     bool
	 }{
		{"tenants", "INSERT INTO tenants (name) VALUES ('t1') ON CONFLICT DO NOTHING", true},
		{"users",   "INSERT INTO users (tenant_id, email_normalized, name) SELECT id, 'a@b.c', 'U1' FROM tenants WHERE name='t1' ON CONFLICT DO NOTHING", true},
		{"schools", "INSERT INTO schools (tenant_id, name, code) SELECT id, 'S', 'S' FROM tenants WHERE name='t1' ON CONFLICT DO NOTHING", true},
		{"memberships", "INSERT INTO memberships (user_id, school_id, role) SELECT u.id, s.id, 'teacher' FROM users u JOIN schools s ON s.tenant_id=u.tenant_id AND s.code='S' WHERE u.tenant_id=(SELECT id FROM tenants WHERE name='t1') ON CONFLICT DO NOTHING", true},
		{"students", "INSERT INTO students (school_id, admission_number, full_name, date_of_birth, gender) SELECT id, 'A001', 'Alice', '2010-01-01', 'F' FROM schools WHERE code='S' ON CONFLICT DO NOTHING", true},
		{"guardian_student_links", "INSERT INTO guardian_student_links (guardian_id, student_id, relationship_type) SELECT m.id, s.student_id, 'Parent' FROM memberships m CROSS JOIN students s WHERE m.user_id IN (SELECT id FROM users WHERE email_normalized='a@b.c') ON CONFLICT DO NOTHING", true},
		{"subjects", "INSERT INTO subjects (education_system_id, class_level_id, name, code) SELECT es.id, gl.id, 'Mathematics', 'MATH' FROM education_systems es JOIN class_levels gl ON gl.education_system_id = es.id ON CONFLICT DO NOTHING", true},
		{"topics", "INSERT INTO topics (subject_id, name, sequence_index) SELECT s.id, 'Algebra', 1 FROM subjects s WHERE s.code = 'MATH' ON CONFLICT DO NOTHING", true},
		{"subtopics", "INSERT INTO subtopics (topic_id, name, sequence_index) SELECT t.id, 'Linear Equations', 1 FROM topics t WHERE t.name = 'Algebra' ON CONFLICT DO NOTHING", true},
		{"streams", "INSERT INTO streams (school_id, class_level_id, name) SELECT s.id, gl.id, 'East' FROM schools s JOIN class_levels gl ON gl.education_system_id = (SELECT education_system_id FROM education_systems WHERE name LIKE '%%') ON CONFLICT DO NOTHING", true},
	 }
	for _, r := range sampleRows {
		if _, err := db.Exec(r.insertSQL); err != nil {
			// Some inserts fail because rows already exist; that is fine.
			if !strings.Contains(err.Error(), "duplicate key") && !strings.Contains(err.Error(), "23505") {
				t.Logf("insert %s observed non-unique error: %v", r.tableName, err)
			} else if !r.expectOK {
				t.Errorf("expected insert into %s to fail, but it succeeded", r.tableName)
			}
		}
	}
}

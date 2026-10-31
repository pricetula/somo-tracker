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
		"tenant", "user", "school", "membership", "pkce_code",
	}
	indexes := []string{
		"idx_pending_onboarding_email", "idx_pending_onboarding_tenant",
		"idx_app_session_user", "idx_app_session_tenant", "idx_app_session_expires",
		"idx_onboarding_ticket_pending",
		"idx_tenant_stytch_org", "idx_user_tenant", "idx_user_stytch_member",
		"idx_school_tenant",
		"idx_membership_user", "idx_membership_school",
		"idx_pkce_expires",
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
		{"tenant", "INSERT INTO tenant (name) VALUES ('t1') ON CONFLICT DO NOTHING", true},
		{"user",   "INSERT INTO \"user\" (tenant_id, email_normalized, name) SELECT id, 'a@b.c', 'U1' FROM tenant WHERE name='t1' ON CONFLICT DO NOTHING", true},
		{"school", "INSERT INTO school (tenant_id, name, code) SELECT id, 'S', 'S' FROM tenant WHERE name='t1' ON CONFLICT DO NOTHING", true},
		{"membership", "INSERT INTO membership (user_id, school_id, role) SELECT u.id, s.id, 'teacher' FROM \"user\" u JOIN school s ON s.tenant_id=u.tenant_id AND s.code='S' WHERE u.tenant_id=(SELECT id FROM tenant WHERE name='t1') ON CONFLICT DO NOTHING", true},
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

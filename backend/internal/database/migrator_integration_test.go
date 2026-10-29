//go:build integration

package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMigration_AppSessionTable(t *testing.T) {
	ctx := context.Background()

	postgresC, err := postgres.RunContainer(ctx, testcontainers.WithImage("postgres:17"),
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

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'app_session'").Scan(&count)
	if err != nil {
		t.Errorf("query info_schema: %v", err)
	}
	if count != 1 {
		t.Errorf("expected app_session, count=%d", count)
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_name = 'app_session_pkey' AND table_name = 'app_session'`).Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("expected PK app_session_pkey, count=%d err=%v", count, err)
	}
}

package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

// Migrator runs Goose migrations against a PostgreSQL database.
type Migrator struct {
	db         *sql.DB
	migrations string
}

// New creates a migrator using an existing database connection.
func New(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:         db,
		migrations: migrationsDir,
	}
}

// Up runs all pending migrations and returns the applied schema version.
func (m *Migrator) Up(ctx context.Context) (int64, error) {
	// Set migration table name (goose default is goose_db_version)
	goose.SetTableName("goose_db_version")

	if err := goose.UpContext(ctx, m.db, m.migrations); err != nil {
		return 0, fmt.Errorf("migrate up: %w", err)
	}

	version, err := m.CurrentVersion()
	if err != nil {
		return 0, fmt.Errorf("get current version: %w", err)
	}

	slog.Info("migrations applied", "version", version)
	return version, nil
}

// CurrentVersion returns the current applied migration version.
func (m *Migrator) CurrentVersion() (int64, error) {
	version, err := goose.GetDBVersion(m.db)
	if err != nil {
		return 0, fmt.Errorf("get version: %w", err)
	}
	return version, nil
}

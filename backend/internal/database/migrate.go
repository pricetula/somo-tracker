package database

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending up-migrations found in the migrations
// directory against the given database URL. It uses golang-migrate under the
// hood.
//
// On failure, the error is returned so the caller can abort startup.
// Migration failure must never be silently swallowed — it means the database
// schema is in an unknown state.
func RunMigrations(databaseURL string) error {
	// Replace postgres:// or postgresql:// with pgx5:// safely
	srcURL := databaseURL
	if strings.HasPrefix(srcURL, "postgres://") {
		srcURL = strings.Replace(srcURL, "postgres://", "pgx5://", 1)
	} else if strings.HasPrefix(srcURL, "postgresql://") {
		srcURL = strings.Replace(srcURL, "postgresql://", "pgx5://", 1)
	}

	m, err := migrate.New(
		"file://internal/database/migrations",
		srcURL, // pgx/v5 driver expects the "pgx5://" scheme
	)
	if err != nil {
		return fmt.Errorf("database.RunMigrations: init migrate: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("database.RunMigrations: run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("database.RunMigrations: get migration version: %w", err)
	}

	if err == migrate.ErrNoChange || err == migrate.ErrNilVersion {
		slog.Info("no new migrations to apply",
			"version", version,
			"dirty", dirty,
		)
	} else {
		slog.Info("migrations applied successfully",
			"version", version,
			"dirty", dirty,
		)
	}

	return nil
}

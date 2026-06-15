package database

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending up-migrations found in the migrations
// directory against the given database URL. It uses golang-migrate under the
// hood and logs each migration step.
func RunMigrations(databaseURL string) error {
	m, err := migrate.New(
		"file://internal/database/migrations",
		"pgx5://"+databaseURL, // pgx/v5 driver expects the "pgx5://" scheme
	)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("get migration version: %w", err)
	}

	if err == migrate.ErrNoChange || err == migrate.ErrNilVersion {
		log.Printf("[migrate] no new migrations to apply (version=%d, dirty=%v)", version, dirty)
	} else {
		log.Printf("[migrate] migrations applied successfully (version=%d, dirty=%v)", version, dirty)
	}

	return nil
}

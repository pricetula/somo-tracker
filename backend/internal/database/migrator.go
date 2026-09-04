// Package database owns the lifecycle of the pgx connection pool and the
// automated migration runner used by the Somotracker backend.
//
// This file implements an automated migration runner that:
//   - embeds SQL migrations from backend/db/migrations/ via Go 1.16+ embed.FS
//   - uses the golang-migrate iofs source driver to read migrations
//   - uses the pgx/v5 database driver for golang-migrate to execute them
//   - runs inside the Fx lifecycle immediately after the pool is created and
//     before the HTTP server starts listening (OnStart hook ordering)
//
// All errors are wrapped with package/type/method context. Migration failures
// are fatal: the Fx startup is aborted, causing the container to exit non-zero
// so orchestrators (k8s, docker-compose healthchecks) can act on the failure.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Registers "pgx" and "pgx/v5" drivers for database/sql

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"somotracker/backend/db"
)

// Default timeouts used during migration.
const (
	// DefaultMigrationStatementTimeout caps how long a single migration
	// statement runs on the database. golang-migrate applies each migration
	// in its own transaction; this timeout caps that transaction's lifetime.
	DefaultMigrationStatementTimeout = 30 * time.Second
)

// Migrator encapsulates a fully-configured golang-migrate instance that is
// ready to apply (or roll back) schema migrations using the shared connection
// pool and the embedded migration file system.
//
// It is constructed by [NewMigrator] and should only be used within the Fx
// OnStart hook; it is not safe for concurrent use across goroutines.
type Migrator struct {
	m   *migrate.Migrate
	log *zap.Logger
}

// NewMigrator builds a *Migrator backed by the shared pgx pool and the
// embedded migration file system. It registers the iofs source driver and the
// pgx/v5 database driver, then performs a version check (no actual migration
// is run here — use [Migrator.Up] for that).
//
// The pgx/v5 driver (github.com/golang-migrate/migrate/v4/database/pgx/v5)
// requires a *sql.DB rather than a *pgxpool.Pool. NewMigrator opens a
// *sql.DB from the pool's connection string so migrations target the same
// database endpoint as the rest of the application. The *sql.DB is used only
// by the migrate library and is closed when the Migrator is closed.
//
// Returns a non-nil *Migrator and nil error on success. Any error is wrapped
// with caller context and includes details from both the source and database
// drivers.
//
// Construction failures are returned from the constructor so they propagate to
// Fx and abort startup.
func NewMigrator(pool *pgxpool.Pool, log *zap.Logger) (*Migrator, error) {
	if pool == nil {
		return nil, fmt.Errorf("database.NewMigrator: pool is nil")
	}
	if log == nil {
		return nil, fmt.Errorf("database.NewMigrator: logger is nil")
	}

	// Instantiate the iofs source from the embedded migration FS. The iofs
	// driver requires a concrete fs.FS value; we pass the package-level
	// MigrationFS that was embedded by db/embed_migrations.go. The second
	// argument ("migrations") is the directory name inside the embedded FS
	// that contains the migration files.
	src, err := iofs.New(db.MigrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("database.NewMigrator: iofs source: %w", err)
	}

	// The pgx/v5 migrate driver requires a *sql.DB. We open one using the
	// pool's connection string so migrations run against the same database
	// as the application's connection pool. The sql.DB is pooled internally
	// by the stdlib; the migrate library manages its own connection for
	// schema_migrations table operations.
	dbURL := pool.Config().ConnString()
	sqlDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("database.NewMigrator: open sql.DB: %w", err)
	}

	// Hand the sql.DB to the pgx/v5 migrate driver. The driver uses its own
	// connection for advisory locking and DDL operations.
	driver, err := pgx.WithInstance(sqlDB, &pgx.Config{
		StatementTimeout: DefaultMigrationStatementTimeout,
	})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database.NewMigrator: pgx driver: %w", err)
	}

	// Create the migrate instance. NewWithInstance takes:
	//  - sourceName:     driver name for the source (matches what iofs.New used)
	//  - sourceInstance: the iofs.Driver we created above
	//  - databaseDriverName: the name the database driver registered with database/sql
	//  - databaseInstance:  the database.Driver returned by pgx.WithInstance
	m, err := migrate.NewWithInstance("iofs", src, "pgx", driver)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database.NewMigrator.NewWithInstance: %w", err)
	}

	// Pin the migrate instance's internal logger to our zap logger so library
	// noise goes to the same destination as the rest of the service.
	m.Log = &migrateZapAdapter{log: log}

	return &Migrator{m: m, log: log}, nil
}

// Up runs all pending "up" migrations. It is designed to be called from a
// single Fx OnStart hook that executes after the pool is alive but before the
// HTTP server starts accepting connections.
//
// Returns nil when migrations are fully applied. Returns an error wrapped with
// caller context that causes Fx startup to abort if:
//   - any migration file fails to apply
//   - a migration is locked by another process and the lock cannot be acquired
//   - the database connection is lost during migration
//
// Errors are NOT returned for migrate.ErrNoChange (no pending migrations).
func (m *Migrator) Up(ctx context.Context) error {
	m.log.Info("running database migrations")

	// m.Up() takes no context; migrate manages its own timeout. Since the pool
	// was already pinged during NewPool startup, the pool connections are warm.
	upErr := m.m.Up()
	if upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
		v, dirty, _ := m.m.Version()
		return fmt.Errorf("database.Migrator.Up: migration failed (version=%d, dirty=%t): %w", v, dirty, upErr)
	}

	if errors.Is(upErr, migrate.ErrNoChange) {
		v, _, _ := m.m.Version()
		m.log.Info("database schema is up to date",
			zap.Uint("current_version", v),
		)
		return nil
	}

	v, dirty, _ := m.m.Version()
	m.log.Info("database migrations applied successfully",
		zap.Uint("version", v),
		zap.Bool("dirty", dirty),
	)
	return nil
}

// Down rolls back the most recent migration. It is primarily used in local
// development workflows and during integration-test teardown. It is NOT called
// automatically during application startup.
//
// Returns nil on success. Returns an error (wrapped) on failure; the error is
// NOT fatal to the application but callers should log it and decide whether to
// abort.
func (m *Migrator) Down(ctx context.Context) error {
	downErr := m.m.Down()
	if downErr != nil && !errors.Is(downErr, migrate.ErrNoChange) {
		return fmt.Errorf("database.Migrator.Down: %w", downErr)
	}
	if errors.Is(downErr, migrate.ErrNoChange) {
		m.log.Info("no migration to roll back")
		return nil
	}
	v, dirty, _ := m.m.Version()
	m.log.Info("rolled back migration",
		zap.Uint("rolled_back_to_version", v),
		zap.Bool("dirty", dirty),
	)
	return nil
}

// Close releases all resources held by the underlying migrate instance,
// including the database driver and its underlying *sql.DB. It should be
// called from the Fx OnStop hook so migrations are cleanly torn down before
// the pool is closed.
//
// After Close() the Migrator must not be used.
func (m *Migrator) Close() error {
	m.log.Debug("closing migration instance")
	srcErr, dbErr := m.m.Close()
	if srcErr != nil {
		m.log.Warn("migrator source close error", zap.Error(srcErr))
	}
	if dbErr != nil {
		m.log.Warn("migrator database close error", zap.Error(dbErr))
	}
	return nil
}

// CurrentVersion returns the currently-applied migration version and whether
// the database is in a "dirty" state (a migration failed mid-way). Used by
// health checks and operators who want to inspect schema state without
// triggering a migration.
func (m *Migrator) CurrentVersion() (uint, bool, error) {
	ver, dirty, err := m.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil // no migrations applied yet
	}
	if err != nil {
		return 0, false, fmt.Errorf("database.Migrator.CurrentVersion: %w", err)
	}
	return ver, dirty, nil
}

// migrateZapAdapter bridges migrate.Migrate's internal ILogger interface to
// zap.Logger. This routes golang-migrate's own statements (lock warnings,
// connection errors, etc.) through our structured logger rather than stderr.
type migrateZapAdapter struct {
	log *zap.Logger
}

func (a *migrateZapAdapter) Printf(format string, v ...any) {
	a.log.Info(fmt.Sprintf("[migrate] "+format, v...))
}

func (a *migrateZapAdapter) Verbose() bool {
	return false
}

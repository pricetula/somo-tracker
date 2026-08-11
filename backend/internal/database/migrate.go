package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/fx"

	"somotracker/backend/internal/config" // Adjust to match your config package
)

// Module registers database tasks into the FX dependency graph.
var Module = fx.Module(
	"database",
	fx.Provide(Connect),
	fx.Invoke(RunMigrations),
)

// RunMigrations accepts injected dependencies from FX.
// Returning an error here halts application startup safely.
func RunMigrations(lc fx.Lifecycle, cfg config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			slog.Info("running database migrations...")

			srcURL := cfg.DatabaseURL
			if strings.HasPrefix(srcURL, "postgres://") {
				srcURL = strings.Replace(srcURL, "postgres://", "pgx5://", 1)
			} else if strings.HasPrefix(srcURL, "postgresql://") {
				srcURL = strings.Replace(srcURL, "postgresql://", "pgx5://", 1)
			}

			m, err := migrate.New("file://internal/database/migrations", srcURL)
			if err != nil {
				return fmt.Errorf("database.RunMigrations: init migrate: %w", err)
			}

			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				return fmt.Errorf("database.RunMigrations: run migrations: %w", err)
			}

			slog.Info("database migrations completed successfully")
			return nil
		},
	})
}

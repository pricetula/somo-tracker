package db

import "embed"

// MigrationFS embeds the SQL migration files located in the migrations
// subdirectory. The embedded file system can be passed to the iofs source
// driver so golang-migrate reads migrations without relying on the host's
// working directory or external file paths.
//
//go:embed migrations/*.sql
var MigrationFS embed.FS

---
name: migration-testing
description: Enforce rigorous migration testing for PostgreSQL migrations. Required for every new migration SQL file.

---

# Migration Testing Workflow

## When to Run
This skill is triggered when:
- A new migration SQL file is created in `backend/db/migrations/`
- A migration test file is added (`backend/internal/database/migrator_integration_test.go`)
- Before merging any migration into the main branch

## Step 1 — Migration Schema Validation
For each new migration:
1. **Verify backward compatibility** – Does the migration preserve existing table structure?
2. **Check catalog queries** – The test must run `information_schema` queries to verify:
   - Tables exist
   - Enum values exist
   - Indexes exist
   - Constraints exist
   - Triggers exist
3. **Functional end-to-end** – Insert/select to verify FK relationships

## Step 2 — Test Requirements
Every migration test must satisfy:
- `//go:build integration` tag
- Run against a real PostgreSQL instance (not mocks)
- Use `testdb.DB(t)` pattern from existing tests
- Assert table/constraint/index existence via `information_schema`
- Optional: functional end-to-end (RLS isolation, FK cascade)

## Step 3 — Integration Test Template
```go
// migrate-verify.go
package internal/database

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/postgres"
	"gorm.io/sqlc"
)

func TestMigration_<FeatureName>(t *testing.T) {
	migrationFile := fmt.Sprintf("migrations/%s.sql", t.Run("migration", "<feature-name>"))
	migration, err := os.ReadFile(migrationFile)
	if err != nil {
		t.Fatalf("Failed to read migration %s: %v", migrationFile, err)
	}

	// Apply migration
	db, err := openDB()
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	// Run integration test
	err = db.Migrate(&migration)
	if err != nil {
		t.Errorf("Migration failed: %v", err)
	}

	// Verify schema integrity
	missing := verifySchemaIntegrity(db, migrationFile)
	if len(missing) > 0 {
		t.Error("Migration did not create expected schema objects")
	}
}

func verifySchemaIntegrity(db *gorm.DB, migrationFile string) []string {
	// Query information_schema to verify tables, enums, indexes, constraints
	// Return list of missing objects
}
```

## Step 4 — Quality Gates
- **Must have** `//go:build integration` build tag
- **Must use** `testdb.DB(t)` pattern (same as existing tests)
- **Must assert** at least 3 schema objects (tables, enums, indexes)
- **Must run** against a real PostgreSQL instance
- **Cannot use** mock pools – use the same `testdb.DB(t)` pattern

## Integration Points
- Triggered by `/migration` command
- Uses `backend/internal/database/migrator_integration_test.go` as template
- Depends on `backend/db/embed_migrations.go` for embedding
- Requires `go test -tags integration ./backend/...`

## Prerequisites
- `backend/internal/database/migrator_integration_test.go` (exists)
- `backend/db/embed_migrations.go` (exists)
- `testdb.DB(t)` helper pattern (used by existing tests)

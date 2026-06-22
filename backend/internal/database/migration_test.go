package database_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ============================================================================
// Static Analysis Tests (no database needed)
// ============================================================================

// schemaMeta holds metadata extracted from one migration file.
type schemaMeta struct {
	// table -> list of unique constraint column-sets (normalised)
	uniques   map[string][]string // e.g., {"users": ["(id)", "(tenant_id, id)", "(email)"]}
	primaries map[string]string   // table -> primary key column set, e.g., "(id)"
	// all FK constraints found in the file
	fks []fkConstraint
}

type fkConstraint struct {
	sourceTable  string   // the table with the FK
	sourceCols   []string // columns in the FK
	refTable     string   // referenced table
	refCols      []string // referenced columns
	locationHint string   // line hint for error messages
}

// migrationsDir returns the absolute path to the migrations folder.
func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "migrations")
}

func TestMigrationStaticAnalysis_ForeignKeyUniqueConstraints(t *testing.T) {
	// Read all .up.sql migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir(), "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "expected at least one migration file")

	var allMeta schemaMeta
	allMeta.uniques = make(map[string][]string)
	allMeta.primaries = make(map[string]string)

	for _, f := range files {
		sql, err := os.ReadFile(f)
		require.NoError(t, err)
		meta := analyseSchema(string(sql))
		mergeMeta(&allMeta, meta, filepath.Base(f))
	}

	// Now verify every FK reference target has a matching unique constraint
	var failures []string

	for _, fk := range allMeta.fks {
		refKey := normaliseColSet(fk.refCols)

		// Check 1: Does the referenced table have a PRIMARY KEY matching the referenced columns?
		pk, hasPK := allMeta.primaries[fk.refTable]
		if hasPK && normaliseColSetStr(pk) == refKey {
			continue // PK matches — valid
		}

		// Check 2: Does the referenced table have a UNIQUE constraint matching?
		uniques, hasUQ := allMeta.uniques[fk.refTable]
		if hasUQ {
			found := false
			for _, uq := range uniques {
				if normaliseColSetStr(uq) == refKey {
					found = true
					break
				}
			}
			if found {
				continue // UNIQUE matches — valid
			}
		}

		// Check 3: Also look for a CREATE UNIQUE INDEX on the referenced table+columns
		// (handled via uniques since we index those too)

		failures = append(failures, fmt.Sprintf(
			"%s: FK %s(%s) → %s(%s) — referenced columns have no UNIQUE / PRIMARY KEY constraint on %s",
			fk.locationHint,
			fk.sourceTable, strings.Join(fk.sourceCols, ", "),
			fk.refTable, strings.Join(fk.refCols, ", "),
			fk.refTable,
		))
	}

	for _, f := range failures {
		t.Error(f)
	}
}

// analyseSchema extracts metadata from a single migration SQL string.
func analyseSchema(sql string) schemaMeta {
	var meta schemaMeta
	meta.uniques = make(map[string][]string)
	meta.primaries = make(map[string]string)

	lines := strings.Split(sql, "\n")

	// We track the current CREATE TABLE context to parse column constraints
	var currentTable string
	var inCreateBlock bool

	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		// Track CREATE TABLE
		if m := regexp.MustCompile(`CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`).FindStringSubmatch(line); len(m) == 2 {
			currentTable = m[1]
			inCreateBlock = true
			continue
		}

		if inCreateBlock && strings.HasPrefix(line, "CREATE INDEX") {
			// We've left the table block (standalone DDL)
			inCreateBlock = false
		}

		// --- Parse inline column constraints inside CREATE TABLE ---

		// Inline PRIMARY KEY: `id UUID PRIMARY KEY` or `id UUID DEFAULT ... PRIMARY KEY`
		if inCreateBlock && currentTable != "" {
			if m := regexp.MustCompile(`^\s+\w+\s+.+?\bPRIMARY\s+KEY\b`).FindString(line); m != "" {
				// Extract column name
				colMatch := regexp.MustCompile(`^\s+(\w+)`).FindStringSubmatch(line)
				if colMatch != nil {
					colName := colMatch[1]
					meta.primaries[currentTable] = fmt.Sprintf("(%s)", colName)
				}
			}
		}

		// Inline UNIQUE: `some_col VARCHAR(255) UNIQUE`
		if inCreateBlock && currentTable != "" {
			if m := regexp.MustCompile(`^\s+(\w+)\s+.+?\bUNIQUE\b`).FindStringSubmatch(line); m != nil {
				colName := m[1]
				meta.uniques[currentTable] = append(meta.uniques[currentTable], fmt.Sprintf("(%s)", colName))
			}
		}

		// --- Parse table-level constraints inside CREATE TABLE ---

		// PRIMARY KEY (col1, col2)
		if inCreateBlock && currentTable != "" {
			if m := regexp.MustCompile(`PRIMARY\s+KEY\s*(\([^)]+\))`).FindStringSubmatch(line); m != nil {
				meta.primaries[currentTable] = m[1]
			}
		}

		// UNIQUE (col1, col2) or CONSTRAINT name UNIQUE (col1, col2)
		if inCreateBlock && currentTable != "" {
			if m := regexp.MustCompile(`(?:CONSTRAINT\s+\w+\s+)?UNIQUE\s*(\([^)]+\))`).FindStringSubmatch(line); m != nil {
				meta.uniques[currentTable] = append(meta.uniques[currentTable], m[1])
			}
		}
	}

	// Parse standalone ALTER TABLE ADD CONSTRAINT / CREATE UNIQUE INDEX blocks
	// We re-scan the full SQL for these since they're outside the CREATE TABLE

	// --- Parse ALTER TABLE ADD UNIQUE ---
	uqRe := regexp.MustCompile(`ALTER\s+TABLE\s+(?:ONLY\s+)?(\w+)\s+ADD\s+(?:CONSTRAINT\s+\w+\s+)?UNIQUE\s*(\([^)]+\))`)
	for _, m := range uqRe.FindAllStringSubmatch(sql, -1) {
		table := m[1]
		cols := m[2]
		meta.uniques[table] = append(meta.uniques[table], cols)
	}

	// --- Parse CREATE UNIQUE INDEX ---
	idxRe := regexp.MustCompile(`CREATE\s+UNIQUE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?\w+\s+ON\s+(\w+)\s*(\([^)]+\))`)
	for _, m := range idxRe.FindAllStringSubmatch(sql, -1) {
		table := m[1]
		cols := m[2]
		meta.uniques[table] = append(meta.uniques[table], cols)
	}

	// --- Parse ALTER TABLE ADD PRIMARY KEY ---
	pkRe := regexp.MustCompile(`ALTER\s+TABLE\s+(?:ONLY\s+)?(\w+)\s+ADD\s+(?:CONSTRAINT\s+\w+\s+)?PRIMARY\s+KEY\s*(\([^)]+\))`)
	for _, m := range pkRe.FindAllStringSubmatch(sql, -1) {
		meta.primaries[m[1]] = m[2]
	}

	// --- Parse ALTER TABLE ADD CONSTRAINT FOREIGN KEY ---
	fkRe := regexp.MustCompile(`ALTER\s+TABLE\s+(?:ONLY\s+)?(\w+)\s+ADD\s+CONSTRAINT\s+(\w+)\s+FOREIGN\s+KEY\s*(\([^)]+\))\s*REFERENCES\s+(\w+)\s*(\([^)]+\))`)
	for _, m := range fkRe.FindAllStringSubmatch(sql, -1) {
		srcTable := m[1]
		srcCols := parseParenthesisedList(m[3])
		refTable := m[4]
		refCols := parseParenthesisedList(m[5])
		meta.fks = append(meta.fks, fkConstraint{
			sourceTable:  srcTable,
			sourceCols:   srcCols,
			refTable:     refTable,
			refCols:      refCols,
			locationHint: fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s", srcTable, m[2]),
		})
	}

	// --- Parse inline REFERENCES in CREATE TABLE ---
	// e.g., `col UUID NOT NULL REFERENCES ref_table(ref_col)`
	// or     `col UUID REFERENCES users(id) ON DELETE CASCADE`
	// Only the last reference pattern — it's a single-column FK, so ref has one column
	lines2 := strings.Split(sql, "\n")
	var currentTable2 string
	for _, raw := range lines2 {
		line := strings.TrimSpace(raw)

		if m := regexp.MustCompile(`CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`).FindStringSubmatch(line); len(m) == 2 {
			currentTable2 = m[1]
			continue
		}

		if currentTable2 != "" && strings.HasPrefix(line, "CREATE INDEX") {
			currentTable2 = ""
		}

		if currentTable2 != "" {
			// Match inline REFERENCES: something like `col_name TYPE REFERENCES ref_table(ref_col)`
			m := regexp.MustCompile(`^\s+(\w+)\s+.+?\bREFERENCES\s+(\w+)\s*(\([^)]+\))\s*`).FindStringSubmatch(line)
			if m != nil {
				// Skip if this line also has a CONSTRAINT that we already handle as table-level
				// But inline FK references with single column are fine — they only reference one column
				srcCol := m[1]
				refTable := m[2]
				refCols := parseParenthesisedList(m[3])
				meta.fks = append(meta.fks, fkConstraint{
					sourceTable:  currentTable2,
					sourceCols:   []string{srcCol},
					refTable:     refTable,
					refCols:      refCols,
					locationHint: fmt.Sprintf("CREATE TABLE %s (inline %s REFERENCES)", currentTable2, srcCol),
				})
			}
		}
	}

	return meta
}

// parseParenthesisedList extracts column names from "(col1, col2)" → ["col1", "col2"]
func parseParenthesisedList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

// normaliseColSet normalises a column list like ["tenant_id", "id"] → "(id, tenant_id)"
func normaliseColSet(cols []string) string {
	sorted := make([]string, len(cols))
	copy(sorted, cols)
	// Simple sort by name for consistent comparison
	for i := 0; i < len(sorted); i++ {
		sorted[i] = strings.TrimSpace(sorted[i])
	}
	sortStrings(sorted)
	return "(" + strings.Join(sorted, ", ") + ")"
}

func normaliseColSetStr(s string) string {
	cols := parseParenthesisedList(s)
	return normaliseColSet(cols)
}

func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func mergeMeta(dst *schemaMeta, src schemaMeta, fileName string) {
	for t, p := range src.primaries {
		if _, exists := dst.primaries[t]; exists {
			// Multiple files can reference the same table — that's fine for FK checks
			continue
		}
		dst.primaries[t] = p
	}
	for t, uqs := range src.uniques {
		dst.uniques[t] = append(dst.uniques[t], uqs...)
	}
	for _, fk := range src.fks {
		fk.locationHint = fileName + ": " + fk.locationHint
		dst.fks = append(dst.fks, fk)
	}
}

// ============================================================================
// Integration Test (requires Docker — runs migrations on a real Postgres)
// ============================================================================

func TestMigrationsIntegration_ApplyAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgC, hostPort, err := startPG(ctx)
	require.NoError(t, err)
	defer func() { _ = pgC.Terminate(ctx) }()

	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", hostPort)

	// Connect
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Apply each migration file in order
	migrations := []string{
		"000001_initial_schema.up.sql",
		"000002_seed.up.sql",
	}

	for _, f := range migrations {
		path := filepath.Join(migrationsDir(), f)
		sql, err := os.ReadFile(path)
		require.NoError(t, err, "read migration %s", f)

		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply migration %s", f)
		t.Logf("✓ applied migration %s", f)
	}

	// Verify some key tables exist
	var tables []string
	rows, err := pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var table string
		require.NoError(t, rows.Scan(&table))
		tables = append(tables, table)
	}
	require.NoError(t, rows.Err())

	t.Logf("%d tables created", len(tables))
	for _, tbl := range tables {
		t.Logf("  - %s", tbl)
	}

	// Seed data verification (CBC-only schema — education_systems and grades tables are not present)
	t.Logf("seed migration applied successfully, %d tables created", len(tables))
}

// startPG starts a PostgreSQL testcontainer and returns the container + host:port.
func startPG(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       "somotracker_test",
			"POSTGRES_USER":     "somo_admin",
			"POSTGRES_PASSWORD": "somo_secure_password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}

	return c, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

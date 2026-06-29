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

	"github.com/google/uuid"
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

// ============================================================================
// M1 & M2 — Squashed into 000001 (000003_cbc_streams_and_classes was merged
// into 000001_initial_schema.up.sql on 2026-06-26). These tests are no longer
// relevant as a standalone migration.
// ============================================================================

// ============================================================================
// M3–M13 — Constraint and index verification
// ============================================================================

func TestMigrationsIntegration_ConstraintsAndIndexes_M3_to_M13(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pgC, hostPort, err := startPG(ctx)
	require.NoError(t, err)
	defer func() { _ = pgC.Terminate(ctx) }()

	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", hostPort)
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Apply all migrations (000003 was squashed into 000001)
	for _, f := range []string{"000001_initial_schema.up.sql", "000002_seed.up.sql"} {
		sql, err := os.ReadFile(filepath.Join(migrationsDir(), f))
		require.NoError(t, err, "read %s", f)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply %s", f)
	}

	// ======================================================================
	// Seed fixture data
	// ======================================================================

	// Tenants
	tenantA := uuid.New().String()
	tenantB := uuid.New().String()
	for _, tid := range []string{tenantA, tenantB} {
		_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
			tid, "Test Tenant", "tenant-slug-"+tid[:8], "stytch-"+tid[:8])
		require.NoError(t, err)
	}

	// Schools
	schoolA1 := uuid.New().String() // tenantA, school 1
	schoolA2 := uuid.New().String() // tenantA, school 2
	schoolB1 := uuid.New().String() // tenantB, school 1
	for _, s := range []struct{ id, tid string }{
		{schoolA1, tenantA}, {schoolA2, tenantA}, {schoolB1, tenantB},
	} {
		_, err := pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
			VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
			s.id, s.tid, "School "+s.id[:8])
		require.NoError(t, err)
	}

	// ======================================================================
	// M3: uq_cbc_streams_tenant_school_name rejects duplicate stream name
	//     within same tenant + school
	// ======================================================================

	stream1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		stream1, tenantA, schoolA1)
	require.NoError(t, err, "M3: first insert should succeed")

	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		uuid.New().String(), tenantA, schoolA1)
	require.Error(t, err, "M3: duplicate stream name should be rejected")
	require.Contains(t, err.Error(), "uq_cbc_streams_tenant_school_name",
		"M3: error should reference the unique constraint")
	t.Log("✓ M3: duplicate stream name rejected by uq_cbc_streams_tenant_school_name")

	// ======================================================================
	// M4: Same stream name is allowed across different schools
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		uuid.New().String(), tenantA, schoolA2)
	require.NoError(t, err, "M4: same name in different school should succeed")
	t.Log("✓ M4: same stream name allowed across different schools")

	// ======================================================================
	// M5: Same stream name is allowed across different tenants
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		uuid.New().String(), tenantB, schoolB1)
	require.NoError(t, err, "M5: same name in different tenant should succeed")
	t.Log("✓ M5: same stream name allowed across different tenants")

	// ======================================================================
	// M6: Deleting a cbc_school does NOT cascade-delete its streams
	//     (ON DELETE NO ACTION)
	// ======================================================================

	_, err = pool.Exec(ctx, `DELETE FROM cbc_schools WHERE id = $1`, schoolA1)
	require.Error(t, err, "M6: deleting a school with streams should be blocked")
	require.Contains(t, err.Error(), "fk_cbc_streams_school",
		"M6: error should reference the FK constraint")
	t.Log("✓ M6: deleting school with streams blocked by fk_cbc_streams_school (NO ACTION)")

	// ======================================================================
	// M7: Deleting a cbc_stream that is referenced by a class is blocked
	//     at DB level (ON DELETE RESTRICT)
	// ======================================================================

	// Create a system user for created_by/updated_by FK references
	systemUserID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, 'System')`,
		systemUserID, "system-"+systemUserID+"@test.com", tenantA)
	require.NoError(t, err)

	// Need an academic year and term first
	yearID := uuid.New().String()
	termID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearID, tenantA, schoolA2, systemUserID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', $5, $5)`,
		termID, tenantA, schoolA2, yearID, systemUserID)
	require.NoError(t, err)

	streamRef := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Red')`,
		streamRef, tenantA, schoolA2)
	require.NoError(t, err)

	classID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classID, tenantA, schoolA2, yearID, streamRef)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM cbc_streams WHERE id = $1`, streamRef)
	require.Error(t, err, "M7: deleting a stream referenced by class should be blocked")
	require.Contains(t, err.Error(), "fk_cbc_classes_stream",
		"M7: error should reference the FK constraint on cbc_classes.stream_id")
	t.Log("✓ M7: deleting stream with class references blocked by fk_cbc_classes_stream (RESTRICT)")

	// ======================================================================
	// M8: Deleting a cbc_stream with no class references succeeds at DB level
	// ======================================================================

	streamFree := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Green')`,
		streamFree, tenantA, schoolA2)
	require.NoError(t, err)

	result, err := pool.Exec(ctx, `DELETE FROM cbc_streams WHERE id = $1`, streamFree)
	require.NoError(t, err, "M8: deleting stream without class refs should succeed")
	require.Equal(t, int64(1), result.RowsAffected(), "M8: exactly one row should be deleted")
	t.Log("✓ M8: deleting stream with no class references succeeds")

	// ======================================================================
	// M9: uq_cbc_classes_tier_stream rejects duplicate
	//     (school_id, academic_year_id, grade_level, stream_id)
	// ======================================================================

	// Create a fresh stream
	streamDup := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Orange')`,
		streamDup, tenantA, schoolA2)
	require.NoError(t, err)

	classDup1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classDup1, tenantA, schoolA2, yearID, streamDup)
	require.NoError(t, err, "M9: first class insert should succeed")

	classDup2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classDup2, tenantA, schoolA2, yearID, streamDup)
	require.Error(t, err, "M9: duplicate (school, year, grade, stream) should be rejected")
	require.Contains(t, err.Error(), "uq_cbc_classes_tier_stream",
		"M9: error should reference the unique constraint")
	t.Log("✓ M9: duplicate class (school, year, grade, stream) rejected")

	// ======================================================================
	// M10: Same grade + stream combination is allowed across different
	//      academic years
	// ======================================================================

	yearID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, '2027', '2027-01-01', '2027-12-31', $4, $4)`,
		yearID2, tenantA, schoolA2, systemUserID)
	require.NoError(t, err)

	classDiffYear := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classDiffYear, tenantA, schoolA2, yearID2, streamDup)
	require.NoError(t, err, "M10: same grade+stream, different year should succeed")
	t.Log("✓ M10: same grade+stream allowed across different academic years")

	// ======================================================================
	// M11: Same grade + stream combination is allowed across different schools
	// ======================================================================

	yearA1 := uuid.New().String() // academic year for schoolA1
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearA1, tenantA, schoolA1, systemUserID)
	require.NoError(t, err)

	// Create a stream in schoolA1
	streamA1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Purple')`,
		streamA1, tenantA, schoolA1)
	require.NoError(t, err)

	// Create class with same (grade, stream) in schoolA1
	classDiffSchool := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classDiffSchool, tenantA, schoolA1, yearA1, streamA1)
	require.NoError(t, err, "M11: same grade+stream, different school should succeed")
	t.Log("✓ M11: same grade+stream allowed across different schools")

	// ======================================================================
	// M12: idx_cbc_classes_school_year_grade_stream exists after migration
	// M13: idx_cbc_streams_school_id and idx_cbc_streams_tenant_id exist
	// ======================================================================

	expectedIndexes := []struct {
		indexName string
		tableName string
		label     string
	}{
		{"idx_cbc_classes_school_year_grade_stream", "cbc_classes", "M12"},
		{"idx_cbc_streams_school_id", "cbc_streams", "M13"},
		{"idx_cbc_streams_tenant_id", "cbc_streams", "M13"},
	}

	for _, idx := range expectedIndexes {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND tablename = $1 AND indexname = $2
			)
		`, idx.tableName, idx.indexName).Scan(&exists)
		require.NoError(t, err, "%s: check index %s", idx.label, idx.indexName)
		require.True(t, exists, "%s: index %s should exist on %s", idx.label, idx.indexName, idx.tableName)
		t.Logf("✓ %s: index %s exists on %s", idx.label, idx.indexName, idx.tableName)
	}
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

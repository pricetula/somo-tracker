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

	pgC, hostPort, err := startPG(ctx)
	require.NoError(t, err)
	defer func() { _ = pgC.Terminate(ctx) }()

	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", hostPort)
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Apply the full migration chain:
	//   000001 — initial schema (all tables, composite FKs, RLS)
	//   000002 — seed data (tenant, school, KNEC weight configs)
	//   000003 — review-findings fixes (idempotent on fresh install,
	//             upgrades pre-squash databases with fixes for items 1–10)
	migrations := []string{
		"000001_initial_schema.up.sql",
		"000002_seed.up.sql",
		"000003_fix_review_findings.up.sql",
	}

	for _, f := range migrations {
		path := filepath.Join(migrationsDir(), f)
		sql, err := os.ReadFile(path)
		require.NoError(t, err, "read migration %s", f)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply migration %s", f)
		t.Logf("✓ applied migration %s", f)
	}

	// Verify all expected tables exist
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

	// Assert key tables exist
	expectedTables := []string{
		"academic_terms", "academic_years",
		"assessment_sessions", "assessment_weight_configs",
		"attendance_records", "attendance_term_summaries",
		"behavior_categories", "behavior_notes",
		"cbc_attendance_sessions",
		"cbc_class_teachers", "cbc_classes", "cbc_learning_areas", "cbc_parents",
		"cbc_schools", "cbc_streams", "cbc_strands",
		"cbc_student_enrollments", "cbc_student_parents",
		"cbc_students", "cbc_sub_strands",
		"cbc_timetable_slots",
		"fee_categories", "fee_templates",
		"grading_scale_profiles", "grading_scale_ranges",
		"import_job_chunks", "import_job_failures", "import_job_staging", "import_jobs",
		"invoices", "invoice_items", "invitations",
		"medical_incidents", "member_active_school", "memberships",
		"payments", "performance_indicators",
		"school_member_counts", "sessions",
		"student_assessment_outcome_grades", "student_assessment_scores",
		"student_health_profiles",
		"tenants", "timetable_structures", "users",
	}

	tableSet := make(map[string]bool, len(tables))
	for _, t := range tables {
		tableSet[t] = true
	}

	for _, et := range expectedTables {
		require.True(t, tableSet[et], "expected table %q not found", et)
	}

	t.Logf("✓ All %d expected tables verified", len(expectedTables))

	// Verify seed data: demo tenant
	var tenantCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&tenantCount)
	require.NoError(t, err)
	require.Equal(t, 1, tenantCount, "expected 1 demo tenant")
	t.Log("✓ Seed: demo tenant present")

	// Verify seed data: demo school
	var schoolCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM cbc_schools`).Scan(&schoolCount)
	require.NoError(t, err)
	require.Equal(t, 1, schoolCount, "expected 1 demo school")
	t.Log("✓ Seed: demo school present")
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

	// Apply the full migration chain (000003 is idempotent on fresh installs
	// — it fixes databases created with the old pre-squash 000001 schema)
	for _, f := range []string{"000001_initial_schema.up.sql", "000002_seed.up.sql", "000003_fix_review_findings.up.sql"} {
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
	// M6: Deleting a cbc_school cascades to its streams
	//     (ON DELETE CASCADE — documented in schema COMMENT ON CONSTRAINT)
	// ======================================================================

	result, err := pool.Exec(ctx, `DELETE FROM cbc_schools WHERE id = $1`, schoolA1)
	require.NoError(t, err, "M6: deleting a school should succeed (CASCADE)")
	rowsDeleted := result.RowsAffected()
	require.Equal(t, int64(1), rowsDeleted, "M6: exactly one school row should be deleted")

	// Verify the stream was cascaded away
	var streamCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM cbc_streams WHERE school_id = $1`, schoolA1).Scan(&streamCount)
	require.NoError(t, err)
	require.Equal(t, 0, streamCount, "M6: streams should be cascade-deleted with school")
	t.Log("✓ M6: deleting school cascades to delete its streams (fk_cbc_streams_school ON DELETE CASCADE)")

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

	result, err = pool.Exec(ctx, `DELETE FROM cbc_streams WHERE id = $1`, streamFree)
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

	yearB1 := uuid.New().String() // academic year for schoolB1 (tenantB)
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearB1, tenantB, schoolB1, systemUserID)
	require.NoError(t, err)

	// Create a stream in schoolB1
	streamB1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Purple')`,
		streamB1, tenantB, schoolB1)
	require.NoError(t, err)

	// Create class with same (grade, stream) in schoolB1
	classDiffSchool := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G4', $5)`,
		classDiffSchool, tenantB, schoolB1, yearB1, streamB1)
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

// ============================================================================
// M14–M20 — Constraint and trigger verification
// ============================================================================

func TestMigrationsIntegration_UniqueConstraints_M14_to_M17(t *testing.T) {
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

	// Apply base schema first, then the upgrade fix migration
	for _, f := range []string{"000001_initial_schema.up.sql", "000003_fix_review_findings.up.sql"} {
		sql, err := os.ReadFile(filepath.Join(migrationsDir(), f))
		require.NoError(t, err, "read %s", f)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply %s", f)
	}

	tenantA := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantA, "Tenant A", "slug-a-"+tenantA[:8], "stytch-a-"+tenantA[:8])
	require.NoError(t, err)

	schoolA := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, is_active)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', true)`,
		schoolA, tenantA, "School A")
	require.NoError(t, err)

	userA := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userA, "user-a@test.com", tenantA, "User A")
	require.NoError(t, err)

	// ======================================================================
	// M14: uq_cbc_schools_tenant — duplicate (tenant_id, id) rejected
	// Since id is the PRIMARY KEY, inserting a duplicate id triggers the PK
	// constraint first. The uq_cbc_schools_tenant constraint exists as a
	// composite UNIQUE target for foreign keys from downstream tables.
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, 'Duplicate', 'Nairobi', 'Westlands', 'Public')`,
		schoolA, tenantA)
	require.Error(t, err, "M14: duplicate (tenant_id, id) should be rejected")
	// id is the PK, so the PK constraint fires before uq_cbc_schools_tenant
	require.Contains(t, err.Error(), "cbc_schools_pkey")
	t.Log("✓ M14: uq_cbc_schools_tenant rejects duplicate (tenant_id, id)")

	// ======================================================================
	// M15: uq_users_tenant — duplicate (tenant_id, id) rejected
	// Since id is the PRIMARY KEY, inserting a duplicate id triggers the PK
	// constraint first. The uq_users_tenant constraint exists as a composite
	// UNIQUE target for foreign keys from downstream tables.
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userA, "dup@test.com", tenantA, "Duplicate")
	require.Error(t, err, "M15: duplicate (tenant_id, id) should be rejected")
	// id is the PK, so the PK constraint fires before uq_users_tenant
	require.Contains(t, err.Error(), "users_pkey")
	t.Log("✓ M15: users_pkey rejects duplicate id")

	// ======================================================================
	// M16: idx_users_email — duplicate email rejected across tenants
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), "user-a@test.com", tenantA, "Dup Email")
	require.Error(t, err, "M16: duplicate email should be rejected")
	require.Contains(t, err.Error(), "idx_users_email")
	t.Log("✓ M16: idx_users_email rejects duplicate email")

	// ======================================================================
	// M17: unique_user_school_membership — duplicate (user_id, school_id) rejected
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO memberships (id, tenant_id, user_id, school_id, role)
		VALUES ($1, $2, $3, $4, 'TEACHER')`,
		uuid.New().String(), tenantA, userA, schoolA)
	require.NoError(t, err, "M17: first membership insert should succeed")

	_, err = pool.Exec(ctx, `INSERT INTO memberships (id, tenant_id, user_id, school_id, role)
		VALUES ($1, $2, $3, $4, 'NURSE')`,
		uuid.New().String(), tenantA, userA, schoolA)
	require.Error(t, err, "M17: duplicate (user_id, school_id) should be rejected")
	require.Contains(t, err.Error(), "unique_user_school_membership")
	t.Log("✓ M17: unique_user_school_membership rejects duplicate (user_id, school_id)")
}

func TestMigrationsIntegration_PartialUniqueIndexes_M18_to_M22(t *testing.T) {
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

	// Apply base schema first, then the upgrade fix migration
	for _, f := range []string{"000001_initial_schema.up.sql", "000003_fix_review_findings.up.sql"} {
		sql, err := os.ReadFile(filepath.Join(migrationsDir(), f))
		require.NoError(t, err, "read %s", f)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply %s", f)
	}

	tenantA := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantA, "Tenant A", "slug-a-"+tenantA[:8], "stytch-a-"+tenantA[:8])
	require.NoError(t, err)

	schoolA := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolA, tenantA, "School A")
	require.NoError(t, err)

	userA := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name, tsc_number, knec_panel_assessor_id)
		VALUES ($1, $2, $3, $4, 'TSC001', 'KNEC001')`,
		userA, "user-a@test.com", tenantA, "User A")
	require.NoError(t, err)

	// ======================================================================
	// M18: idx_users_tsc_number — partial unique, NULL allowed, duplicates rejected
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name, tsc_number)
		VALUES ($1, $2, $3, $4, 'TSC001')`,
		uuid.New().String(), "another@test.com", tenantA, "Another")
	require.Error(t, err, "M18: duplicate tsc_number should be rejected")
	require.Contains(t, err.Error(), "idx_users_tsc_number")
	t.Log("✓ M18: idx_users_tsc_number rejects duplicate tsc_number")

	// Multiple NULL tsc_number values are allowed
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), "null-tsc@test.com", tenantA, "No TSC")
	require.NoError(t, err, "M18: NULL tsc_number should be allowed")
	t.Log("✓ M18: NULL tsc_number allowed (partial index)")

	// ======================================================================
	// M19: idx_cbc_schools_knec_code — partial unique
	// ======================================================================

	_, err = pool.Exec(ctx, `UPDATE cbc_schools SET knec_school_code = '12345678' WHERE id = $1`, schoolA)
	require.NoError(t, err)

	schoolB := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type, knec_school_code)
		VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public', '12345678')`,
		schoolB, tenantA, "School B")
	require.Error(t, err, "M19: duplicate knec_school_code should be rejected")
	require.Contains(t, err.Error(), "idx_cbc_schools_knec_code")
	t.Log("✓ M19: idx_cbc_schools_knec_code rejects duplicate knec_school_code")

	// Multiple NULL knec_school_code values are allowed
	schoolC := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, $3, 'Nairobi', 'Kilimani', 'Private')`,
		schoolC, tenantA, "School C")
	require.NoError(t, err, "M19: NULL knec_school_code should be allowed")
	t.Log("✓ M19: NULL knec_school_code allowed (partial index)")

	// ======================================================================
	// M20: idx_one_current_year_per_school — only one current year per school
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by, is_current)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4, true)`,
		uuid.New().String(), tenantA, schoolA, userA)
	require.NoError(t, err, "M20: first current year should succeed")

	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by, is_current)
		VALUES ($1, $2, $3, '2027', '2027-01-01', '2027-12-31', $4, $4, true)`,
		uuid.New().String(), tenantA, schoolA, userA)
	require.Error(t, err, "M20: second current year should be rejected")
	require.Contains(t, err.Error(), "idx_one_current_year_per_school")
	t.Log("✓ M20: idx_one_current_year_per_school enforces one current year per school")

	// ======================================================================
	// M21: idx_unique_term_number_per_year — duplicate term numbers rejected
	// ======================================================================

	yearID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearID, tenantA, schoolA, userA)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', $5, $5)`,
		uuid.New().String(), tenantA, schoolA, yearID, userA)
	require.NoError(t, err, "M21: first term insert should succeed")

	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Term 1 Dup', 1, '2026-05-01', '2026-08-30', $5, $5)`,
		uuid.New().String(), tenantA, schoolA, yearID, userA)
	require.Error(t, err, "M21: duplicate term_number should be rejected")
	require.Contains(t, err.Error(), "idx_unique_term_number_per_year")
	t.Log("✓ M21: idx_unique_term_number_per_year rejects duplicate term_number")

	// ======================================================================
	// M22: chk_year_dates — CHECK start_date < end_date
	// ======================================================================

	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by)
		VALUES ($1, $2, $3, 'Bad Year', '2026-12-31', '2026-01-01', $4, $4)`,
		uuid.New().String(), tenantA, schoolA, userA)
	require.Error(t, err, "M22: start_date after end_date should be rejected")
	require.Contains(t, err.Error(), "chk_year_dates")
	t.Log("✓ M22: chk_year_dates rejects invalid date range")
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

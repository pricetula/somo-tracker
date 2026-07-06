package curriculum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ============================================================================
// Mock Tx
// ============================================================================

// mockTx implements pgx.Tx for testing.
type mockTx struct {
	pgx.Tx // embed so we only implement what we need

	execFunc     func(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	queryRowFunc func(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
	commitFunc   func(ctx context.Context) error
	rollbackFunc func(ctx context.Context) error
	committed    bool
}

func (m *mockTx) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockTx) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, arguments...)
	}
	return &mockRow{id: "mock_id"}
}

func (m *mockTx) Commit(ctx context.Context) error {
	m.committed = true
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}
	return nil
}

func (m *mockTx) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}
	return pgx.ErrTxClosed
}

// mockRow implements pgx.Row for testing.
type mockRow struct {
	id  string
	err error
}

func (r *mockRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 0 {
		if s, ok := dest[0].(*string); ok {
			*s = r.id
		}
	}
	return nil
}

// ── Mock pool ──────────────────────────────────────────────────────────────

// newSeedingService creates a SeedingService backed by a mock tx.
func newSeedingService(beginFn func(ctx context.Context) (pgx.Tx, error)) *SeedingService {
	return &SeedingService{
		beginTx: beginFn,
	}
}

// ============================================================================
// Tests: Filename Parsing
// ============================================================================

func TestDiscoverJSONFiles_Success(t *testing.T) {
	// Create a temp directory with test JSON files
	dir := t.TempDir()

	files := map[string]string{
		"pp1.json":                   `[{"name":"Math","code":"MATH_PP1","education_level":"Early_Years","strands":[]}]`,
		"pp2.json":                   `[{"name":"Math","code":"MATH_PP2","education_level":"Early_Years","strands":[]}]`,
		"grade1.json":                `[{"name":"Math","code":"MATH_G1","education_level":"Early_Years","strands":[]}]`,
		"grade4.json":                `[{"name":"Math","code":"MATH_G4","education_level":"Upper_Primary","strands":[]}]`,
		"grade10.stem.json":          `[{"name":"Core Math","code":"MATH_CORE_G10","education_level":"Senior_School","strands":[]}]`,
		"grade10.socialscience.json": `[{"name":"History","code":"HIST_G10","education_level":"Senior_School","strands":[]}]`,
		"ignored.txt":                "not a json file",
		"unknown.json":               `[{"name":"Unknown","code":"UNKNOWN","education_level":"Early_Years","strands":[]}]`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	gradeFiles, err := discoverJSONFilesFromFS(os.DirFS(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect 6 recognized grade files: pp1, pp2, grade1, grade4,
	// grade10.stem, grade10.socialscience
	// "unknown.json" is skipped because it's not in the mapping.
	// "ignored.txt" is skipped because it's not .json.
	expectedCount := 6
	if len(gradeFiles) != expectedCount {
		t.Fatalf("expected %d grade files, got %d: %+v", expectedCount, len(gradeFiles), gradeFiles)
	}

	// Verify specific mappings
	mapping := map[string]string{
		"pp1":                   "PP1",
		"pp2":                   "PP2",
		"grade1":                "G1",
		"grade4":                "G4",
		"grade10.stem":          "G10",
		"grade10.socialscience": "G10",
	}

	for _, gf := range gradeFiles {
		expectedGrade, ok := mapping[gf.Stem]
		if !ok {
			t.Errorf("unexpected stem %q", gf.Stem)
			continue
		}
		if gf.Grade != expectedGrade {
			t.Errorf("stem %q: expected grade %q, got %q", gf.Stem, expectedGrade, gf.Grade)
		}
	}
}

func TestDiscoverJSONFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	gradeFiles, err := discoverJSONFilesFromFS(os.DirFS(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gradeFiles) != 0 {
		t.Fatalf("expected 0 files, got %d", len(gradeFiles))
	}
}

func TestDiscoverJSONFiles_NonExistentDir(t *testing.T) {
	_, err := discoverJSONFilesFromFS(os.DirFS("/nonexistent/path"))
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// ============================================================================
// Tests: deriveEducationLevel
// ============================================================================

func TestDeriveEducationLevel(t *testing.T) {
	tests := []struct {
		grade    string
		expected string
	}{
		{"PP1", "Early_Years"},
		{"PP2", "Early_Years"},
		{"G1", "Early_Years"},
		{"G2", "Early_Years"},
		{"G3", "Early_Years"},
		{"G4", "Upper_Primary"},
		{"G5", "Upper_Primary"},
		{"G6", "Upper_Primary"},
		{"G7", "Junior_Secondary"},
		{"G8", "Junior_Secondary"},
		{"G9", "Junior_Secondary"},
		{"G10", "Senior_School"},
		{"G11", "Senior_School"},
		{"G12", "Senior_School"},
		{"Unknown", "Early_Years"}, // default
	}

	for _, tc := range tests {
		t.Run(tc.grade, func(t *testing.T) {
			got := deriveEducationLevel(tc.grade)
			if got != tc.expected {
				t.Errorf("deriveEducationLevel(%q) = %q, want %q", tc.grade, got, tc.expected)
			}
		})
	}
}

// ============================================================================
// Tests: SeedSchoolCurriculum — Validation
// ============================================================================

func TestSeedSchoolCurriculum_EmptyTenantID(t *testing.T) {
	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return &mockTx{}, nil
	})

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.Nil, uuid.New(), "/tmp")
	if err == nil {
		t.Fatal("expected error for nil tenant_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSeedSchoolCurriculum_EmptySchoolID(t *testing.T) {
	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return &mockTx{}, nil
	})

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.Nil, "/tmp")
	if err == nil {
		t.Fatal("expected error for nil school_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSeedSchoolCurriculum_EmptyDir(t *testing.T) {
	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return &mockTx{}, nil
	})

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), "")
	if err == nil {
		t.Fatal("expected error for empty dir, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSeedSchoolCurriculum_NoJSONFiles(t *testing.T) {
	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return &mockTx{}, nil
	})

	dir := t.TempDir()
	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), dir)
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: SeedSchoolCurriculum — Happy Path
// ============================================================================

func TestSeedSchoolCurriculum_HappyPath(t *testing.T) {
	// Create temp dir with a PP1 JSON file
	dir := t.TempDir()

	pp1Content := `[
		{
			"name": "Language Activities",
			"code": "LAN_PP1",
			"education_level": "Early_Years",
			"strands": [
				{
					"name": "Listening and Speaking",
					"sub_strands": [
						{
							"name": "Listening and Attention",
							"performance_indicators": [
								"Develop appropriate listening skills",
								"Demonstrate sustained attention"
							]
						},
						{
							"name": "Free Expression",
							"performance_indicators": [
								"Express own opinions confidently"
							]
						}
					]
				},
				{
					"name": "Reading Readiness",
					"sub_strands": [
						{
							"name": "Book Handling Skills",
							"performance_indicators": [
								"Demonstrate correct book orientation"
							]
						}
					]
				}
			]
		},
		{
			"name": "Mathematical Activities",
			"code": "MATH_PP1",
			"education_level": "Early_Years",
			"strands": [
				{
					"name": "Numbers",
					"sub_strands": [
						{
							"name": "Rote Counting",
							"performance_indicators": [
								"Count from 1 to 10",
								"Demonstrate number awareness"
							]
						}
					]
				}
			]
		}
	]`

	if err := os.WriteFile(filepath.Join(dir, "pp1.json"), []byte(pp1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Track the SQL operations that were performed
	laCount := 0
	strandCount := 0
	subStrandCount := 0
	deletes := 0
	laSQLTracked := false

	commitCalled := false

	tx := &mockTx{
		execFunc: func(ctx context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
			if strings.HasPrefix(sql, "DELETE") {
				deletes++
			}
			return pgconn.CommandTag{}, nil
		},
		queryRowFunc: func(ctx context.Context, sql string, _ ...interface{}) pgx.Row {
			if strings.Contains(sql, "cbc_learning_areas") {
				laCount++
				// Verify the INSERT contains grade_level column
				if strings.Contains(sql, "grade_level") {
					laSQLTracked = true
				}
				return &mockRow{id: fmt.Sprintf("la_%d", laCount)}
			}
			if strings.Contains(sql, "cbc_strands") {
				strandCount++
				return &mockRow{id: fmt.Sprintf("strand_%d", strandCount)}
			}
			if strings.Contains(sql, "cbc_sub_strands") {
				subStrandCount++
				return &mockRow{id: fmt.Sprintf("sub_%d", subStrandCount)}
			}
			return &mockRow{id: "unknown"}
		},
		commitFunc: func(ctx context.Context) error {
			commitCalled = true
			return nil
		},
	}

	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return tx, nil
	})

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !commitCalled {
		t.Error("expected commit to be called")
	}

	// Verify grade_level is included in the learning area INSERT
	if !laSQLTracked {
		t.Error("expected learning area INSERT to contain grade_level column")
	}

	// Verify counts
	if laCount != 2 {
		t.Errorf("expected 2 learning area inserts, got %d", laCount)
	}
	if strandCount != 3 {
		t.Errorf("expected 3 strand inserts (2 from LAN + 1 from MATH), got %d", strandCount)
	}
	if subStrandCount != 4 {
		t.Errorf("expected 4 sub-strand inserts (3 from LAN + 1 from MATH), got %d", subStrandCount)
	}
	if deletes != 2 {
		t.Errorf("expected 2 deletes (1 per learning area), got %d", deletes)
	}
}

// ============================================================================
// Tests: SeedSchoolCurriculum — Transaction Rollback on Error
// ============================================================================

func TestSeedSchoolCurriculum_BeginTxError(t *testing.T) {
	expectedErr := errors.New("connection failed")
	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return nil, expectedErr
	})

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pp1.json"), []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSeedSchoolCurriculum_RollbackOnFileError(t *testing.T) {
	dir := t.TempDir()
	// Write an invalid JSON file
	if err := os.WriteFile(filepath.Join(dir, "pp1.json"), []byte(`{invalid json}`), 0644); err != nil {
		t.Fatal(err)
	}

	commitCalled := false

	tx := &mockTx{
		execFunc: func(ctx context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
		queryRowFunc: func(ctx context.Context, sql string, _ ...interface{}) pgx.Row {
			return &mockRow{id: "la_001"}
		},
		commitFunc: func(ctx context.Context) error {
			commitCalled = true
			return nil
		},
	}

	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return tx, nil
	})

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), dir)
	if err == nil {
		t.Fatal("expected error due to invalid JSON, got nil")
	}

	if commitCalled {
		t.Error("commit should NOT have been called on error")
	}
}

func TestSeedSchoolCurriculum_RollbackOnCommitError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pp1.json"), []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}

	commitCalled := false
	expectedCommitErr := errors.New("commit failed")

	tx := &mockTx{
		execFunc: func(ctx context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
		queryRowFunc: func(ctx context.Context, sql string, _ ...interface{}) pgx.Row {
			return &mockRow{id: "mock_id"}
		},
		commitFunc: func(ctx context.Context) error {
			commitCalled = true
			return expectedCommitErr
		},
	}

	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return tx, nil
	})

	err := svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), dir)
	if err == nil {
		t.Fatal("expected error from commit failure, got nil")
	}

	if !commitCalled {
		t.Error("commit should have been attempted")
	}
}

// ============================================================================
// Tests: SeedSchoolCurriculum — ON CONFLICT Re-run Safety
// ============================================================================

func TestSeedSchoolCurriculum_ReRunSafety(t *testing.T) {
	dir := t.TempDir()

	content := `[
		{
			"name": "Language Activities",
			"code": "LAN_PP1",
			"education_level": "Early_Years",
			"strands": [
				{
					"name": "Listening",
					"sub_strands": [
						{
							"name": "Attention",
							"performance_indicators": ["Listen carefully"]
						}
					]
				}
			]
		}
	]`

	if err := os.WriteFile(filepath.Join(dir, "pp1.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Run the seed twice
	runSeed := func() error {
		firstStrandIDs := 0

		tx := &mockTx{
			execFunc: func(ctx context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
				return pgconn.CommandTag{}, nil
			},
			queryRowFunc: func(ctx context.Context, sql string, _ ...interface{}) pgx.Row {
				if strings.Contains(sql, "cbc_learning_areas") {
					return &mockRow{id: "la_001"}
				}
				if strings.Contains(sql, "cbc_strands") {
					firstStrandIDs++
					return &mockRow{id: fmt.Sprintf("strand_r%d", firstStrandIDs)}
				}
				if strings.Contains(sql, "cbc_sub_strands") {
					return &mockRow{id: fmt.Sprintf("sub_r%d", firstStrandIDs)}
				}
				return &mockRow{id: "mock_id"}
			},
			commitFunc: func(ctx context.Context) error {
				return nil
			},
		}

		svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
			return tx, nil
		})

		return svc.SeedSchoolCurriculum(context.Background(), uuid.New(), uuid.New(), dir)
	}

	// First run
	if err := runSeed(); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Second run (re-run) — should succeed without error
	if err := runSeed(); err != nil {
		t.Fatalf("second run (re-run) failed: %v", err)
	}
}

// ============================================================================
// Tests: GradeFile — gradeMapping completeness
// ============================================================================

func TestGradeMapping_CoversAllExpectedFiles(t *testing.T) {
	// Verify all expected grade levels are covered
	expectedGrades := []struct {
		stem  string
		grade string
	}{
		{"pp1", "PP1"},
		{"pp2", "PP2"},
		{"grade1", "G1"},
		{"grade2", "G2"},
		{"grade3", "G3"},
		{"grade4", "G4"},
		{"grade5", "G5"},
		{"grade6", "G6"},
		{"grade7", "G7"},
		{"grade8", "G8"},
		{"grade9", "G9"},
		{"grade10.stem", "G10"},
		{"grade10.socialscience", "G10"},
		{"grade10.artssportscience", "G10"},
		{"grade11.stem", "G11"},
		{"grade11.socialscience", "G11"},
		{"grade11.artssportscience", "G11"},
		{"grade12.stem", "G12"},
		{"grade12.socialscience", "G12"},
		{"grade12.artssportscience", "G12"},
	}

	for _, eg := range expectedGrades {
		got, ok := gradeMapping[eg.stem]
		if !ok {
			t.Errorf("gradeMapping missing entry for stem %q", eg.stem)
			continue
		}
		if got != eg.grade {
			t.Errorf("gradeMapping[%q] = %q, want %q", eg.stem, got, eg.grade)
		}
	}
}

// ============================================================================
// Test: Embedded Curriculum Default
// ============================================================================

func TestSeedSchoolCurriculumDefault_EmbeddedFSIsAccessible(t *testing.T) {
	// Verify the embedded filesystem contains all expected files
	gradeFiles, err := discoverJSONFilesFromFS(defaultCBCFS)
	if err != nil {
		t.Fatalf("discover from embedded FS failed: %v", err)
	}

	// We expect 20 files: pp1, pp2, grade1-grade9 (single), grade10-grade12 (3 each)
	expectedCount := 20
	if len(gradeFiles) != expectedCount {
		t.Fatalf("expected %d grade files in embedded FS, got %d: %+v",
			expectedCount, len(gradeFiles), gradeFiles)
	}

	// Verify a specific file can be read and parsed
	pp1Data, err := fs.ReadFile(defaultCBCFS, "pp1.json")
	if err != nil {
		t.Fatalf("failed to read pp1.json from embedded FS: %v", err)
	}

	var las CurriculumData
	if err := json.Unmarshal(pp1Data, &las); err != nil {
		t.Fatalf("failed to parse pp1.json: %v", err)
	}

	if len(las) == 0 {
		t.Fatal("pp1.json should contain at least one learning area")
	}

	// Verify transaction path works with embedded FS
	tx := &mockTx{
		execFunc: func(ctx context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
		queryRowFunc: func(ctx context.Context, sql string, _ ...interface{}) pgx.Row {
			return &mockRow{id: "embedded_la"}
		},
		commitFunc: func(ctx context.Context) error {
			return nil
		},
	}

	svc := newSeedingService(func(ctx context.Context) (pgx.Tx, error) {
		return tx, nil
	})

	err = svc.SeedSchoolCurriculumDefault(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("SeedSchoolCurriculumDefault failed: %v", err)
	}
}

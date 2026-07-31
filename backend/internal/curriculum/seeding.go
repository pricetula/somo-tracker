package curriculum

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"somotracker/backend/internal/database"
)

// ── Embedded Curriculum Files ──────────────────────────────────────────────

// cbcCurriculumFS embeds all CBC curriculum JSON files from the cbcdata/
// subdirectory into the binary so they are always available in production
// without any filesystem deployment.
//
//go:embed cbcdata/*.json
var cbcCurriculumFS embed.FS

// ── Filename-to-Grade mapping ──────────────────────────────────────────────

// gradeMapping translates file stems to CBC grade level identifiers.
// E.g. "pp1" → "PP1", "grade4" → "G4", "grade10.stem" → "G10".
// Built via an IIFE to avoid a bare init() function.
var gradeMapping = func() map[string]string {
	m := map[string]string{
		"pp1": "PP1",
		"pp2": "PP2",
	}
	for i := 1; i <= 9; i++ {
		m[fmt.Sprintf("grade%d", i)] = fmt.Sprintf("G%d", i)
	}
	for i := 10; i <= 12; i++ {
		for _, suffix := range []string{"stem", "socialscience", "artssportscience"} {
			m[fmt.Sprintf("grade%d.%s", i, suffix)] = fmt.Sprintf("G%d", i)
		}
	}
	return m
}()

// ── SeedingService ─────────────────────────────────────────────────────────

// SeedingService ingests CBC curriculum JSON files and seeds them into the
// database within a single transaction.
type SeedingService struct {
	pool    *pgxpool.Pool
	beginTx func(ctx context.Context) (pgx.Tx, error)
}

// NewSeedingService creates a new SeedingService.
func NewSeedingService(pools *database.Pools) *SeedingService {
	return &SeedingService{
		pool: pools.PG,
		beginTx: func(ctx context.Context) (pgx.Tx, error) {
			return pools.PG.Begin(ctx)
		},
	}
}

// defaultCBCFS is the sub-filesystem rooted at cbcdata/, used by
// SeedSchoolCurriculumDefault. Exposed as a var so tests can override it.
// Built via an IIFE to avoid a bare init() function.
var defaultCBCFS = func() fs.FS {
	sub, err := fs.Sub(cbcCurriculumFS, "cbcdata")
	if err != nil {
		// This should never happen — cbcdata/ is embedded and always present.
		panic("curriculum: failed to create embedded sub-filesystem: " + err.Error())
	}
	return sub
}()

// SeedSchoolCurriculumDefault seeds the curriculum using the embedded
// cbcdata/ JSON files that are compiled into the binary. This is the
// recommended method for production use — no external files needed.
func (s *SeedingService) SeedSchoolCurriculumDefault(
	ctx context.Context,
	tenantID, schoolID uuid.UUID,
) error {
	return s.seedFromFS(ctx, tenantID, schoolID, defaultCBCFS, "cbcdata")
}

// SeedSchoolCurriculum ingests all .json files from the given filesystem
// directory and seeds the curriculum hierarchy (learning areas → strands →
// sub-strands → performance indicators) for the specified tenant and school.
// The entire operation is atomic — if any file fails, the transaction is
// rolled back.
//
// For production use, prefer SeedSchoolCurriculumDefault which uses the
// embedded files. This method is useful for tests with custom data or when
// overriding the curriculum files at runtime.
//
// Filenames must follow the convention described in gradeMapping (e.g.
// "pp1.json", "grade4.json", "grade10.stem.json"). Learning areas use
// ON CONFLICT (tenant_id, school_id, code) for safe re-runs.
func (s *SeedingService) SeedSchoolCurriculum(
	ctx context.Context,
	tenantID, schoolID uuid.UUID,
	cbcDir string,
) error {
	if cbcDir == "" {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: cbcDir is required: %w", ErrInvalidInput)
	}
	return s.seedFromFS(ctx, tenantID, schoolID, os.DirFS(cbcDir), cbcDir)
}

// seedFromFS is the shared implementation for both embedded and filesystem
// seeding. It accepts an fs.FS, discovers .json files matching the grade
// naming convention, and seeds them all within a single transaction.
func (s *SeedingService) seedFromFS(
	ctx context.Context,
	tenantID, schoolID uuid.UUID,
	cbcFS fs.FS,
	fsLabel string,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: tenant_id is required: %w", ErrInvalidInput)
	}
	if schoolID == uuid.Nil {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: school_id is required: %w", ErrInvalidInput)
	}

	// 1. Discover JSON files from the FS
	files, err := discoverJSONFilesFromFS(cbcFS)
	if err != nil {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: no .json files found in %q: %w", fsLabel, ErrInvalidInput)
	}

	// 2. Start a single database transaction for the entire seed
	tx, err := s.beginTx(ctx)
	if err != nil {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			slog.WarnContext(ctx, "seeding: deferred tx rollback returned unexpected error",
				slog.String("error", rbErr.Error()),
			)
		}
	}()

	tenantIDStr := tenantID.String()
	schoolIDStr := schoolID.String()

	// 3. Process each file
	for _, gf := range files {
		if err := s.seedFileFromFS(ctx, tx, gf, cbcFS, tenantIDStr, schoolIDStr); err != nil {
			return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: %w", err)
		}
	}

	// 4. Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("curriculum.SeedingService.SeedSchoolCurriculum: commit: %w", err)
	}

	return nil
}

// ── File Discovery ─────────────────────────────────────────────────────────

// discoverJSONFilesFromFS reads all .json files from the given fs.FS,
// parses the filenames, and returns a list of GradeFile entries. Non-JSON
// files and unrecognised stems are skipped silently.
func discoverJSONFilesFromFS(cbcFS fs.FS) ([]GradeFile, error) {
	entries, err := fs.ReadDir(cbcFS, ".")
	if err != nil {
		return nil, fmt.Errorf("discoverJSONFilesFromFS: read dir: %w", err)
	}

	var files []GradeFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		stem := strings.TrimSuffix(entry.Name(), ".json")
		grade, ok := gradeMapping[stem]
		if !ok {
			continue
		}

		files = append(files, GradeFile{
			Stem:  stem,
			Grade: grade,
		})
	}

	if files == nil {
		files = []GradeFile{}
	}

	return files, nil
}

// ── Per-File Seeding ───────────────────────────────────────────────────────

// seedFileFromFS reads a single JSON file from the given fs.FS and seeds its
// contents into the database within the given transaction.
func (s *SeedingService) seedFileFromFS(
	ctx context.Context,
	tx pgx.Tx,
	gf GradeFile,
	cbcFS fs.FS,
	tenantID, schoolID string,
) error {
	fileName := gf.Stem + ".json"

	data, err := fs.ReadFile(cbcFS, fileName)
	if err != nil {
		return fmt.Errorf("seeding.seedFileFromFS: read %q: %w", fileName, err)
	}

	var las CurriculumData
	if err := json.Unmarshal(data, &las); err != nil {
		return fmt.Errorf("seeding.seedFileFromFS: parse %q: %w", fileName, err)
	}

	for _, la := range las {
		if err := s.seedLearningArea(ctx, tx, la, gf.Grade, tenantID, schoolID); err != nil {
			return fmt.Errorf("seeding.seedFileFromFS: %q → learning area %q: %w", fileName, la.Name, err)
		}
	}

	return nil
}

// ── Database Operations ────────────────────────────────────────────────────

// seedLearningArea inserts (or upserts) a learning area and seeds its children.
func (s *SeedingService) seedLearningArea(
	ctx context.Context,
	tx pgx.Tx,
	la LearningAreaInput,
	grade, tenantID, schoolID string,
) error {
	educationLevel := la.EducationLevel
	if educationLevel == "" {
		// Derive from grade if not provided
		educationLevel = deriveEducationLevel(grade)
	}

	// Upsert learning area: INSERT ON CONFLICT DO UPDATE so that RETURNING id
	// always works, even on re-runs.
	// grade_level is stored explicitly per grade file.
	const upsertLA = `
		INSERT INTO cbc_learning_areas (tenant_id, school_id, name, code, education_level, grade_level)
		VALUES ($1, $2, $3, $4, $5::cbc_education_level, $6::cbc_grade_level)
		ON CONFLICT (tenant_id, school_id, code, grade_level)
		DO UPDATE SET name = EXCLUDED.name, education_level = EXCLUDED.education_level
		RETURNING id
	`
	var learningAreaID string
	err := tx.QueryRow(ctx, upsertLA,
		tenantID, schoolID, la.Name, la.Code, educationLevel, grade,
	).Scan(&learningAreaID)
	if err != nil {
		return fmt.Errorf("seeding.seedLearningArea: upsert %q: %w", la.Code, err)
	}

	// Delete existing children for this learning area so we can re-insert cleanly.
	// (Strands CASCADE to sub-strands and indicators.)
	const deleteStrands = `DELETE FROM cbc_strands WHERE learning_area_id = $1`
	if _, err := tx.Exec(ctx, deleteStrands, learningAreaID); err != nil {
		return fmt.Errorf("seeding.seedLearningArea: delete strands for %q: %w", la.Code, err)
	}

	// Re-insert strands
	for _, strand := range la.Strands {
		if err := s.seedStrand(ctx, tx, strand, tenantID, learningAreaID, grade); err != nil {
			return fmt.Errorf("seeding.seedLearningArea: %q → strand %q: %w", la.Code, strand.Name, err)
		}
	}

	return nil
}

// seedStrand inserts a strand and seeds its sub-strands.
func (s *SeedingService) seedStrand(
	ctx context.Context,
	tx pgx.Tx,
	strand StrandInput,
	tenantID, learningAreaID, grade string,
) error {
	const insertStrand = `
		INSERT INTO cbc_strands (learning_area_id, name, tenant_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var strandID string
	err := tx.QueryRow(ctx, insertStrand, learningAreaID, strand.Name, tenantID).Scan(&strandID)
	if err != nil {
		return fmt.Errorf("seeding.seedStrand: insert %q: %w", strand.Name, err)
	}

	for _, ss := range strand.SubStrands {
		if err := s.seedSubStrand(ctx, tx, ss, tenantID, strandID); err != nil {
			return fmt.Errorf("seeding.seedStrand: %q → sub-strand %q: %w", strand.Name, ss.Name, err)
		}
	}

	return nil
}

// seedSubStrand inserts a sub-strand and its performance indicators.
func (s *SeedingService) seedSubStrand(
	ctx context.Context,
	tx pgx.Tx,
	ss SubStrandInput,
	tenantID, strandID string,
) error {
	const insertSubStrand = `
		INSERT INTO cbc_sub_strands (strand_id, name, tenant_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var subStrandID string
	err := tx.QueryRow(ctx, insertSubStrand, strandID, ss.Name, tenantID).Scan(&subStrandID)
	if err != nil {
		return fmt.Errorf("seeding.seedSubStrand: insert %q: %w", ss.Name, err)
	}

	// Insert performance indicators with 1-indexed sequence_order
	for i, pi := range ss.PerformanceIndicators {
		if err := s.seedPerformanceIndicator(ctx, tx, pi, tenantID, subStrandID, i+1); err != nil {
			return fmt.Errorf("seeding.seedSubStrand: %q → PI %d: %w", ss.Name, i+1, err)
		}
	}

	return nil
}

// seedPerformanceIndicator inserts a single performance indicator.
func (s *SeedingService) seedPerformanceIndicator(
	ctx context.Context,
	tx pgx.Tx,
	description, tenantID, subStrandID string,
	sequenceOrder int,
) error {
	const insertPI = `
		INSERT INTO performance_indicators (sub_strand_id, description, sequence_order, tenant_id)
		VALUES ($1, $2, $3, $4)
	`
	_, err := tx.Exec(ctx, insertPI, subStrandID, description, sequenceOrder, tenantID)
	if err != nil {
		return fmt.Errorf("seeding.seedPerformanceIndicator: %w", err)
	}
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

// deriveEducationLevel maps a grade level to its CBC education level.
func deriveEducationLevel(grade string) string {
	switch grade {
	case "PP1", "PP2", "G1", "G2", "G3":
		return "Early_Years"
	case "G4", "G5", "G6":
		return "Upper_Primary"
	case "G7", "G8", "G9":
		return "Junior_Secondary"
	case "G10", "G11", "G12":
		return "Senior_School"
	default:
		return "Early_Years"
	}
}

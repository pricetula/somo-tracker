package students

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"somotracker/backend/internal/imports"
)

// gradeStream is a (grade_level, stream_name) pair used for class resolution lookups.
type gradeStream struct {
	GradeLevel string
	StreamName string
}

// ============================================================================
// AugmentedRow is what ValidatedRow.RawData becomes after ResolveReferences.
// ============================================================================

// augmentedImportRow extends ImportRow with the resolved class_id and
// tenant/school/academic context injected by ResolveReferences.
type augmentedImportRow struct {
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	AdmissionNumber      *string `json:"admission_number,omitempty"`
	GradeLevel           string  `json:"grade_level"`
	StreamName           string  `json:"stream_name"`
	ResolvedClassID      *string `json:"resolved_class_id,omitempty"`
	TenantID             string  `json:"tenant_id"`
	SchoolID             string  `json:"school_id"`
	AcademicTermID       string  `json:"academic_term_id"`
	AcademicYearID       string  `json:"academic_year_id"`
}

// ============================================================================
// StudentImporter — implements imports.Importer for student bulk import
// ============================================================================

// StudentImporter handles the student-specific import logic.
type StudentImporter struct {
	repo ImportRepository
}

// NewStudentImporter creates a new StudentImporter.
func NewStudentImporter(repo ImportRepository) *StudentImporter {
	return &StudentImporter{repo: repo}
}

// JobType returns STUDENT_IMPORT.
func (si *StudentImporter) JobType() imports.ImportJobType {
	return imports.ImportJobTypeStudentImport
}

// Validate checks each raw row for schema correctness and business rules.
func (si *StudentImporter) Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]imports.ValidatedRow, []imports.RowFailure) {
	var validated []imports.ValidatedRow
	var failures []imports.RowFailure

	for i, rawData := range raw {
		var row ImportRow
		if err := json.Unmarshal(rawData, &row); err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("invalid JSON: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		// Schema validation
		if row.FullName == "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: "full_name is required",
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		if row.Gender != "M" && row.Gender != "F" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("invalid gender %q (must be M or F)", row.Gender),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		if row.GradeLevel == "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: "grade_level is required",
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		if row.StreamName == "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: "stream_name is required",
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		validated = append(validated, imports.ValidatedRow{RawData: rawData})
	}

	if len(failures) > 0 {
		slog.Debug("students.StudentImporter.Validate: schema validation failures",
			"total", len(raw),
			"valid", len(validated),
			"failed", len(failures),
		)
	}

	return validated, failures
}

// ResolveReferences resolves grade_level + stream_name pairs to class_ids
// by querying cbc_classes / cbc_streams in bulk, and injects the results
// into each row's RawData.
func (si *StudentImporter) ResolveReferences(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []imports.ValidatedRow) ([]imports.ValidatedRow, []imports.RowFailure) {
	if len(rows) == 0 {
		return rows, nil
	}

	// Parse metadata to get academic_term_id and academic_year_id
	var meta struct {
		AcademicTermID string `json:"academic_term_id"`
		AcademicYearID string `json:"academic_year_id"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil {
		slog.Error("students.StudentImporter.ResolveReferences: invalid metadata", "error", err)
		// Fail all rows
		failures := make([]imports.RowFailure, 0, len(rows))
		for i, row := range rows {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   row.RawData,
				ErrorMessage: "job metadata missing academic_term_id or academic_year_id",
				ErrorType:    imports.ImportFailureBusinessRule,
			})
		}
		return nil, failures
	}

	if meta.AcademicTermID == "" || meta.AcademicYearID == "" {
		failures := make([]imports.RowFailure, 0, len(rows))
		for i, row := range rows {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   row.RawData,
				ErrorMessage: "job metadata missing academic_term_id or academic_year_id",
				ErrorType:    imports.ImportFailureBusinessRule,
			})
		}
		return nil, failures
	}

	// Step 1: Collect all distinct (grade_level, stream_name) pairs
	distinct := make(map[gradeStream]bool)
	rowGrades := make([]gradeStream, len(rows))
	for i, row := range rows {
		var importRow ImportRow
		if err := json.Unmarshal(row.RawData, &importRow); err != nil {
			// Should not happen since Validate already succeeded
			continue
		}
		gs := gradeStream{GradeLevel: importRow.GradeLevel, StreamName: importRow.StreamName}
		distinct[gs] = true
		rowGrades[i] = gs
	}

	// Step 2: Build a lookup map from distinct pairs
	// Query cbc_classes JOIN cbc_streams to resolve (grade_level, stream_name) → class_id
	lookup, err := si.buildClassLookup(ctx, tenantID.String(), schoolID.String(), meta.AcademicYearID, distinct)
	if err != nil {
		// Log but continue — unresolvable rows will fail individually
		slog.ErrorContext(ctx, "students.StudentImporter.ResolveReferences: build class lookup failed",
			"error", err,
		)
	}

	// Step 3: Build resolved rows, failing any that can't be resolved
	var resolved []imports.ValidatedRow
	var failures []imports.RowFailure

	for i, row := range rows {
		gs := rowGrades[i]
		classID, found := lookup[gs]

		aug := augmentedImportRow{
			TenantID:        tenantID.String(),
			SchoolID:        schoolID.String(),
			AcademicTermID:  meta.AcademicTermID,
			AcademicYearID:  meta.AcademicYearID,
			ResolvedClassID: classID,
		}

		// Unmarshal the original row into the augmented struct
		var importRow ImportRow
		if err := json.Unmarshal(row.RawData, &importRow); err == nil {
			aug.FullName = importRow.FullName
			aug.Gender = importRow.Gender
			aug.DateOfBirth = importRow.DateOfBirth
			aug.UPINumber = importRow.UPINumber
			aug.KNECAssessmentNumber = importRow.KNECAssessmentNumber
			aug.AdmissionNumber = importRow.AdmissionNumber
			aug.GradeLevel = importRow.GradeLevel
			aug.StreamName = importRow.StreamName
		}

		augData, err := json.Marshal(aug)
		if err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   row.RawData,
				ErrorMessage: fmt.Sprintf("marshal augmented row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		if !found || classID == nil {
			var resolvedClassIDStr string
			if classID != nil {
				resolvedClassIDStr = *classID
			}
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   row.RawData,
				ErrorMessage: fmt.Sprintf("grade_level %q + stream_name %q could not be resolved to a class (got %q)", gs.GradeLevel, gs.StreamName, resolvedClassIDStr),
				ErrorType:    imports.ImportFailureBusinessRule,
			})
			continue
		}

		resolved = append(resolved, imports.ValidatedRow{RawData: augData})
	}

	return resolved, failures
}

// buildClassLookup queries cbc_classes JOIN cbc_streams to resolve
// (grade_level, stream_name) → class_id for all distinct pairs.
func (si *StudentImporter) buildClassLookup(ctx context.Context, tenantID, schoolID, academicYearID string, pairs map[gradeStream]bool) (map[gradeStream]*string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	// Query using the repository
	// We need a method that returns class_id for a given (tenant_id, school_id, academic_year_id, grade_level, stream_name)
	// Since ImportRepository only has ResolveClassByGradeAndStream (single-row), we'll iterate.
	lookup := make(map[gradeStream]*string, len(pairs))
	for p := range pairs {
		classID, err := si.repo.ResolveClassByGradeAndStream(ctx, tenantID, schoolID, academicYearID, p.GradeLevel, p.StreamName)
		if err != nil {
			slog.WarnContext(ctx, "students.StudentImporter.buildClassLookup: resolve failed",
				"grade_level", p.GradeLevel,
				"stream_name", p.StreamName,
				"error", err,
			)
			// Don't fail yet — nil classID means unresolvable
		}
		lookup[p] = classID
	}

	return lookup, nil
}

// BulkInsert attempts to insert all resolved rows.
// For student imports, we do the insert inside the executor but we resolve
// class IDs via ResolveReferences first. Here we attempt the multi-row INSERT.
func (si *StudentImporter) BulkInsert(ctx context.Context, tx pgx.Tx, rows []imports.ValidatedRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	// We can't do a single multi-row INSERT across cbc_students and
	// cbc_student_enrollments since the student ID is generated per row.
	// Instead, we insert students one at a time but attempt a bulk enrollment insert.
	//
	// Return an error to force the savepoint fallback which will handle
	// the per-row logic correctly.
	return 0, fmt.Errorf("student import requires per-row inserts for student+enrollment pair")
}

// InsertOne inserts a single student + enrollment pair inside a savepoint.
func (si *StudentImporter) InsertOne(ctx context.Context, tx pgx.Tx, row imports.ValidatedRow) error {
	var aug augmentedImportRow
	if err := json.Unmarshal(row.RawData, &aug); err != nil {
		return fmt.Errorf("unmarshal augmented row: %w", err)
	}

	// Guard: resolved_class_id must be present
	if aug.ResolvedClassID == nil || *aug.ResolvedClassID == "" {
		return fmt.Errorf("grade_level %q + stream_name %q could not be resolved to a class", aug.GradeLevel, aug.StreamName)
	}

	// Step 1: Insert the student record
	studentID, err := si.insertStudent(ctx, tx, aug)
	if err != nil {
		return fmt.Errorf("insert student: %w", err)
	}

	// Step 2: Insert the enrollment
	if err := si.insertEnrollment(ctx, tx, studentID, aug); err != nil {
		return fmt.Errorf("insert enrollment for student %s: %w", studentID, err)
	}

	return nil
}

// insertStudent inserts a single student row and returns the new ID.
func (si *StudentImporter) insertStudent(ctx context.Context, tx pgx.Tx, aug augmentedImportRow) (string, error) {
	query := `
		INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, date_of_birth,
		                          upi_number, knec_assessment_number, admission_number, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		RETURNING id
	`
	var id string
	err := tx.QueryRow(ctx, query,
		aug.TenantID, aug.SchoolID, aug.FullName, aug.Gender,
		aug.DateOfBirth, aug.UPINumber, aug.KNECAssessmentNumber, aug.AdmissionNumber,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// insertEnrollment inserts a student enrollment.
func (si *StudentImporter) insertEnrollment(ctx context.Context, tx pgx.Tx, studentID string, aug augmentedImportRow) error {
	query := `
		INSERT INTO cbc_student_enrollments (tenant_id, school_id, student_id, academic_term_id, class_id, status)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE')
	`
	_, err := tx.Exec(ctx, query,
		aug.TenantID, aug.SchoolID, studentID, aug.AcademicTermID, aug.ResolvedClassID,
	)
	if err != nil {
		return err
	}
	return nil
}

// compile-time interface check
var _ imports.Importer = (*StudentImporter)(nil)

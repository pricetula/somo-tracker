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

// ============================================================================
// AugmentedRow is what ValidatedRow.RawData becomes after ResolveReferences.
// ============================================================================

// augmentedImportRow extends ImportRow with the tenant/school/academic context
// injected by ResolveReferences plus the staging row ID for idempotent inserts.
type augmentedImportRow struct {
	FullName             string  `json:"full_name"`
	Gender               string  `json:"gender"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	UPINumber            *string `json:"upi_number,omitempty"`
	KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
	AdmissionNumber      *string `json:"admission_number,omitempty"`
	ClassID              string  `json:"class_id,omitempty"`
	StagingRowID         string  `json:"staging_row_id,omitempty"`
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
// class_id is optional — when empty, the student is created without an enrollment.
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

// ResolveReferences injects metadata (tenant_id, school_id, academic_term_id,
// academic_year_id) and staging_row_id into each row. No class resolution is
// needed since class_id is passed directly from the frontend.
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
		return nil, allFail(rows, "job metadata is invalid")
	}

	if meta.AcademicTermID == "" || meta.AcademicYearID == "" {
		return nil, allFail(rows, "job metadata missing academic_term_id or academic_year_id")
	}

	// Build resolved rows by injecting tenant/school/academic context
	var resolved []imports.ValidatedRow
	var failures []imports.RowFailure

	for i, row := range rows {
		var importRow ImportRow
		if err := json.Unmarshal(row.RawData, &importRow); err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   row.RawData,
				ErrorMessage: fmt.Sprintf("unmarshal row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		aug := augmentedImportRow{
			TenantID:             tenantID.String(),
			SchoolID:             schoolID.String(),
			AcademicTermID:       meta.AcademicTermID,
			AcademicYearID:       meta.AcademicYearID,
			FullName:             importRow.FullName,
			Gender:               importRow.Gender,
			DateOfBirth:          importRow.DateOfBirth,
			UPINumber:            importRow.UPINumber,
			KNECAssessmentNumber: importRow.KNECAssessmentNumber,
			AdmissionNumber:      importRow.AdmissionNumber,
			ClassID:              importRow.ClassID,
			StagingRowID:         row.StagingRowID.String(),
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

		resolved = append(resolved, imports.ValidatedRow{
			RawData:      augData,
			StagingRowID: row.StagingRowID,
		})
	}

	return resolved, failures
}

// BulkInsert attempts to insert all resolved rows.
// For student imports we use per-row inserts (returns error to trigger savepoint fallback).
func (si *StudentImporter) BulkInsert(ctx context.Context, tx pgx.Tx, rows []imports.ValidatedRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	return 0, fmt.Errorf("student import requires per-row inserts for student+enrollment pair")
}

// InsertOne inserts a single student (and optionally an enrollment) inside a savepoint.
// The student INSERT includes staging_row_id for idempotent redelivery safety.
// If a (school_id, staging_row_id) unique constraint is violated (the row was
// already inserted by a prior attempt), it is treated as success — the row is
// not recorded as a failure.
func (si *StudentImporter) InsertOne(ctx context.Context, tx pgx.Tx, row imports.ValidatedRow) error {
	var aug augmentedImportRow
	if err := json.Unmarshal(row.RawData, &aug); err != nil {
		return fmt.Errorf("unmarshal augmented row: %w", err)
	}

	// Step 1: Insert the student record with staging_row_id for idempotency
	studentID, err := si.insertStudent(ctx, tx, aug)
	if err != nil {
		return fmt.Errorf("insert student: %w", err)
	}

	// Step 2: Insert the enrollment only if class_id was provided
	if aug.ClassID != "" {
		if err := si.insertEnrollment(ctx, tx, studentID, aug); err != nil {
			return fmt.Errorf("insert enrollment for student %s: %w", studentID, err)
		}
	}

	return nil
}

// insertStudent inserts a single student row and returns the new ID.
// Uses ON CONFLICT on (school_id, staging_row_id) for defense-in-depth:
// if a row with this staging_row_id already exists (crash after insert but before
// staging mark), the conflict is treated as a no-op success and the existing ID
// is returned rather than surfacing a duplicate error.
func (si *StudentImporter) insertStudent(ctx context.Context, tx pgx.Tx, aug augmentedImportRow) (string, error) {
	query := `
		INSERT INTO cbc_students (tenant_id, school_id, full_name, gender, date_of_birth,
		                          upi_number, knec_assessment_number, admission_number,
		                          staging_row_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true)
		ON CONFLICT (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL
		DO UPDATE SET staging_row_id = EXCLUDED.staging_row_id
		RETURNING id
	`
	var id string
	err := tx.QueryRow(ctx, query,
		aug.TenantID, aug.SchoolID, aug.FullName, aug.Gender,
		aug.DateOfBirth, aug.UPINumber, aug.KNECAssessmentNumber, aug.AdmissionNumber,
		aug.StagingRowID,
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
		aug.TenantID, aug.SchoolID, studentID, aug.AcademicTermID, aug.ClassID,
	)
	if err != nil {
		return err
	}
	return nil
}

// allFail creates a failure for every row with the given message.
func allFail(rows []imports.ValidatedRow, msg string) []imports.RowFailure {
	failures := make([]imports.RowFailure, 0, len(rows))
	for i, row := range rows {
		failures = append(failures, imports.RowFailure{
			RowNumber:    i,
			RawPayload:   row.RawData,
			ErrorMessage: msg,
			ErrorType:    imports.ImportFailureBusinessRule,
		})
	}
	return failures
}

// compile-time interface check
var _ imports.Importer = (*StudentImporter)(nil)

package students

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// Constants
// ============================================================================

// siInsertStudent and siInsertEnrollment are function variables that can be
// replaced by tests to inject errors without a real database.
var siInsertStudent = (*StudentImporter).insertStudent
var siInsertEnrollment = (*StudentImporter).insertEnrollment

// maxStudentAgeYears is the upper bound for a student's age. Students older
// than this at the time of import are rejected as implausible. The value of
// 25 covers the full CBC range (early childhood through senior secondary)
// plus a reasonable buffer for grade retention or late enrollment.
const maxStudentAgeYears = 25

// knownConstraintMessages maps Postgres constraint/index names to friendly
// error messages. Any constraint violation not listed here falls through to
// the generic DB_CONSTRAINT_VIOLATION handler.
var knownConstraintMessages = map[string]struct {
	Message   string
	ErrorType imports.ImportFailureType
}{
	"fk_enrollments_tenant_class": {
		Message:   "The specified class does not exist or does not belong to this school",
		ErrorType: imports.ImportFailureInvalidClassReference,
	},
	"unique_student_term_enrollment": {
		Message:   "This student is already enrolled in the selected academic term",
		ErrorType: imports.ImportFailureBusinessRule,
	},
}

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
// Checks performed per row:
//   - JSON is valid
//   - full_name is non-empty
//   - gender is "M" or "F"
//   - date_of_birth, if present, is a parseable date, not in the future,
//     and not implausibly old (> maxStudentAgeYears)
//   - class_id, if present, is a well-formed UUID (fail fast before
//     ResolveReferences or insert attempt)
func (si *StudentImporter) Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]imports.ValidatedRow, []imports.RowFailure) {
	var validated []imports.ValidatedRow
	var failures []imports.RowFailure

	today := time.Now().Truncate(24 * time.Hour)
	maxBirthDate := today.AddDate(-maxStudentAgeYears, 0, 0)

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

		// Schema validation: full_name
		if row.FullName == "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: "full_name is required",
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		// Schema validation: gender
		if row.Gender != "M" && row.Gender != "F" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    i,
				RawPayload:   rawData,
				ErrorMessage: fmt.Sprintf("invalid gender %q (must be M or F)", row.Gender),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		// Schema validation: date_of_birth (if present)
		if row.DateOfBirth != nil && *row.DateOfBirth != "" {
			dob, err := time.Parse("2006-01-02", *row.DateOfBirth)
			if err != nil {
				failures = append(failures, imports.RowFailure{
					RowNumber:    i,
					RawPayload:   rawData,
					ErrorMessage: fmt.Sprintf("date_of_birth %q is not a valid date (expected YYYY-MM-DD)", *row.DateOfBirth),
					ErrorType:    imports.ImportFailureSchemaValidation,
				})
				continue
			}

			dobDate := dob.Truncate(24 * time.Hour)
			if dobDate.After(today) {
				failures = append(failures, imports.RowFailure{
					RowNumber:    i,
					RawPayload:   rawData,
					ErrorMessage: fmt.Sprintf("date_of_birth %q is in the future", *row.DateOfBirth),
					ErrorType:    imports.ImportFailureSchemaValidation,
				})
				continue
			}

			if dobDate.Before(maxBirthDate) {
				failures = append(failures, imports.RowFailure{
					RowNumber:    i,
					RawPayload:   rawData,
					ErrorMessage: fmt.Sprintf("date_of_birth %q is implausibly old (max age %d years)", *row.DateOfBirth, maxStudentAgeYears),
					ErrorType:    imports.ImportFailureSchemaValidation,
				})
				continue
			}
		}

		// Schema validation: class_id UUID format (fail fast)
		if row.ClassID != "" {
			if _, err := uuid.Parse(row.ClassID); err != nil {
				failures = append(failures, imports.RowFailure{
					RowNumber:    i,
					RawPayload:   rawData,
					ErrorMessage: fmt.Sprintf("class_id %q is not a valid UUID", row.ClassID),
					ErrorType:    imports.ImportFailureSchemaValidation,
				})
				continue
			}
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
// academic_year_id) and staging_row_id into each row. For rows that include a
// class_id, it verifies that the class exists AND belongs to the same tenant
// and school. If the class is missing or cross-tenant, the row is failed with
// INVALID_CLASS_REFERENCE.
//
// As an insert-time safety net, it also checks admission_number, upi_number,
// and knec_assessment_number against already-persisted students for this
// tenant/school. Any row whose value already exists in the DB fails with the
// matching DUPLICATE_* error type. This catches races between concurrent
// imports and manual-entry submissions that skip the proactive frontend check.
// Within-batch duplicates (two rows in the same submission sharing a value)
// are NOT checked here — that is a frontend-only responsibility.
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

	// ── Phase 1: Unmarshal all rows and collect values for duplicate check ──
	type parsedRow struct {
		index     int
		row       ImportRow
		rawData   json.RawMessage
		stagingID uuid.UUID
	}
	parsed := make([]parsedRow, 0, len(rows))

	var admVals, upiVals, knecVals []string
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

		// Class ID existence + tenant scope check (fail-fast before duplicate check)
		if importRow.ClassID != "" {
			exists, err := si.repo.ValidateClassExists(ctx, tenantID.String(), schoolID.String(), importRow.ClassID)
			if err != nil {
				failures = append(failures, imports.RowFailure{
					RowNumber:    i,
					RawPayload:   row.RawData,
					ErrorMessage: fmt.Sprintf("could not verify class_id %q: %v", importRow.ClassID, err),
					ErrorType:    imports.ImportFailureSchemaValidation,
				})
				continue
			}
			if !exists {
				failures = append(failures, imports.RowFailure{
					RowNumber:    i,
					RawPayload:   row.RawData,
					ErrorMessage: fmt.Sprintf("class_id %q does not exist or does not belong to this school", importRow.ClassID),
					ErrorType:    imports.ImportFailureInvalidClassReference,
				})
				continue
			}
		}

		// Collect values for duplicate check
		if importRow.AdmissionNumber != nil && *importRow.AdmissionNumber != "" {
			admVals = append(admVals, *importRow.AdmissionNumber)
		}
		if importRow.UPINumber != nil && *importRow.UPINumber != "" {
			upiVals = append(upiVals, *importRow.UPINumber)
		}
		if importRow.KNECAssessmentNumber != nil && *importRow.KNECAssessmentNumber != "" {
			knecVals = append(knecVals, *importRow.KNECAssessmentNumber)
		}

		parsed = append(parsed, parsedRow{
			index:     i,
			row:       importRow,
			rawData:   row.RawData,
			stagingID: row.StagingRowID,
		})
	}

	// ── Phase 2: Check against existing DB records ──
	// Only query if we have any values to check and there are remaining rows.
	var existingAdm, existingUPI, existingKnec map[string]struct{}
	if len(parsed) > 0 && (len(admVals) > 0 || len(upiVals) > 0 || len(knecVals) > 0) {
		existingAdmSlice, existingUPISlice, existingKnecSlice, err := si.repo.CheckExistingFieldValues(
			ctx, tenantID.String(), schoolID.String(),
			admVals, upiVals, knecVals,
		)
		if err != nil {
			// DB error during duplicate check — fail all remaining rows rather
			// than silently allowing duplicates through.
			for _, p := range parsed {
				failures = append(failures, imports.RowFailure{
					RowNumber:    p.index,
					RawPayload:   p.rawData,
					ErrorMessage: fmt.Sprintf("could not verify field uniqueness: %v", err),
					ErrorType:    imports.ImportFailureSchemaValidation,
				})
			}
			return nil, failures
		}

		existingAdm = sliceToSet(existingAdmSlice)
		existingUPI = sliceToSet(existingUPISlice)
		existingKnec = sliceToSet(existingKnecSlice)
	} else {
		existingAdm = make(map[string]struct{})
		existingUPI = make(map[string]struct{})
		existingKnec = make(map[string]struct{})
	}

	// ── Phase 3: Build resolved rows, rejecting duplicates ──
	var resolved []imports.ValidatedRow

	for _, p := range parsed {
		dupType, dupMessage := si.checkFieldDuplicates(p.row, existingAdm, existingUPI, existingKnec)
		if dupType != "" {
			failures = append(failures, imports.RowFailure{
				RowNumber:    p.index,
				RawPayload:   p.rawData,
				ErrorMessage: dupMessage,
				ErrorType:    dupType,
			})
			continue
		}

		aug := augmentedImportRow{
			TenantID:             tenantID.String(),
			SchoolID:             schoolID.String(),
			AcademicTermID:       meta.AcademicTermID,
			AcademicYearID:       meta.AcademicYearID,
			FullName:             p.row.FullName,
			Gender:               p.row.Gender,
			DateOfBirth:          p.row.DateOfBirth,
			UPINumber:            p.row.UPINumber,
			KNECAssessmentNumber: p.row.KNECAssessmentNumber,
			AdmissionNumber:      p.row.AdmissionNumber,
			ClassID:              p.row.ClassID,
			StagingRowID:         p.stagingID.String(),
		}

		augData, err := json.Marshal(aug)
		if err != nil {
			failures = append(failures, imports.RowFailure{
				RowNumber:    p.index,
				RawPayload:   p.rawData,
				ErrorMessage: fmt.Sprintf("marshal augmented row: %v", err),
				ErrorType:    imports.ImportFailureSchemaValidation,
			})
			continue
		}

		resolved = append(resolved, imports.ValidatedRow{
			RawData:      augData,
			StagingRowID: p.stagingID,
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
//
// DB constraint violations are translated from raw Postgres error text into
// friendly, typed errors. Known constraints (e.g., fk_enrollments_tenant_class,
// unique_student_term_enrollment) get specific messages and error types. Any
// unmapped constraint violation falls back to a generic message with error_type
// DB_CONSTRAINT_VIOLATION — raw SQL/driver text is never leaked into the error.
func (si *StudentImporter) InsertOne(ctx context.Context, tx pgx.Tx, row imports.ValidatedRow) error {
	var aug augmentedImportRow
	if err := json.Unmarshal(row.RawData, &aug); err != nil {
		return fmt.Errorf("unmarshal augmented row: %w", err)
	}

	// Step 1: Insert the student record with staging_row_id for idempotency
	// Uses siInsertStudent variable for testability.
	studentID, err := siInsertStudent(si, ctx, tx, aug)
	if err != nil {
		return si.translateConstraintError(err)
	}

	// Step 2: Insert the enrollment only if class_id was provided
	if aug.ClassID != "" {
		// Uses siInsertEnrollment variable for testability.
		if err := siInsertEnrollment(si, ctx, tx, studentID, aug); err != nil {
			return si.translateConstraintError(err)
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

// translateConstraintError inspects a driver error and returns a friendlier,
// typed error. Known Postgres constraint/index violations are mapped to
// specific messages and error types. Any unknown constraint violation gets
// a generic message with DB_CONSTRAINT_VIOLATION. Non-constraint errors
// (e.g., network failures) are passed through unchanged.
func (si *StudentImporter) translateConstraintError(err error) error {
	// Try to extract a Postgres error with a constraint name
	var pgErr *pgconn.PgError
	if !asPgError(err, &pgErr) {
		// Not a Postgres driver error — pass through unchanged
		return err
	}

	// Skip the staging-row conflict which is handled by ON CONFLICT above
	// — if it reaches this code path it's unexpected, but we treat it as a
	// generic constraint violation rather than leaking SQL text.
	if pgErr.ConstraintName == "" {
		return &imports.ImportError{
			Type:    imports.ImportFailureDBConstraintViolation,
			Message: "This record could not be saved due to a data conflict",
		}
	}

	// Check known constraint translations
	if mapping, ok := knownConstraintMessages[pgErr.ConstraintName]; ok {
		return &imports.ImportError{
			Type:    mapping.ErrorType,
			Message: mapping.Message,
		}
	}

	// Unmapped constraint — generic fallback, never leak raw SQL text
	return &imports.ImportError{
		Type:    imports.ImportFailureDBConstraintViolation,
		Message: "This record could not be saved due to a data conflict",
	}
}

// asPgError unwraps err through any layers of fmt.Errorf %w wrapping and
// reports whether the inner error is a *pgconn.PgError. If yes, pgErr is set
// to that value.
func asPgError(err error, pgErr **pgconn.PgError) bool {
	if err == nil {
		return false
	}
	// pgx wraps driver errors with fmt.Errorf; errors.As handles unwrapping.
	var target *pgconn.PgError
	if errors.As(err, &target) {
		*pgErr = target
		return true
	}
	return false
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

// checkFieldDuplicates checks whether any of the row's three tracked field
// values already exist in the DB. Returns the DUPLICATE_* error type and a
// friendly message, or empty string if no duplicate is found.
// If multiple fields match, priority order is: admission_number >
// upi_number > knec_assessment_number (as specified in the task).
func (si *StudentImporter) checkFieldDuplicates(row ImportRow,
	existingAdm, existingUPI, existingKnec map[string]struct{}) (imports.ImportFailureType, string) {

	if row.AdmissionNumber != nil && *row.AdmissionNumber != "" {
		if _, exists := existingAdm[*row.AdmissionNumber]; exists {
			return imports.ImportFailureDuplicateAdmissionNumber,
				fmt.Sprintf("admission number %s already exists for this school", *row.AdmissionNumber)
		}
	}

	if row.UPINumber != nil && *row.UPINumber != "" {
		if _, exists := existingUPI[*row.UPINumber]; exists {
			return imports.ImportFailureDuplicateUPINumber,
				fmt.Sprintf("UPI number %s already exists for this school", *row.UPINumber)
		}
	}

	if row.KNECAssessmentNumber != nil && *row.KNECAssessmentNumber != "" {
		if _, exists := existingKnec[*row.KNECAssessmentNumber]; exists {
			return imports.ImportFailureDuplicateKneCNumber,
				fmt.Sprintf("KNEC assessment number %s already exists for this school", *row.KNECAssessmentNumber)
		}
	}

	return "", ""
}

// sliceToSet converts a string slice to a set (map[string]struct{}) for O(1) lookups.
func sliceToSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	return set
}

// compile-time interface check
var _ imports.Importer = (*StudentImporter)(nil)

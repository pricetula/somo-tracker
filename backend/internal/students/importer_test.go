package students

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// MockImportRepository
// ============================================================================

type MockImportRepository struct {
	validateAcademicTermFn       func(ctx context.Context, tenantID, schoolID, academicTermID string) (bool, error)
	checkSchoolAdminMembershipFn func(ctx context.Context, userID, tenantID, schoolID string) (bool, error)
	getAcademicYearIDForTermFn   func(ctx context.Context, tenantID, schoolID, academicTermID string) (string, error)
	validateClassExistsFn        func(ctx context.Context, tenantID, schoolID, classID string) (bool, error)
	checkExistingFieldValuesFn   func(ctx context.Context, tenantID, schoolID string, admissionNumbers, upiNumbers, knecNumbers []string) ([]string, []string, []string, error)
}

var _ ImportRepository = (*MockImportRepository)(nil)

func (m *MockImportRepository) ValidateAcademicTerm(ctx context.Context, tenantID, schoolID, academicTermID string) (bool, error) {
	if m.validateAcademicTermFn != nil {
		return m.validateAcademicTermFn(ctx, tenantID, schoolID, academicTermID)
	}
	return true, nil
}

func (m *MockImportRepository) CheckSchoolAdminMembership(ctx context.Context, userID, tenantID, schoolID string) (bool, error) {
	if m.checkSchoolAdminMembershipFn != nil {
		return m.checkSchoolAdminMembershipFn(ctx, userID, tenantID, schoolID)
	}
	return true, nil
}

func (m *MockImportRepository) GetAcademicYearIDForTerm(ctx context.Context, tenantID, schoolID, academicTermID string) (string, error) {
	if m.getAcademicYearIDForTermFn != nil {
		return m.getAcademicYearIDForTermFn(ctx, tenantID, schoolID, academicTermID)
	}
	return "year_001", nil
}

func (m *MockImportRepository) ValidateClassExists(ctx context.Context, tenantID, schoolID, classID string) (bool, error) {
	if m.validateClassExistsFn != nil {
		return m.validateClassExistsFn(ctx, tenantID, schoolID, classID)
	}
	return true, nil
}

func (m *MockImportRepository) CheckExistingFieldValues(ctx context.Context, tenantID, schoolID string, admissionNumbers, upiNumbers, knecNumbers []string) ([]string, []string, []string, error) {
	if m.checkExistingFieldValuesFn != nil {
		return m.checkExistingFieldValuesFn(ctx, tenantID, schoolID, admissionNumbers, upiNumbers, knecNumbers)
	}
	// Default: no duplicates exist
	return []string{}, []string{}, []string{}, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	imp  *StudentImporter
	repo *MockImportRepository
}

func newTestHarness() *testHarness {
	repo := &MockImportRepository{}
	imp := NewStudentImporter(repo)
	return &testHarness{
		imp:  imp,
		repo: repo,
	}
}

// ============================================================================
// Tests: JobType
// ============================================================================

func TestStudentImporter_JobType(t *testing.T) {
	imp := NewStudentImporter(&MockImportRepository{})
	if imp.JobType() != imports.ImportJobTypeStudentImport {
		t.Fatalf("expected STUDENT_IMPORT, got %s", imp.JobType())
	}
}

// ============================================================================
// Tests: Validate — Happy Paths
// ============================================================================

func TestValidate_HappyPath(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Alice Wanjiku","gender":"F","class_id":"11111111-1111-1111-1111-111111111111"}`),
		json.RawMessage(`{"full_name":"Bob Kiplagat","gender":"M","class_id":"22222222-2222-2222-2222-222222222222"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(valid) != 2 {
		t.Fatalf("expected 2 valid rows, got %d", len(valid))
	}
}

func TestValidate_WithOptionalFields(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Carol Mwangi","gender":"F","date_of_birth":"2010-05-15","upi_number":"UPI12345","knec_assessment_number":"KNEC67890","admission_number":"ADM001","class_id":"11111111-1111-1111-1111-111111111111"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(valid))
	}
}

func TestValidate_WithoutClassID(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Test Student","gender":"F"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(valid))
	}
}

// ============================================================================
// Tests: Validate — date_of_birth
// ============================================================================

func TestValidate_DateOfBirth_Valid(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		// Use today's date minus ~10 years — well within range
		json.RawMessage(fmt.Sprintf(`{"full_name":"Young Student","gender":"M","date_of_birth":"%s"}`, time.Now().AddDate(-10, 0, 0).Format("2006-01-02"))),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures for valid DOB, got %d: %v", len(failures), failures)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(valid))
	}
}

func TestValidate_DateOfBirth_Future(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(fmt.Sprintf(`{"full_name":"Future Student","gender":"F","date_of_birth":"%s"}`, time.Now().AddDate(0, 0, 1).Format("2006-01-02"))),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows for future DOB, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for future DOB, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION for future DOB, got %s", failures[0].ErrorType)
	}
	if !containsStr(failures[0].ErrorMessage, "future") {
		t.Fatalf("expected error message to mention 'future', got %q", failures[0].ErrorMessage)
	}
}

func TestValidate_DateOfBirth_Unparseable(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Bad Date","gender":"M","date_of_birth":"not-a-date"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows for unparseable DOB, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for unparseable DOB, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION for unparseable DOB, got %s", failures[0].ErrorType)
	}
	if !containsStr(failures[0].ErrorMessage, "not a valid date") {
		t.Fatalf("expected error message to mention 'not a valid date', got %q", failures[0].ErrorMessage)
	}
}

func TestValidate_DateOfBirth_ImplausiblyOld(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	// 100 years ago should exceed maxStudentAgeYears (25)
	raw := []json.RawMessage{
		json.RawMessage(fmt.Sprintf(`{"full_name":"Old Student","gender":"F","date_of_birth":"%s"}`, time.Now().AddDate(-100, 0, 0).Format("2006-01-02"))),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows for implausibly old DOB, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for implausibly old DOB, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION for old DOB, got %s", failures[0].ErrorType)
	}
	if !containsStr(failures[0].ErrorMessage, "implausibly old") {
		t.Fatalf("expected error message to mention 'implausibly old', got %q", failures[0].ErrorMessage)
	}
}

// ============================================================================
// Tests: Validate — class_id UUID check
// ============================================================================

func TestValidate_ClassID_ValidUUID(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"UUID Student","gender":"M","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures for valid UUID class_id, got %d: %v", len(failures), failures)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(valid))
	}
}

func TestValidate_ClassID_Malformed(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Bad Class","gender":"F","class_id":"not-a-uuid-at-all"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows for malformed class_id, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for malformed class_id, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION for malformed class_id, got %s", failures[0].ErrorType)
	}
	if !containsStr(failures[0].ErrorMessage, "not a valid UUID") {
		t.Fatalf("expected error message to mention 'not a valid UUID', got %q", failures[0].ErrorMessage)
	}
}

func TestValidate_ClassID_MalformedNumeric(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	// "class_001" is not a UUID — should fail
	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Legacy Class","gender":"M","class_id":"class_001"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows for malformed class_id, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for malformed class_id, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION for malformed class_id, got %s", failures[0].ErrorType)
	}
}

// ============================================================================
// Tests: Validate — Sad Paths
// ============================================================================

func TestValidate_MissingFullName(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"gender":"M","class_id":"11111111-1111-1111-1111-111111111111"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION, got %s", failures[0].ErrorType)
	}
	if failures[0].ErrorMessage == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestValidate_InvalidGender(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Test Student","gender":"X","class_id":"11111111-1111-1111-1111-111111111111"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureSchemaValidation {
		t.Fatalf("expected SCHEMA_VALIDATION, got %s", failures[0].ErrorType)
	}
}

func TestValidate_OnlyFullNameAndGender(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"full_name":"Jane Doe","gender":"F"}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(valid))
	}
}

func TestValidate_MalformedJSON(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{bad json}`),
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows, got %d", len(valid))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
}

// ============================================================================
// Tests: ResolveReferences — Class existence checks
// ============================================================================

func TestResolveReferences_ClassIDExists(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Alice","gender":"F","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	// Mock: class exists and belongs to this tenant/school
	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		if tid == tenantID.String() && sid == schoolID.String() && cid == "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
			return true, nil
		}
		return false, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures when class exists, got %d: %v", len(failures), failures)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved row, got %d", len(resolved))
	}

	var aug augmentedImportRow
	if err := json.Unmarshal(resolved[0].RawData, &aug); err != nil {
		t.Fatalf("unmarshal resolved row: %v", err)
	}
	if aug.ClassID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Fatalf("expected ClassID to be preserved, got %q", aug.ClassID)
	}
}

func TestResolveReferences_ClassID_DifferentSchool(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Cross Tenant","gender":"M","class_id":"b2c3d4e5-f6a7-8901-bcde-f12345678901"}`)},
	}

	// Mock: class does NOT exist for this tenant/school
	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return false, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(resolved) != 0 {
		t.Fatalf("expected 0 resolved rows for cross-school class, got %d", len(resolved))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for cross-school class, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureInvalidClassReference {
		t.Fatalf("expected INVALID_CLASS_REFERENCE, got %s", failures[0].ErrorType)
	}
	if !containsStr(failures[0].ErrorMessage, "does not exist") {
		t.Fatalf("expected error message to mention 'does not exist', got %q", failures[0].ErrorMessage)
	}
}

// ============================================================================
// Tests: ResolveReferences — Happy Path (existing tests refactored)
// ============================================================================

func TestResolveReferences_WithoutClassID(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"No Class Student","gender":"F"}`)},
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved row, got %d", len(resolved))
	}

	var aug augmentedImportRow
	if err := json.Unmarshal(resolved[0].RawData, &aug); err != nil {
		t.Fatalf("unmarshal resolved row: %v", err)
	}
	if aug.ClassID != "" {
		t.Fatalf("expected empty ClassID for row without class_id, got %q", aug.ClassID)
	}
	if aug.FullName != "No Class Student" {
		t.Fatalf("expected full_name 'No Class Student', got %q", aug.FullName)
	}
	if aug.Gender != "F" {
		t.Fatalf("expected gender 'F', got %q", aug.Gender)
	}
}

func TestResolveReferences_HappyPath(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Alice","gender":"F","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved row, got %d", len(resolved))
	}

	var aug augmentedImportRow
	if err := json.Unmarshal(resolved[0].RawData, &aug); err != nil {
		t.Fatalf("unmarshal resolved row: %v", err)
	}
	if aug.ClassID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Fatalf("expected ClassID 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', got %q", aug.ClassID)
	}
	if aug.AcademicTermID != "term_001" {
		t.Fatalf("expected academic_term_id 'term_001', got %q", aug.AcademicTermID)
	}
	if aug.AcademicYearID != "year_001" {
		t.Fatalf("expected academic_year_id 'year_001', got %q", aug.AcademicYearID)
	}
	if aug.TenantID != tenantID.String() {
		t.Fatalf("expected tenant_id %s, got %q", tenantID.String(), aug.TenantID)
	}
	if aug.SchoolID != schoolID.String() {
		t.Fatalf("expected school_id %s, got %q", schoolID.String(), aug.SchoolID)
	}
	if aug.FullName != "Alice" {
		t.Fatalf("expected full_name 'Alice', got %q", aug.FullName)
	}
}

func TestResolveReferences_MissingMetadata(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Test","gender":"M","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	metadata := json.RawMessage(`{}`)

	resolved, failures := h.imp.ResolveReferences(ctx, uuid.New(), uuid.New(), metadata, rows)
	if len(resolved) != 0 {
		t.Fatalf("expected 0 resolved rows with empty metadata")
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure with empty metadata, got %d", len(failures))
	}
}

// ============================================================================
// Tests: BulkInsert + InsertOne
// ============================================================================

func TestBulkInsert_ReturnsErrorToTriggerSavepoint(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	_, err := h.imp.BulkInsert(ctx, nil, []imports.ValidatedRow{
		{RawData: json.RawMessage(`{}`)},
	})
	if err == nil {
		t.Fatal("expected BulkInsert to return error for student import")
	}
}

func TestInsertOne_RoundTrip(t *testing.T) {
	aug := augmentedImportRow{
		FullName:       "Alice",
		Gender:         "F",
		ClassID:        "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		TenantID:       uuid.New().String(),
		SchoolID:       uuid.New().String(),
		AcademicTermID: "term_001",
		AcademicYearID: "year_001",
	}
	data, err := json.Marshal(aug)
	if err != nil {
		t.Fatalf("marshal augmented row: %v", err)
	}

	var back augmentedImportRow
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if back.FullName != "Alice" {
		t.Fatalf("expected FullName 'Alice', got %q", back.FullName)
	}
	if back.ClassID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Fatalf("expected ClassID 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', got %q", back.ClassID)
	}
}

func TestInsertOne_WithoutEnrollment_RoundTrip(t *testing.T) {
	aug := augmentedImportRow{
		FullName:       "Jane NoClass",
		Gender:         "F",
		TenantID:       uuid.New().String(),
		SchoolID:       uuid.New().String(),
		AcademicTermID: "term_001",
		AcademicYearID: "year_001",
		// ClassID is empty — no enrollment
	}
	data, err := json.Marshal(aug)
	if err != nil {
		t.Fatalf("marshal augmented row without class: %v", err)
	}

	var back augmentedImportRow
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if back.FullName != "Jane NoClass" {
		t.Fatalf("expected FullName 'Jane NoClass', got %q", back.FullName)
	}
	if back.ClassID != "" {
		t.Fatalf("expected empty ClassID, got %q", back.ClassID)
	}
}

// ============================================================================
// Tests: DB Constraint Translation
// ============================================================================

func TestInsertOne_UnmappedConstraintViolation(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	// Build an augmented row with a valid-looking UUID class_id
	classID := uuid.New().String()
	aug := augmentedImportRow{
		FullName:       "Constraint Violation",
		Gender:         "M",
		ClassID:        classID,
		TenantID:       uuid.New().String(),
		SchoolID:       uuid.New().String(),
		AcademicTermID: "term_001",
		AcademicYearID: "year_001",
		StagingRowID:   uuid.New().String(),
	}

	// Spy on translateConstraintError: create a pgconn.PgError with an unknown
	// constraint name to simulate an unmapped constraint violation.
	pgErr := &pgconn.PgError{
		Code:           "23505", // unique_violation
		ConstraintName: "some_unknown_constraint",
		Message:        "duplicate key value violates unique constraint \"some_unknown_constraint\"",
	}
	wrappedErr := fmt.Errorf("insert student: %w", pgErr)

	// Save the original function and restore after test
	origInsertStudent := siInsertStudent
	defer func() {
		siInsertStudent = origInsertStudent
	}()

	// Make insertStudent return the wrapped pgconn error
	siInsertStudent = func(si *StudentImporter, ctx context.Context, tx pgx.Tx, aug augmentedImportRow) (string, error) {
		return "", wrappedErr
	}

	rowData, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: rowData}

	err := h.imp.InsertOne(ctx, nil, row)
	if err == nil {
		t.Fatal("expected error from InsertOne with constraint violation")
	}

	// Verify it's an ImportError with the generic type and message
	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected *imports.ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureDBConstraintViolation {
		t.Fatalf("expected DB_CONSTRAINT_VIOLATION type, got %s", impErr.Type)
	}
	if impErr.Message != "This record could not be saved due to a data conflict" {
		t.Fatalf("expected generic error message, got %q", impErr.Message)
	}
	// Raw SQL/driver text MUST NOT appear in the error message
	if containsStr(impErr.Message, "some_unknown_constraint") || containsStr(impErr.Message, "duplicate key") {
		t.Fatalf("raw driver text leaked into error message: %q", impErr.Message)
	}
}

// TestInsertOne_KnownConstraintFKEnrollments verifies that a known
// fk_enrollments_tenant_class violation is translated to
// INVALID_CLASS_REFERENCE with a friendly message.
func TestInsertOne_KnownConstraintFKEnrollments(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	classID := uuid.New().String()
	aug := augmentedImportRow{
		FullName:       "FK Violation Student",
		Gender:         "F",
		ClassID:        classID,
		TenantID:       uuid.New().String(),
		SchoolID:       uuid.New().String(),
		AcademicTermID: "term_001",
		AcademicYearID: "year_001",
		StagingRowID:   uuid.New().String(),
	}

	pgErr := &pgconn.PgError{
		Code:           "23503", // foreign_key_violation
		ConstraintName: "fk_enrollments_tenant_class",
		Message:        "insert or update on table \"cbc_student_enrollments\" violates foreign key constraint \"fk_enrollments_tenant_class\"",
	}
	wrappedErr := fmt.Errorf("insert enrollment for student %s: %w", uuid.New().String(), pgErr)

	origInsertStudent := siInsertStudent
	defer func() {
		siInsertStudent = origInsertStudent
	}()

	// First insertStudent succeeds, then insertEnrollment fails
	callCount := 0
	siInsertStudent = func(si *StudentImporter, ctx context.Context, tx pgx.Tx, aug augmentedImportRow) (string, error) {
		callCount++
		return uuid.New().String(), nil // student insert succeeds
	}

	// intercept insertEnrollment by replacing the insertEnrollment method
	origInsertEnrollment := siInsertEnrollment
	defer func() {
		siInsertEnrollment = origInsertEnrollment
	}()

	siInsertEnrollment = func(si *StudentImporter, ctx context.Context, tx pgx.Tx, studentID string, aug augmentedImportRow) error {
		return wrappedErr
	}

	rowData, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: rowData}

	err := h.imp.InsertOne(ctx, nil, row)
	if err == nil {
		t.Fatal("expected error from InsertOne with FK violation")
	}

	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected *imports.ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureInvalidClassReference {
		t.Fatalf("expected INVALID_CLASS_REFERENCE for FK constraint, got %s", impErr.Type)
	}
	if !containsStr(impErr.Message, "does not exist") {
		t.Fatalf("expected friendly message about class not existing, got %q", impErr.Message)
	}
}

// TestInsertOne_KnownConstraintDupEnrollment verifies that a
// unique_student_term_enrollment violation is translated to
// BUSINESS_RULE_VIOLATION.
func TestInsertOne_KnownConstraintDupEnrollment(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	classID := uuid.New().String()
	aug := augmentedImportRow{
		FullName:       "Dup Enrollment",
		Gender:         "F",
		ClassID:        classID,
		TenantID:       uuid.New().String(),
		SchoolID:       uuid.New().String(),
		AcademicTermID: "term_001",
		AcademicYearID: "year_001",
		StagingRowID:   uuid.New().String(),
	}

	pgErr := &pgconn.PgError{
		Code:           "23505", // unique_violation
		ConstraintName: "unique_student_term_enrollment",
		Message:        "duplicate key value violates unique constraint \"unique_student_term_enrollment\"",
	}
	wrappedErr := fmt.Errorf("insert enrollment for student %s: %w", uuid.New().String(), pgErr)

	origInsertStudent := siInsertStudent
	defer func() {
		siInsertStudent = origInsertStudent
	}()
	origInsertEnrollment := siInsertEnrollment
	defer func() {
		siInsertEnrollment = origInsertEnrollment
	}()

	siInsertStudent = func(si *StudentImporter, ctx context.Context, tx pgx.Tx, aug augmentedImportRow) (string, error) {
		return uuid.New().String(), nil
	}
	siInsertEnrollment = func(si *StudentImporter, ctx context.Context, tx pgx.Tx, studentID string, aug augmentedImportRow) error {
		return wrappedErr
	}

	rowData, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: rowData}

	err := h.imp.InsertOne(ctx, nil, row)
	if err == nil {
		t.Fatal("expected error from InsertOne with duplicate enrollment")
	}

	var impErr *imports.ImportError
	if !errors.As(err, &impErr) {
		t.Fatalf("expected *imports.ImportError, got %T: %v", err, err)
	}
	if impErr.Type != imports.ImportFailureBusinessRule {
		t.Fatalf("expected BUSINESS_RULE_VIOLATION for duplicate enrollment, got %s", impErr.Type)
	}
	if !containsStr(impErr.Message, "already enrolled") {
		t.Fatalf("expected message about 'already enrolled', got %q", impErr.Message)
	}
}

// TestInsertOne_NonPgError verifies that non-Postgres errors
// (e.g., context errors, network errors) pass through unchanged.
func TestInsertOne_NonPgError(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	aug := augmentedImportRow{
		FullName:       "Network Error Student",
		Gender:         "M",
		TenantID:       uuid.New().String(),
		SchoolID:       uuid.New().String(),
		AcademicTermID: "term_001",
		AcademicYearID: "year_001",
		StagingRowID:   uuid.New().String(),
	}

	origInsertStudent := siInsertStudent
	defer func() {
		siInsertStudent = origInsertStudent
	}()

	siInsertStudent = func(si *StudentImporter, ctx context.Context, tx pgx.Tx, aug augmentedImportRow) (string, error) {
		return "", errors.New("network timeout")
	}

	rowData, _ := json.Marshal(aug)
	row := imports.ValidatedRow{RawData: rowData}

	err := h.imp.InsertOne(ctx, nil, row)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should NOT be an ImportError — it should pass through the original error
	var impErr *imports.ImportError
	if errors.As(err, &impErr) {
		t.Fatal("expected plain error, not ImportError for non-Postgres error")
	}
	if err.Error() != "network timeout" && !containsStr(err.Error(), "network timeout") {
		// The error gets wrapped via insertStudent, so check the end
		if !containsStr(err.Error(), "network timeout") {
			t.Fatalf("expected original error to be preserved, got %q", err.Error())
		}
	}
}

// ============================================================================
// Integration-Style: Full Flow with 2000 Students
// ============================================================================

func TestStudentImporter_ValidateAndResolve2000(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	raw := make([]json.RawMessage, 2000)
	for i := 0; i < 2000; i++ {
		raw[i] = json.RawMessage(`{"full_name":"Student ` + itoa(i) + `","gender":"M","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)
	}

	valid, failures := h.imp.Validate(ctx, tenantID, schoolID, raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 validation failures for 2000 valid rows, got %d: %v", len(failures), failures)
	}
	if len(valid) != 2000 {
		t.Fatalf("expected 2000 valid rows, got %d", len(valid))
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)
	resolved, resolveFailures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, valid)
	if len(resolveFailures) != 0 {
		t.Fatalf("expected 0 resolve failures, got %d: %v", len(resolveFailures), resolveFailures)
	}
	if len(resolved) != 2000 {
		t.Fatalf("expected 2000 resolved rows, got %d", len(resolved))
	}

	for i, row := range resolved {
		var aug augmentedImportRow
		if err := json.Unmarshal(row.RawData, &aug); err != nil {
			t.Fatalf("row %d unmarshal failed: %v", i, err)
		}
		if aug.ClassID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
			t.Fatalf("row %d expected ClassID 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', got %q", i, aug.ClassID)
		}
		if aug.FullName == "" {
			t.Fatalf("row %d has empty full_name", i)
		}
		if aug.Gender != "M" {
			t.Fatalf("row %d expected gender M, got %s", i, aug.Gender)
		}
	}
}

func TestStudentImporter_Validate2000WithSomeFailures(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := make([]json.RawMessage, 2000)
	for i := 0; i < 2000; i++ {
		if i%100 == 0 {
			raw[i] = json.RawMessage(`{"full_name":"Bad Student ` + itoa(i) + `","gender":"X","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)
		} else {
			raw[i] = json.RawMessage(`{"full_name":"Student ` + itoa(i) + `","gender":"M","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)
		}
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	expectedFailures := 2000 / 100
	if len(failures) != expectedFailures {
		t.Fatalf("expected %d validation failures, got %d", expectedFailures, len(failures))
	}
	if len(valid) != 2000-expectedFailures {
		t.Fatalf("expected %d valid rows, got %d", 2000-expectedFailures, len(valid))
	}
}

// ============================================================================
// Tests: Insert-time Duplicate Detection
// ============================================================================

func TestResolveReferences_DuplicateAdmissionNumber(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Dup Adm","gender":"F","admission_number":"ADM001","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}
	h.repo.checkExistingFieldValuesFn = func(_ context.Context, tid, sid string, adm, upi, knec []string) ([]string, []string, []string, error) {
		return []string{"ADM001"}, []string{}, []string{}, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(resolved) != 0 {
		t.Fatalf("expected 0 resolved rows for duplicate admission_number, got %d", len(resolved))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for duplicate admission_number, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureDuplicateAdmissionNumber {
		t.Fatalf("expected DUPLICATE_ADMISSION_NUMBER, got %s", failures[0].ErrorType)
	}
	if !containsStr(failures[0].ErrorMessage, "ADM001") {
		t.Fatalf("expected error message to mention 'ADM001', got %q", failures[0].ErrorMessage)
	}
}

func TestResolveReferences_DuplicateUPINumber(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Dup UPI","gender":"M","upi_number":"UPI999","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}
	h.repo.checkExistingFieldValuesFn = func(_ context.Context, tid, sid string, adm, upi, knec []string) ([]string, []string, []string, error) {
		return []string{}, []string{"UPI999"}, []string{}, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(resolved) != 0 {
		t.Fatalf("expected 0 resolved rows for duplicate UPI, got %d", len(resolved))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for duplicate UPI, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureDuplicateUPINumber {
		t.Fatalf("expected DUPLICATE_UPI_NUMBER, got %s", failures[0].ErrorType)
	}
}

func TestResolveReferences_DuplicateKNECNumber(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Dup KNEC","gender":"F","knec_assessment_number":"KNEC123","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}
	h.repo.checkExistingFieldValuesFn = func(_ context.Context, tid, sid string, adm, upi, knec []string) ([]string, []string, []string, error) {
		return []string{}, []string{}, []string{"KNEC123"}, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(resolved) != 0 {
		t.Fatalf("expected 0 resolved rows for duplicate KNEC, got %d", len(resolved))
	}
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for duplicate KNEC, got %d", len(failures))
	}
	if failures[0].ErrorType != imports.ImportFailureDuplicateKneCNumber {
		t.Fatalf("expected DUPLICATE_KNEC_NUMBER, got %s", failures[0].ErrorType)
	}
}

// TestResolveReferences_SameBatchSharedNumber verifies that two rows in the
// same submission sharing an admission_number are NOT flagged by the backend
// safety net. Within-batch duplicate detection is a frontend responsibility.
func TestResolveReferences_SameBatchSharedNumber_BothSucceed(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	// Both rows share the same admission_number — the backend must NOT flag them
	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Alice","gender":"F","admission_number":"ADM_SHARED","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
		{RawData: json.RawMessage(`{"full_name":"Bob","gender":"M","admission_number":"ADM_SHARED","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}
	h.repo.checkExistingFieldValuesFn = func(_ context.Context, tid, sid string, adm, upi, knec []string) ([]string, []string, []string, error) {
		// Neither value exists in the DB yet
		return []string{}, []string{}, []string{}, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures for same-batch shared number, got %d: %v", len(failures), failures)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved rows, got %d", len(resolved))
	}
}

func TestResolveReferences_NoDuplicatesInBatch(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	// Multiple rows with different values — none exist in DB
	rows := []imports.ValidatedRow{
		{RawData: json.RawMessage(`{"full_name":"Alice","gender":"F","admission_number":"ADM001","upi_number":"UPI001","knec_assessment_number":"KNEC001","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
		{RawData: json.RawMessage(`{"full_name":"Bob","gender":"M","admission_number":"ADM002","upi_number":"UPI002","knec_assessment_number":"KNEC002","class_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`)},
	}

	h.repo.validateClassExistsFn = func(_ context.Context, tid, sid, cid string) (bool, error) {
		return true, nil
	}
	h.repo.checkExistingFieldValuesFn = func(_ context.Context, tid, sid string, adm, upi, knec []string) ([]string, []string, []string, error) {
		return []string{}, []string{}, []string{}, nil
	}

	metadata := json.RawMessage(`{"academic_term_id":"term_001","academic_year_id":"year_001"}`)

	resolved, failures := h.imp.ResolveReferences(ctx, tenantID, schoolID, metadata, rows)
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures, got %d: %v", len(failures), failures)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved rows, got %d", len(resolved))
	}
}

// ============================================================================
// Tests: Edge cases
// ============================================================================

func TestValidate_EmptyRowArray(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), []json.RawMessage{})
	if len(failures) != 0 {
		t.Fatalf("expected 0 failures for empty array, got %d", len(failures))
	}
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows for empty array, got %d", len(valid))
	}
}

func TestValidate_AllRowsFail_ResolveHappy(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := make([]json.RawMessage, 10)
	for i := 0; i < 10; i++ {
		raw[i] = json.RawMessage(`{"full_name":"","gender":"X","class_id":""}`)
	}

	valid, failures := h.imp.Validate(ctx, uuid.New(), uuid.New(), raw)
	if len(valid) != 0 {
		t.Fatalf("expected 0 valid rows, got %d", len(valid))
	}
	if len(failures) != 10 {
		t.Fatalf("expected 10 failures, got %d", len(failures))
	}

	metadata := json.RawMessage(`{"academic_term_id":"t","academic_year_id":"y"}`)
	resolved, resolveFailures := h.imp.ResolveReferences(ctx, uuid.New(), uuid.New(), metadata, valid)
	if len(resolved) != 0 {
		t.Fatalf("expected 0 resolved rows from empty input, got %d", len(resolved))
	}
	if len(resolveFailures) != 0 {
		t.Fatalf("expected 0 resolve failures from empty input, got %d", len(resolveFailures))
	}
}

// ============================================================================
// Tests: Compile-time check
// ============================================================================

func TestStudentImporter_ImplementsImporter(t *testing.T) {
	var _ imports.Importer = (*StudentImporter)(nil)
	t.Log("StudentImporter implements imports.Importer")
}

// ============================================================================
// helpers
// ============================================================================

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

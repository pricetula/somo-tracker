package students

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"somotracker/backend/internal/imports"
)

// ============================================================================
// MockImportRepository
// ============================================================================

type MockImportRepository struct {
	validateAcademicTermFn       func(ctx context.Context, tenantID, schoolID, academicTermID string) (bool, error)
	checkSchoolAdminMembershipFn func(ctx context.Context, userID, tenantID, schoolID string) (bool, error)
	getAcademicYearIDForTermFn   func(ctx context.Context, tenantID, schoolID, academicTermID string) (string, error)
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
		json.RawMessage(`{"full_name":"Alice Wanjiku","gender":"F","class_id":"class_001"}`),
		json.RawMessage(`{"full_name":"Bob Kiplagat","gender":"M","class_id":"class_001"}`),
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
		json.RawMessage(`{"full_name":"Carol Mwangi","gender":"F","date_of_birth":"2010-05-15","upi_number":"UPI12345","knec_assessment_number":"KNEC67890","admission_number":"ADM001","class_id":"class_001"}`),
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
// Tests: Validate — Sad Paths
// ============================================================================

func TestValidate_MissingFullName(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	raw := []json.RawMessage{
		json.RawMessage(`{"gender":"M","class_id":"class_001"}`),
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
		json.RawMessage(`{"full_name":"Test Student","gender":"X","class_id":"class_001"}`),
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
// Tests: ResolveReferences — Happy Path
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
		{RawData: json.RawMessage(`{"full_name":"Alice","gender":"F","class_id":"class_001"}`)},
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
	if aug.ClassID != "class_001" {
		t.Fatalf("expected ClassID 'class_001', got %q", aug.ClassID)
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
		{RawData: json.RawMessage(`{"full_name":"Test","gender":"M","class_id":"class_001"}`)},
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
		ClassID:        "class_001",
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
	if back.ClassID != "class_001" {
		t.Fatalf("expected ClassID 'class_001', got %q", back.ClassID)
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
// Integration-Style: Full Flow with 2000 Students
// ============================================================================

func TestStudentImporter_ValidateAndResolve2000(t *testing.T) {
	h := newTestHarness()
	ctx := context.Background()

	tenantID := uuid.New()
	schoolID := uuid.New()

	raw := make([]json.RawMessage, 2000)
	for i := 0; i < 2000; i++ {
		raw[i] = json.RawMessage(`{"full_name":"Student ` + itoa(i) + `","gender":"M","class_id":"class_G4_Blue"}`)
	}

	valid, failures := h.imp.Validate(ctx, tenantID, schoolID, raw)
	if len(failures) != 0 {
		t.Fatalf("expected 0 validation failures for 2000 valid rows, got %d: %v", len(failures), failures)
	}
	if len(valid) != 2000 {
		t.Fatalf("expected 2000 valid rows, got %d", len(valid))
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
		if aug.ClassID != "class_G4_Blue" {
			t.Fatalf("row %d expected ClassID 'class_G4_Blue', got %q", i, aug.ClassID)
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
			raw[i] = json.RawMessage(`{"full_name":"Bad Student ` + itoa(i) + `","gender":"X","class_id":"class_001"}`)
		} else {
			raw[i] = json.RawMessage(`{"full_name":"Student ` + itoa(i) + `","gender":"M","class_id":"class_001"}`)
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

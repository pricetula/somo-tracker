package portfolio

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	createEntryFn                            func(ctx context.Context, e *PortfolioEntry) (string, error)
	getEntryByIDFn                           func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error)
	listEntriesFn                            func(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error)
	updateEntryFn                            func(ctx context.Context, e *PortfolioEntry) error
	deleteEntryFn                            func(ctx context.Context, id, tenantID string) error
	entryExistsForStudentSubStrandEvidenceFn func(ctx context.Context, tenantID, studentID, subStrandID, evidenceType string) (bool, error)
}

func (m *MockRepository) CreateEntry(ctx context.Context, e *PortfolioEntry) (string, error) {
	if m.createEntryFn != nil {
		return m.createEntryFn(ctx, e)
	}
	return "entry_001", nil
}

func (m *MockRepository) GetEntryByID(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
	if m.getEntryByIDFn != nil {
		return m.getEntryByIDFn(ctx, id, tenantID)
	}
	return &PortfolioEntry{
		ID: id, TenantID: tenantID,
		StudentID: "student_001", SubStrandID: "substrand_001",
		EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
	}, nil
}

func (m *MockRepository) ListEntries(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error) {
	if m.listEntriesFn != nil {
		return m.listEntriesFn(ctx, tenantID, query)
	}
	return []PortfolioEntry{}, nil
}

func (m *MockRepository) UpdateEntry(ctx context.Context, e *PortfolioEntry) error {
	if m.updateEntryFn != nil {
		return m.updateEntryFn(ctx, e)
	}
	return nil
}

func (m *MockRepository) DeleteEntry(ctx context.Context, id, tenantID string) error {
	if m.deleteEntryFn != nil {
		return m.deleteEntryFn(ctx, id, tenantID)
	}
	return nil
}

func (m *MockRepository) EntryExistsForStudentSubStrandEvidence(ctx context.Context, tenantID, studentID, subStrandID, evidenceType string) (bool, error) {
	if m.entryExistsForStudentSubStrandEvidenceFn != nil {
		return m.entryExistsForStudentSubStrandEvidenceFn(ctx, tenantID, studentID, subStrandID, evidenceType)
	}
	return false, nil
}

// ============================================================================
// MockStudentResolver
// ============================================================================

type MockStudentResolver struct {
	studentExistsFn func(ctx context.Context, tenantID, studentID string) (bool, error)
}

func (m *MockStudentResolver) StudentExists(ctx context.Context, tenantID, studentID string) (bool, error) {
	if m.studentExistsFn != nil {
		return m.studentExistsFn(ctx, tenantID, studentID)
	}
	return true, nil
}

// ============================================================================
// MockSubStrandResolver
// ============================================================================

type MockSubStrandResolver struct {
	subStrandExistsFn func(ctx context.Context, subStrandID string) (bool, error)
}

func (m *MockSubStrandResolver) SubStrandExists(ctx context.Context, subStrandID string) (bool, error) {
	if m.subStrandExistsFn != nil {
		return m.subStrandExistsFn(ctx, subStrandID)
	}
	return true, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc               *Service
	repo              *MockRepository
	studentResolver   *MockStudentResolver
	subStrandResolver *MockSubStrandResolver
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	studentResolver := &MockStudentResolver{}
	subStrandResolver := &MockSubStrandResolver{}
	svc := NewService(repo, studentResolver, subStrandResolver)
	return &testHarness{
		svc:               svc,
		repo:              repo,
		studentResolver:   studentResolver,
		subStrandResolver: subStrandResolver,
	}
}

// ============================================================================
// Suite A — Create Portfolio Entry
// ============================================================================

// A1 — Create entry returns ID
func TestCreateEntry_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.createEntryFn = func(ctx context.Context, e *PortfolioEntry) (string, error) {
		if e.StudentID != "student_001" {
			t.Errorf("expected student_id 'student_001', got %q", e.StudentID)
		}
		if e.SubStrandID != "substrand_001" {
			t.Errorf("expected sub_strand_id 'substrand_001', got %q", e.SubStrandID)
		}
		if e.EvidenceType != "Physical_File_Reference" {
			t.Errorf("expected evidence_type 'Physical_File_Reference', got %q", e.EvidenceType)
		}
		if e.StoragePointer != "Binder 2B, page 14" {
			t.Errorf("expected storage_pointer 'Binder 2B, page 14', got %q", e.StoragePointer)
		}
		return "entry_001", nil
	}

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	entry, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != "entry_001" {
		t.Errorf("expected id 'entry_001', got %q", entry.ID)
	}
	if entry.StudentID != payload.StudentID {
		t.Errorf("expected student_id %q, got %q", payload.StudentID, entry.StudentID)
	}
	if entry.EvidenceType != payload.EvidenceType {
		t.Errorf("expected evidence_type %q, got %q", payload.EvidenceType, entry.EvidenceType)
	}
}

// A2 — Create entry with all optional fields
func TestCreateEntry_WithOptionals_Success(t *testing.T) {
	h := newTestHarness()

	dateCollected := "2026-06-15"
	linkedResultID := "result_001"

	h.repo.createEntryFn = func(ctx context.Context, e *PortfolioEntry) (string, error) {
		if e.DateCollected == nil || *e.DateCollected != "2026-06-15" {
			t.Errorf("expected date_collected '2026-06-15', got %v", e.DateCollected)
		}
		if e.LinkedResultID == nil || *e.LinkedResultID != "result_001" {
			t.Errorf("expected linked_result_id 'result_001', got %v", e.LinkedResultID)
		}
		return "entry_002", nil
	}

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Digital_Artifact_URL",
		StoragePointer: "https://storage.example.com/artifact.pdf",
		LinkedResultID: &linkedResultID,
		DateCollected:  &dateCollected,
	}

	entry, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != "entry_002" {
		t.Errorf("expected id 'entry_002', got %q", entry.ID)
	}
}

// A3 — Create entry with empty storage_pointer → ErrInvalidInput
func TestCreateEntry_EmptyStoragePointer(t *testing.T) {
	h := newTestHarness()

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "", // invalid
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for empty storage_pointer, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// A4 — Create entry with invalid evidence_type → ErrInvalidEvidenceType
func TestCreateEntry_InvalidEvidenceType(t *testing.T) {
	h := newTestHarness()

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Invalid_Type", // invalid
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for invalid evidence_type, got nil")
	}
	if !errors.Is(err, ErrInvalidEvidenceType) {
		t.Fatalf("expected ErrInvalidEvidenceType, got %v", err)
	}
}

// A5 — Create entry with non-existent student → ErrStudentNotFound
func TestCreateEntry_StudentNotFound(t *testing.T) {
	h := newTestHarness()

	h.studentResolver.studentExistsFn = func(ctx context.Context, tenantID, studentID string) (bool, error) {
		return false, nil
	}

	payload := CreatePortfolioEntryPayload{
		StudentID:      "nonexistent_student",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for non-existent student, got nil")
	}
	if !errors.Is(err, ErrStudentNotFound) {
		t.Fatalf("expected ErrStudentNotFound, got %v", err)
	}
}

// A6 — Create entry with non-existent sub_strand → ErrSubStrandNotFound
func TestCreateEntry_SubStrandNotFound(t *testing.T) {
	h := newTestHarness()

	h.subStrandResolver.subStrandExistsFn = func(ctx context.Context, subStrandID string) (bool, error) {
		return false, nil
	}

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "nonexistent_substrand",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for non-existent sub_strand, got nil")
	}
	if !errors.Is(err, ErrSubStrandNotFound) {
		t.Fatalf("expected ErrSubStrandNotFound, got %v", err)
	}
}

// A7 — Create entry with invalid date_collected format → ErrInvalidInput
func TestCreateEntry_InvalidDateCollected(t *testing.T) {
	h := newTestHarness()

	badDate := "15-06-2026" // wrong format

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
		DateCollected:  &badDate,
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for invalid date format, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// A8 — Create entry with duplicate (student, sub_strand, evidence_type) → ErrDuplicateAdvised
func TestCreateEntry_DuplicateAdvised(t *testing.T) {
	h := newTestHarness()

	h.repo.entryExistsForStudentSubStrandEvidenceFn = func(ctx context.Context, tenantID, studentID, subStrandID, evidenceType string) (bool, error) {
		return true, nil
	}

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for duplicate entry, got nil")
	}
	if !errors.Is(err, ErrDuplicateAdvised) {
		t.Fatalf("expected ErrDuplicateAdvised, got %v", err)
	}
}

// A9 — Create entry with missing student_id → ErrInvalidInput
func TestCreateEntry_MissingStudentID(t *testing.T) {
	h := newTestHarness()

	payload := CreatePortfolioEntryPayload{
		StudentID:      "",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for missing student_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// A10 — Create entry with missing sub_strand_id → ErrInvalidInput
func TestCreateEntry_MissingSubStrandID(t *testing.T) {
	h := newTestHarness()

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for missing sub_strand_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Suite B — List Portfolio Entries
// ============================================================================

// B1 — List entries returns all entries for tenant
func TestListEntries_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.listEntriesFn = func(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenant_id 'tenant_001', got %q", tenantID)
		}
		if query.StudentID != "" {
			t.Errorf("expected student_id empty, got %q", query.StudentID)
		}
		return []PortfolioEntry{
			{ID: "entry_001", TenantID: tenantID, StudentID: "student_001", SubStrandID: "substrand_001", EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14"},
			{ID: "entry_002", TenantID: tenantID, StudentID: "student_002", SubStrandID: "substrand_001", EvidenceType: "Video_Recording", StoragePointer: "https://videos.example.com/lesson1.mp4"},
		}, nil
	}

	entries, err := h.svc.ListEntries(context.Background(), "tenant_001", ListEntriesQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

// B2 — List entries filtered by student_id
func TestListEntries_FilterByStudent(t *testing.T) {
	h := newTestHarness()

	filterStudentID := "student_001"
	called := false

	h.repo.listEntriesFn = func(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error) {
		if query.StudentID != filterStudentID {
			t.Errorf("expected student_id %q, got %q", filterStudentID, query.StudentID)
		}
		called = true
		return []PortfolioEntry{
			{ID: "entry_001", TenantID: tenantID, StudentID: filterStudentID},
		}, nil
	}

	entries, err := h.svc.ListEntries(context.Background(), "tenant_001", ListEntriesQuery{StudentID: filterStudentID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected ListEntries to be called with student_id filter")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

// B3 — List entries filtered by sub_strand_id
func TestListEntries_FilterBySubStrand(t *testing.T) {
	h := newTestHarness()

	filterSubStrandID := "substrand_002"
	called := false

	h.repo.listEntriesFn = func(ctx context.Context, tenantID string, query ListEntriesQuery) ([]PortfolioEntry, error) {
		if query.SubStrandID != filterSubStrandID {
			t.Errorf("expected sub_strand_id %q, got %q", filterSubStrandID, query.SubStrandID)
		}
		called = true
		return []PortfolioEntry{
			{ID: "entry_003", TenantID: tenantID, SubStrandID: filterSubStrandID},
		}, nil
	}

	entries, err := h.svc.ListEntries(context.Background(), "tenant_001", ListEntriesQuery{SubStrandID: filterSubStrandID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected ListEntries to be called with sub_strand_id filter")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

// ============================================================================
// Suite C — Get Portfolio Entry
// ============================================================================

// C1 — Get entry by ID returns the entry
func TestGetEntry_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		if id != "entry_001" {
			t.Errorf("expected id 'entry_001', got %q", id)
		}
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Audio_Log", StoragePointer: "recording.mp3",
		}, nil
	}

	entry, err := h.svc.GetEntry(context.Background(), "entry_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != "entry_001" {
		t.Errorf("expected id 'entry_001', got %q", entry.ID)
	}
	if entry.EvidenceType != "Audio_Log" {
		t.Errorf("expected evidence_type 'Audio_Log', got %q", entry.EvidenceType)
	}
}

// C2 — Get entry with empty ID → ErrInvalidInput
func TestGetEntry_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetEntry(context.Background(), "", "tenant_001")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Suite D — Update Portfolio Entry
// ============================================================================

// D1 — Update storage_pointer → 200
func TestUpdateEntry_StoragePointer(t *testing.T) {
	h := newTestHarness()

	newPointer := "Updated Binder 3A, page 5"
	updated := false

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
		}, nil
	}

	h.repo.updateEntryFn = func(ctx context.Context, e *PortfolioEntry) error {
		if e.StoragePointer != newPointer {
			t.Errorf("expected storage_pointer %q, got %q", newPointer, e.StoragePointer)
		}
		updated = true
		return nil
	}

	payload := UpdatePortfolioEntryPayload{
		StoragePointer: &newPointer,
	}

	err := h.svc.UpdateEntry(context.Background(), "entry_001", "tenant_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected UpdateEntry to be called")
	}
}

// D2 — Link rubric result to entry
func TestUpdateEntry_LinkResult(t *testing.T) {
	h := newTestHarness()

	linkedResultID := "result_002"
	updated := false

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
		}, nil
	}

	h.repo.updateEntryFn = func(ctx context.Context, e *PortfolioEntry) error {
		if e.LinkedResultID == nil || *e.LinkedResultID != "result_002" {
			t.Errorf("expected linked_result_id 'result_002', got %v", e.LinkedResultID)
		}
		updated = true
		return nil
	}

	payload := UpdatePortfolioEntryPayload{
		LinkedResultID: &linkedResultID,
	}

	err := h.svc.UpdateEntry(context.Background(), "entry_001", "tenant_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected UpdateEntry to be called")
	}
}

// D3 — Unlink rubric result from entry (set to empty string)
func TestUpdateEntry_UnlinkResult(t *testing.T) {
	h := newTestHarness()

	linkedResultID := "result_001"
	unlinked := false

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
			LinkedResultID: &linkedResultID,
		}, nil
	}

	h.repo.updateEntryFn = func(ctx context.Context, e *PortfolioEntry) error {
		if e.LinkedResultID != nil {
			t.Errorf("expected linked_result_id to be nil (unlinked), got %v", e.LinkedResultID)
		}
		unlinked = true
		return nil
	}

	emptyStr := ""
	payload := UpdatePortfolioEntryPayload{
		LinkedResultID: &emptyStr,
	}

	err := h.svc.UpdateEntry(context.Background(), "entry_001", "tenant_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unlinked {
		t.Fatal("expected UpdateEntry to be called")
	}
}

// D4 — Update entry not found → ErrNotFound
func TestUpdateEntry_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		return nil, ErrNotFound
	}

	newPointer := "Binder 3A"
	payload := UpdatePortfolioEntryPayload{
		StoragePointer: &newPointer,
	}

	err := h.svc.UpdateEntry(context.Background(), "nonexistent", "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for non-existent entry, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// D5 — Update with empty storage_pointer → ErrInvalidInput
func TestUpdateEntry_EmptyStoragePointer(t *testing.T) {
	h := newTestHarness()

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
		}, nil
	}

	emptyStr := ""
	payload := UpdatePortfolioEntryPayload{
		StoragePointer: &emptyStr,
	}

	err := h.svc.UpdateEntry(context.Background(), "entry_001", "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for empty storage_pointer, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// D6 — Update with invalid date_collected → ErrInvalidInput
func TestUpdateEntry_InvalidDateCollected(t *testing.T) {
	h := newTestHarness()

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
		}, nil
	}

	badDate := "not-a-date"
	payload := UpdatePortfolioEntryPayload{
		DateCollected: &badDate,
	}

	err := h.svc.UpdateEntry(context.Background(), "entry_001", "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for invalid date_collected, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Suite E — Delete Portfolio Entry
// ============================================================================

// E1 — Delete entry → 204
func TestDeleteEntry_Success(t *testing.T) {
	h := newTestHarness()

	deleted := false
	h.repo.deleteEntryFn = func(ctx context.Context, id, tenantID string) error {
		deleted = true
		return nil
	}

	err := h.svc.DeleteEntry(context.Background(), "entry_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected DeleteEntry to be called")
	}
}

// E2 — Delete entry not found → ErrNotFound
func TestDeleteEntry_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteEntryFn = func(ctx context.Context, id, tenantID string) error {
		return ErrNotFound
	}

	err := h.svc.DeleteEntry(context.Background(), "nonexistent", "tenant_001")
	if err == nil {
		t.Fatal("expected error for non-existent entry, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Suite F — Tenant Isolation
// ============================================================================

// F1 — Cross-tenant access denied (entry not found for wrong tenant)
func TestGetEntry_CrossTenantIsolation(t *testing.T) {
	h := newTestHarness()

	h.repo.getEntryByIDFn = func(ctx context.Context, id, tenantID string) (*PortfolioEntry, error) {
		// Simulate that entry exists only for tenant_001, not tenant_002
		if tenantID != "tenant_001" {
			return nil, ErrNotFound
		}
		return &PortfolioEntry{
			ID: id, TenantID: tenantID,
			StudentID: "student_001", SubStrandID: "substrand_001",
			EvidenceType: "Physical_File_Reference", StoragePointer: "Binder 2B, page 14",
		}, nil
	}

	// Should succeed for correct tenant
	_, err := h.svc.GetEntry(context.Background(), "entry_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error for correct tenant: %v", err)
	}

	// Should fail for wrong tenant
	_, err = h.svc.GetEntry(context.Background(), "entry_001", "tenant_002")
	if err == nil {
		t.Fatal("expected error for cross-tenant access, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant, got %v", err)
	}
}

// ============================================================================
// Suite G — Validation: Edge Cases
// ============================================================================

// G1 — All valid evidence types are accepted
func TestCreateEntry_AllValidEvidenceTypes(t *testing.T) {
	validTypes := []string{
		"Physical_File_Reference",
		"Digital_Artifact_URL",
		"Video_Recording",
		"Audio_Log",
		"Observation_Checklist",
	}

	for _, et := range validTypes {
		h := newTestHarness()

		h.repo.createEntryFn = func(ctx context.Context, e *PortfolioEntry) (string, error) {
			return "entry_" + et, nil
		}

		payload := CreatePortfolioEntryPayload{
			StudentID:      "student_001",
			SubStrandID:    "substrand_001",
			EvidenceType:   et,
			StoragePointer: "some location",
		}

		_, err := h.svc.CreateEntry(context.Background(), "tenant_001", payload)
		if err != nil {
			t.Errorf("unexpected error for evidence_type %q: %v", et, err)
		}
	}
}

// G2 — Tenant ID is required for CreateEntry
func TestCreateEntry_EmptyTenant(t *testing.T) {
	h := newTestHarness()

	payload := CreatePortfolioEntryPayload{
		StudentID:      "student_001",
		SubStrandID:    "substrand_001",
		EvidenceType:   "Physical_File_Reference",
		StoragePointer: "Binder 2B, page 14",
	}

	_, err := h.svc.CreateEntry(context.Background(), "", payload)
	if err == nil {
		t.Fatal("expected error for empty tenant, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

package assessment

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	createBlueprintFn                func(ctx context.Context, bp *AssessmentBlueprint) (string, error)
	getBlueprintByIDFn               func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error)
	listBlueprintsFn                 func(ctx context.Context, tenantID string, query ListBlueprintsQuery) ([]AssessmentBlueprint, error)
	updateBlueprintFn                func(ctx context.Context, bp *AssessmentBlueprint) error
	deleteBlueprintFn                func(ctx context.Context, id, tenantID, schoolID string) error
	getBlueprintDetailFn             func(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error)
	linkIndicatorsFn                 func(ctx context.Context, blueprintID string, indicatorIDs []string) error
	unlinkIndicatorFn                func(ctx context.Context, blueprintID, indicatorID string) error
	isIndicatorLinkedFn              func(ctx context.Context, blueprintID, indicatorID string) (bool, error)
	listBlueprintIndicatorsFn        func(ctx context.Context, blueprintID string) ([]LinkedIndicator, error)
	listWeightConfigsFn              func(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error)
	createSessionFn                  func(ctx context.Context, s *AssessmentSession) (string, error)
	getSessionByIDFn                 func(ctx context.Context, id, tenantID string) (*AssessmentSession, error)
	listSessionsFn                   func(ctx context.Context, tenantID string, query ListSessionsQuery) ([]AssessmentSession, int, error)
	updateSessionFn                  func(ctx context.Context, s *AssessmentSession) error
	deleteSessionFn                  func(ctx context.Context, id, tenantID string) error
	getSessionDetailFn               func(ctx context.Context, id, tenantID string) (*SessionDetail, error)
	batchUpsertResultsFn             func(ctx context.Context, sessionID, tenantID string, results []LearnerRubricResult) (int, error)
	listResultsFn                    func(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error)
	submitForReviewFn                func(ctx context.Context, sessionID, tenantID, submittedAt string) error
	approveSessionFn                 func(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt string) error
	rejectSessionFn                  func(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt, rejectionReason string) error
	publishSessionFn                 func(ctx context.Context, sessionID, tenantID, publisherUserID, publishedAt string) error
	publishSessionsFn                func(ctx context.Context, sessionIDs []string, tenantID, publisherUserID, publishedAt string) (int, error)
	listSessionsForAdminQueueFn      func(ctx context.Context, tenantID string, query AdminQueuesQuery) ([]SessionAdminView, int, error)
	listPublishedResultsForStudentFn func(ctx context.Context, tenantID string, query ParentSessionsQuery) ([]ParentSessionResultView, int, error)
}

func (m *MockRepository) CreateBlueprint(ctx context.Context, bp *AssessmentBlueprint) (string, error) {
	if m.createBlueprintFn != nil {
		return m.createBlueprintFn(ctx, bp)
	}
	return "bp_001", nil
}

func (m *MockRepository) GetBlueprintByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
	if m.getBlueprintByIDFn != nil {
		return m.getBlueprintByIDFn(ctx, id, tenantID, schoolID)
	}
	return &AssessmentBlueprint{ID: id, TenantID: tenantID, SchoolID: schoolID, GradeLevel: "G7", Term: 1, AcademicYear: 2026, Title: "Test Blueprint", Type: "Formative_Classroom"}, nil
}

func (m *MockRepository) ListBlueprints(ctx context.Context, tenantID string, query ListBlueprintsQuery) ([]AssessmentBlueprint, error) {
	if m.listBlueprintsFn != nil {
		return m.listBlueprintsFn(ctx, tenantID, query)
	}
	return []AssessmentBlueprint{}, nil
}

func (m *MockRepository) UpdateBlueprint(ctx context.Context, bp *AssessmentBlueprint) error {
	if m.updateBlueprintFn != nil {
		return m.updateBlueprintFn(ctx, bp)
	}
	return nil
}

func (m *MockRepository) DeleteBlueprint(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteBlueprintFn != nil {
		return m.deleteBlueprintFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) GetBlueprintDetail(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error) {
	if m.getBlueprintDetailFn != nil {
		return m.getBlueprintDetailFn(ctx, id, tenantID, schoolID)
	}
	return &BlueprintDetail{
		AssessmentBlueprint: AssessmentBlueprint{ID: id, TenantID: tenantID, SchoolID: schoolID},
		Indicators:          []LinkedIndicator{},
	}, nil
}

func (m *MockRepository) LinkIndicators(ctx context.Context, blueprintID string, indicatorIDs []string) error {
	if m.linkIndicatorsFn != nil {
		return m.linkIndicatorsFn(ctx, blueprintID, indicatorIDs)
	}
	return nil
}

func (m *MockRepository) UnlinkIndicator(ctx context.Context, blueprintID, indicatorID string) error {
	if m.unlinkIndicatorFn != nil {
		return m.unlinkIndicatorFn(ctx, blueprintID, indicatorID)
	}
	return nil
}

func (m *MockRepository) IsIndicatorLinked(ctx context.Context, blueprintID, indicatorID string) (bool, error) {
	if m.isIndicatorLinkedFn != nil {
		return m.isIndicatorLinkedFn(ctx, blueprintID, indicatorID)
	}
	return false, nil
}

func (m *MockRepository) ListWeightConfigs(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error) {
	if m.listWeightConfigsFn != nil {
		return m.listWeightConfigsFn(ctx, query)
	}
	return []AssessmentWeightConfig{}, nil
}

func (m *MockRepository) ListBlueprintIndicators(ctx context.Context, blueprintID string) ([]LinkedIndicator, error) {
	if m.listBlueprintIndicatorsFn != nil {
		return m.listBlueprintIndicatorsFn(ctx, blueprintID)
	}
	return []LinkedIndicator{}, nil
}

func (m *MockRepository) CreateSession(ctx context.Context, s *AssessmentSession) (string, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, s)
	}
	return "session_001", nil
}

func (m *MockRepository) GetSessionByID(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
	if m.getSessionByIDFn != nil {
		return m.getSessionByIDFn(ctx, id, tenantID)
	}
	return &AssessmentSession{
		ID: id, TenantID: tenantID,
		BlueprintID: "bp_001", ClassID: "class_001",
		DateAdministered: "2026-06-01",
	}, nil
}

func (m *MockRepository) ListSessions(ctx context.Context, tenantID string, query ListSessionsQuery) ([]AssessmentSession, int, error) {
	if m.listSessionsFn != nil {
		return m.listSessionsFn(ctx, tenantID, query)
	}
	return []AssessmentSession{}, 0, nil
}

func (m *MockRepository) UpdateSession(ctx context.Context, s *AssessmentSession) error {
	if m.updateSessionFn != nil {
		return m.updateSessionFn(ctx, s)
	}
	return nil
}

func (m *MockRepository) DeleteSession(ctx context.Context, id, tenantID string) error {
	if m.deleteSessionFn != nil {
		return m.deleteSessionFn(ctx, id, tenantID)
	}
	return nil
}

func (m *MockRepository) GetSessionDetail(ctx context.Context, id, tenantID string) (*SessionDetail, error) {
	if m.getSessionDetailFn != nil {
		return m.getSessionDetailFn(ctx, id, tenantID)
	}
	return &SessionDetail{
		AssessmentSession: AssessmentSession{ID: id, TenantID: tenantID},
		Results:           []LearnerRubricResult{},
	}, nil
}

func (m *MockRepository) BatchUpsertResults(ctx context.Context, sessionID, tenantID string, results []LearnerRubricResult) (int, error) {
	if m.batchUpsertResultsFn != nil {
		return m.batchUpsertResultsFn(ctx, sessionID, tenantID, results)
	}
	return len(results), nil
}

func (m *MockRepository) ListResults(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
	if m.listResultsFn != nil {
		return m.listResultsFn(ctx, sessionID, tenantID)
	}
	return []LearnerRubricResult{}, nil
}

func (m *MockRepository) SubmitForReview(ctx context.Context, sessionID, tenantID, submittedAt string) error {
	if m.submitForReviewFn != nil {
		return m.submitForReviewFn(ctx, sessionID, tenantID, submittedAt)
	}
	return nil
}

func (m *MockRepository) ApproveSession(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt string) error {
	if m.approveSessionFn != nil {
		return m.approveSessionFn(ctx, sessionID, tenantID, reviewerUserID, reviewedAt)
	}
	return nil
}

func (m *MockRepository) RejectSession(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt, rejectionReason string) error {
	if m.rejectSessionFn != nil {
		return m.rejectSessionFn(ctx, sessionID, tenantID, reviewerUserID, reviewedAt, rejectionReason)
	}
	return nil
}

func (m *MockRepository) PublishSession(ctx context.Context, sessionID, tenantID, publisherUserID, publishedAt string) error {
	if m.publishSessionFn != nil {
		return m.publishSessionFn(ctx, sessionID, tenantID, publisherUserID, publishedAt)
	}
	return nil
}

func (m *MockRepository) PublishSessions(ctx context.Context, sessionIDs []string, tenantID, publisherUserID, publishedAt string) (int, error) {
	if m.publishSessionsFn != nil {
		return m.publishSessionsFn(ctx, sessionIDs, tenantID, publisherUserID, publishedAt)
	}
	return len(sessionIDs), nil
}

func (m *MockRepository) ListSessionsForAdminQueue(ctx context.Context, tenantID string, query AdminQueuesQuery) ([]SessionAdminView, int, error) {
	if m.listSessionsForAdminQueueFn != nil {
		return m.listSessionsForAdminQueueFn(ctx, tenantID, query)
	}
	return []SessionAdminView{}, 0, nil
}

func (m *MockRepository) ListPublishedResultsForStudent(ctx context.Context, tenantID string, query ParentSessionsQuery) ([]ParentSessionResultView, int, error) {
	if m.listPublishedResultsForStudentFn != nil {
		return m.listPublishedResultsForStudentFn(ctx, tenantID, query)
	}
	return []ParentSessionResultView{}, 0, nil
}

// ============================================================================
// MockLearningAreaResolver
// ============================================================================

type MockLearningAreaResolver struct {
	getPerformanceIndicatorEducationLevelFn func(ctx context.Context, indicatorID string) (string, error)
}

func (m *MockLearningAreaResolver) GetPerformanceIndicatorEducationLevel(ctx context.Context, indicatorID string) (string, error) {
	if m.getPerformanceIndicatorEducationLevelFn != nil {
		return m.getPerformanceIndicatorEducationLevelFn(ctx, indicatorID)
	}
	return "Junior_Secondary", nil
}

// ============================================================================
// MockClassStudentResolver
// ============================================================================

type MockClassStudentResolver struct {
	isStudentInClassFn func(ctx context.Context, studentID, classID string) (bool, error)
}

func (m *MockClassStudentResolver) IsStudentInClass(ctx context.Context, studentID, classID string) (bool, error) {
	if m.isStudentInClassFn != nil {
		return m.isStudentInClassFn(ctx, studentID, classID)
	}
	return true, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc        *Service
	repo       *MockRepository
	laResolver *MockLearningAreaResolver
	csResolver *MockClassStudentResolver
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	laResolver := &MockLearningAreaResolver{}
	csResolver := &MockClassStudentResolver{}
	svc := NewService(repo, laResolver, csResolver)
	return &testHarness{
		svc:        svc,
		repo:       repo,
		laResolver: laResolver,
		csResolver: csResolver,
	}
}

// ============================================================================
// Suite A — Blueprint CRUD
// ============================================================================

// A1 — Create blueprint returns ID
func TestCreateBlueprint_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.createBlueprintFn = func(ctx context.Context, bp *AssessmentBlueprint) (string, error) {
		if bp.Term != 1 {
			t.Errorf("expected term 1, got %d", bp.Term)
		}
		return "bp_001", nil
	}

	payload := CreateBlueprintPayload{
		Title:        "Term 1 Formative Assessment",
		Type:         "Formative_Classroom",
		GradeLevel:   "G7",
		AcademicYear: 2026,
		Term:         1,
	}

	bp, err := h.svc.CreateBlueprint(context.Background(), "tenant_001", "school_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bp.ID != "bp_001" {
		t.Errorf("expected id 'bp_001', got %q", bp.ID)
	}
	if bp.Title != payload.Title {
		t.Errorf("expected title %q, got %q", payload.Title, bp.Title)
	}
}

// A2 — Create blueprint with invalid term (outside 1-3) → ErrInvalidInput
func TestCreateBlueprint_InvalidTerm(t *testing.T) {
	h := newTestHarness()

	payload := CreateBlueprintPayload{
		Title:        "Bad Term Blueprint",
		Type:         "Formative_Classroom",
		GradeLevel:   "G7",
		AcademicYear: 2026,
		Term:         4, // invalid
	}

	_, err := h.svc.CreateBlueprint(context.Background(), "tenant_001", "school_001", payload)
	if err == nil {
		t.Fatal("expected error for invalid term, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// A3 — Create blueprint with academic_year < 2017 → ErrInvalidInput
func TestCreateBlueprint_InvalidAcademicYear(t *testing.T) {
	h := newTestHarness()

	payload := CreateBlueprintPayload{
		Title:        "Old Blueprint",
		Type:         "Formative_Classroom",
		GradeLevel:   "G7",
		AcademicYear: 2016, // invalid
		Term:         1,
	}

	_, err := h.svc.CreateBlueprint(context.Background(), "tenant_001", "school_001", payload)
	if err == nil {
		t.Fatal("expected error for old academic year, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// A4 — Create duplicate blueprint → ErrAlreadyExists
func TestCreateBlueprint_Duplicate(t *testing.T) {
	h := newTestHarness()

	h.repo.createBlueprintFn = func(ctx context.Context, bp *AssessmentBlueprint) (string, error) {
		return "", errors.New("duplicate key value violates unique constraint \"uq_blueprint_per_school_grade_term\"")
	}

	payload := CreateBlueprintPayload{
		Title:        "Term 1 Formative Assessment",
		Type:         "Formative_Classroom",
		GradeLevel:   "G7",
		AcademicYear: 2026,
		Term:         1,
	}

	_, err := h.svc.CreateBlueprint(context.Background(), "tenant_001", "school_001", payload)
	if err == nil {
		t.Fatal("expected ErrAlreadyExists for duplicate, got nil")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

// A5 — List blueprints filtered by school_id, grade_level, term
func TestListBlueprints_Filtered(t *testing.T) {
	h := newTestHarness()

	var capturedQuery ListBlueprintsQuery
	h.repo.listBlueprintsFn = func(ctx context.Context, tenantID string, query ListBlueprintsQuery) ([]AssessmentBlueprint, error) {
		capturedQuery = query
		return []AssessmentBlueprint{
			{ID: "bp_001", Title: "T1 Formative", GradeLevel: "G7", Term: 1, AcademicYear: 2026},
		}, nil
	}

	query := ListBlueprintsQuery{
		SchoolID:     "school_001",
		GradeLevel:   "G7",
		Term:         1,
		AcademicYear: 2026,
	}

	blueprints, err := h.svc.ListBlueprints(context.Background(), "tenant_001", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blueprints) != 1 {
		t.Fatalf("expected 1 blueprint, got %d", len(blueprints))
	}
	if capturedQuery.SchoolID != "school_001" {
		t.Errorf("expected school filter 'school_001', got %q", capturedQuery.SchoolID)
	}
	if capturedQuery.GradeLevel != "G7" {
		t.Errorf("expected grade_level 'G7', got %q", capturedQuery.GradeLevel)
	}
	if capturedQuery.Term != 1 {
		t.Errorf("expected term 1, got %d", capturedQuery.Term)
	}
}

// A6 — Get blueprint detail with linked indicators
func TestGetBlueprintDetail_WithIndicators(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error) {
		return &BlueprintDetail{
			AssessmentBlueprint: AssessmentBlueprint{
				ID: id, TenantID: tenantID, SchoolID: schoolID,
				Title: "Test Blueprint", Type: "Formative_Classroom",
				GradeLevel: "G7", AcademicYear: 2026, Term: 1,
			},
			Indicators: []LinkedIndicator{
				{ID: "pi_001", Description: "Identify parts of speech"},
				{ID: "pi_002", Description: "Construct simple sentences"},
			},
		}, nil
	}

	detail, err := h.svc.GetBlueprintDetail(context.Background(), "bp_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detail.Indicators) != 2 {
		t.Fatalf("expected 2 indicators, got %d", len(detail.Indicators))
	}
	if detail.Indicators[0].ID != "pi_001" {
		t.Errorf("expected first indicator 'pi_001', got %q", detail.Indicators[0].ID)
	}
}

// A7 — Update blueprint title/type
func TestUpdateBlueprint_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
		return &AssessmentBlueprint{
			ID: id, TenantID: tenantID, SchoolID: schoolID,
			Title: "Old Title", Type: "Formative_Classroom",
			GradeLevel: "G7", AcademicYear: 2026, Term: 1,
		}, nil
	}

	var updatedTitle string
	h.repo.updateBlueprintFn = func(ctx context.Context, bp *AssessmentBlueprint) error {
		updatedTitle = bp.Title
		return nil
	}

	newTitle := "Updated Title"
	payload := UpdateBlueprintPayload{Title: &newTitle}

	err := h.svc.UpdateBlueprint(context.Background(), "bp_001", "tenant_001", "school_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedTitle != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", updatedTitle)
	}
}

// A8 — Get non-existent blueprint → ErrNotFound
func TestGetBlueprint_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetBlueprintDetail(context.Background(), "nonexistent", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for non-existent blueprint, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// A9 — Delete blueprint → 204 (no error)
func TestDeleteBlueprint_Success(t *testing.T) {
	h := newTestHarness()

	var deletedID string
	h.repo.deleteBlueprintFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		deletedID = id
		return nil
	}

	err := h.svc.DeleteBlueprint(context.Background(), "bp_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "bp_001" {
		t.Errorf("expected deleted ID 'bp_001', got %q", deletedID)
	}
}

// A10 — Delete non-existent blueprint → ErrNotFound
func TestDeleteBlueprint_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteBlueprintFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	err := h.svc.DeleteBlueprint(context.Background(), "nonexistent", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for non-existent, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Suite B — Blueprint ↔ Indicator Linking
// ============================================================================

// B1 — Link indicators to blueprint → success
func TestLinkIndicators_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
		return &AssessmentBlueprint{
			ID: id, TenantID: tenantID, SchoolID: schoolID,
			GradeLevel: "G7", // maps to Junior_Secondary
		}, nil
	}

	h.repo.isIndicatorLinkedFn = func(ctx context.Context, blueprintID, indicatorID string) (bool, error) {
		return false, nil
	}

	h.laResolver.getPerformanceIndicatorEducationLevelFn = func(ctx context.Context, indicatorID string) (string, error) {
		return "Junior_Secondary", nil // matches G7
	}

	var linkedIDs []string
	h.repo.linkIndicatorsFn = func(ctx context.Context, blueprintID string, indicatorIDs []string) error {
		linkedIDs = indicatorIDs
		return nil
	}

	payload := LinkIndicatorPayload{
		IndicatorIDs: []string{"pi_001", "pi_002"},
	}

	err := h.svc.LinkIndicators(context.Background(), "bp_001", "tenant_001", "school_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(linkedIDs) != 2 {
		t.Fatalf("expected 2 linked indicators, got %d", len(linkedIDs))
	}
}

// B2 — Link indicator from wrong grade level → ErrGradeLevelMismatch
func TestLinkIndicators_GradeLevelMismatch(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
		return &AssessmentBlueprint{
			ID: id, TenantID: tenantID, SchoolID: schoolID,
			GradeLevel: "G7", // Junior_Secondary
		}, nil
	}

	h.repo.isIndicatorLinkedFn = func(ctx context.Context, blueprintID, indicatorID string) (bool, error) {
		return false, nil
	}

	// Indicator belongs to Upper_Primary, not Junior_Secondary
	h.laResolver.getPerformanceIndicatorEducationLevelFn = func(ctx context.Context, indicatorID string) (string, error) {
		return "Upper_Primary", nil
	}

	payload := LinkIndicatorPayload{
		IndicatorIDs: []string{"pi_001"},
	}

	err := h.svc.LinkIndicators(context.Background(), "bp_001", "tenant_001", "school_001", payload)
	if err == nil {
		t.Fatal("expected grade level mismatch error, got nil")
	}
	if !errors.Is(err, ErrGradeLevelMismatch) {
		t.Fatalf("expected ErrGradeLevelMismatch, got %v", err)
	}
}

// B3 — Link already-linked indicator → ErrIndicatorLinked
func TestLinkIndicators_AlreadyLinked(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
		return &AssessmentBlueprint{
			ID: id, TenantID: tenantID, SchoolID: schoolID,
			GradeLevel: "G7",
		}, nil
	}

	h.repo.isIndicatorLinkedFn = func(ctx context.Context, blueprintID, indicatorID string) (bool, error) {
		return true, nil // already linked
	}

	payload := LinkIndicatorPayload{
		IndicatorIDs: []string{"pi_001"},
	}

	err := h.svc.LinkIndicators(context.Background(), "bp_001", "tenant_001", "school_001", payload)
	if err == nil {
		t.Fatal("expected already-linked error, got nil")
	}
	if !errors.Is(err, ErrIndicatorLinked) {
		t.Fatalf("expected ErrIndicatorLinked, got %v", err)
	}
}

// B4 — Unlink indicator from blueprint → success
func TestUnlinkIndicator_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
		return &AssessmentBlueprint{ID: id, TenantID: tenantID, SchoolID: schoolID}, nil
	}

	var unlinkedBlueprintID, unlinkedIndicatorID string
	h.repo.unlinkIndicatorFn = func(ctx context.Context, blueprintID, indicatorID string) error {
		unlinkedBlueprintID = blueprintID
		unlinkedIndicatorID = indicatorID
		return nil
	}

	err := h.svc.UnlinkIndicator(context.Background(), "bp_001", "pi_001", "tenant_001", "school_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unlinkedBlueprintID != "bp_001" {
		t.Errorf("expected blueprint 'bp_001', got %q", unlinkedBlueprintID)
	}
	if unlinkedIndicatorID != "pi_001" {
		t.Errorf("expected indicator 'pi_001', got %q", unlinkedIndicatorID)
	}
}

// B5 — Unlink non-existent indicator-link → ErrNotFound
func TestUnlinkIndicator_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintByIDFn = func(ctx context.Context, id, tenantID, schoolID string) (*AssessmentBlueprint, error) {
		return &AssessmentBlueprint{ID: id, TenantID: tenantID, SchoolID: schoolID}, nil
	}

	h.repo.unlinkIndicatorFn = func(ctx context.Context, blueprintID, indicatorID string) error {
		return ErrNotFound
	}

	err := h.svc.UnlinkIndicator(context.Background(), "bp_001", "nonexistent", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Suite C — Weight Configs
// ============================================================================

// C1 — List weight configs filtered by grade_level
func TestListWeightConfigs_Filtered(t *testing.T) {
	h := newTestHarness()

	var capturedQuery ListWeightConfigsQuery
	h.repo.listWeightConfigsFn = func(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error) {
		capturedQuery = query
		return []AssessmentWeightConfig{
			{
				ID:                 "wc_001",
				GradeLevel:         "G6",
				AssessmentTypeCode: "National_KPSEA",
				TargetExam:         "KPSEA",
				WeightPercent:      "40.00",
				EffectiveFrom:      2026,
			},
		}, nil
	}

	query := ListWeightConfigsQuery{
		GradeLevel: "G6",
		TargetExam: "KPSEA",
	}

	configs, err := h.svc.ListWeightConfigs(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if capturedQuery.GradeLevel != "G6" {
		t.Errorf("expected grade_level 'G6', got %q", capturedQuery.GradeLevel)
	}
	if capturedQuery.TargetExam != "KPSEA" {
		t.Errorf("expected target_exam 'KPSEA', got %q", capturedQuery.TargetExam)
	}
	if configs[0].WeightPercent != "40.00" {
		t.Errorf("expected weight '40.00', got %q", configs[0].WeightPercent)
	}
}

// C2 — List weight configs with no filters returns all
func TestListWeightConfigs_NoFilter(t *testing.T) {
	h := newTestHarness()

	h.repo.listWeightConfigsFn = func(ctx context.Context, query ListWeightConfigsQuery) ([]AssessmentWeightConfig, error) {
		return []AssessmentWeightConfig{
			{ID: "wc_001", GradeLevel: "G6", AssessmentTypeCode: "National_KPSEA", TargetExam: "KPSEA", WeightPercent: "40.00", EffectiveFrom: 2026},
			{ID: "wc_002", GradeLevel: "G6", AssessmentTypeCode: "KNEC_SBA_Project", TargetExam: "KPSEA", WeightPercent: "60.00", EffectiveFrom: 2026},
		}, nil
	}

	configs, err := h.svc.ListWeightConfigs(context.Background(), ListWeightConfigsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
}

// ============================================================================
// Suite D — Tenant Isolation & Edge Cases
// ============================================================================

// D1 — Empty tenant ID returns ErrInvalidInput
func TestCreateBlueprint_EmptyTenant(t *testing.T) {
	h := newTestHarness()

	payload := CreateBlueprintPayload{
		Title: "Test", Type: "Formative_Classroom",
		GradeLevel: "G7", AcademicYear: 2026, Term: 1,
	}

	_, err := h.svc.CreateBlueprint(context.Background(), "", "school_001", payload)
	if err == nil {
		t.Fatal("expected error for empty tenant, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// D2 — Cross-tenant access returns ErrNotFound (simulated by repository)
func TestGetBlueprint_CrossTenant(t *testing.T) {
	h := newTestHarness()

	h.repo.getBlueprintDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*BlueprintDetail, error) {
		// Cross-tenant: blueprint exists but belongs to different tenant
		return nil, ErrNotFound
	}

	_, err := h.svc.GetBlueprintDetail(context.Background(), "bp_001", "wrong_tenant", "school_001")
	if err == nil {
		t.Fatal("expected ErrNotFound for cross-tenant access, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// D3 — Delete blueprint that has assessment sessions → ErrConflict
func TestDeleteBlueprint_Referenced(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteBlueprintFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrConflict
	}

	err := h.svc.DeleteBlueprint(context.Background(), "bp_001", "tenant_001", "school_001")
	if err == nil {
		t.Fatal("expected error for referenced blueprint, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

// ============================================================================
// Suite E — Session Status Transitions (State Machine)
// ============================================================================

// E1 — Submit for review from DRAFT succeeds with results
func TestSubmitForReview_FromDraft_Success(t *testing.T) {
	h := newTestHarness()

	// Fix the time for deterministic testing
	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "DRAFT", AssessedByUserID: "user_001",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	// Session has results
	h.repo.listResultsFn = func(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
		return []LearnerRubricResult{
			{ID: "result_001"},
			{ID: "result_002"},
		}, nil
	}

	var submitted bool
	h.repo.submitForReviewFn = func(ctx context.Context, sessionID, tenantID, submittedAt string) error {
		submitted = true
		if submittedAt != "2026-07-15T10:00:00Z" {
			t.Errorf("expected submittedAt '2026-07-15T10:00:00Z', got %q", submittedAt)
		}
		return nil
	}

	result, err := h.svc.SubmitForReview(context.Background(), "session_001", "tenant_001", "user_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !submitted {
		t.Error("expected SubmitForReview to be called")
	}
	if result.Status != "PENDING_REVIEW" {
		t.Errorf("expected status PENDING_REVIEW, got %q", result.Status)
	}
	if result.SubmittedAt == nil || *result.SubmittedAt != "2026-07-15T10:00:00Z" {
		t.Errorf("expected submitted_at to be set")
	}
}

// E2 — Submit for review from REJECTED succeeds
func TestSubmitForReview_FromRejected_Success(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	reason := "Missing scores for indicator X"
	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "REJECTED", AssessedByUserID: "user_001",
		RejectionReason: &reason,
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	h.repo.listResultsFn = func(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
		return []LearnerRubricResult{{ID: "result_001"}}, nil
	}

	var submitted bool
	h.repo.submitForReviewFn = func(ctx context.Context, sessionID, tenantID, submittedAt string) error {
		submitted = true
		return nil
	}

	result, err := h.svc.SubmitForReview(context.Background(), "session_001", "tenant_001", "user_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !submitted {
		t.Error("expected SubmitForReview to be called")
	}
	if result.Status != "PENDING_REVIEW" {
		t.Errorf("expected status PENDING_REVIEW, got %q", result.Status)
	}
}

// E3 — Submit for review with no results fails
func TestSubmitForReview_NoResults(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "DRAFT", AssessedByUserID: "user_001",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	// Empty results
	h.repo.listResultsFn = func(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
		return []LearnerRubricResult{}, nil
	}

	_, err := h.svc.SubmitForReview(context.Background(), "session_001", "tenant_001", "user_001")
	if err == nil {
		t.Fatal("expected error for session with no results, got nil")
	}
	if !errors.Is(err, ErrNoResultsToSubmit) {
		t.Fatalf("expected ErrNoResultsToSubmit, got %v", err)
	}
}

// E4 — Non-owner cannot submit for review
func TestSubmitForReview_NotOwner(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	// Session owned by different user
	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "DRAFT", AssessedByUserID: "other_teacher",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	h.repo.listResultsFn = func(ctx context.Context, sessionID, tenantID string) ([]LearnerRubricResult, error) {
		return []LearnerRubricResult{{ID: "result_001"}}, nil
	}

	_, err := h.svc.SubmitForReview(context.Background(), "session_001", "tenant_001", "user_001")
	if err == nil {
		t.Fatal("expected error for non-owner, got nil")
	}
	if !errors.Is(err, ErrNotSessionOwner) {
		t.Fatalf("expected ErrNotSessionOwner, got %v", err)
	}
}

// E5 — Submit from invalid status (APPROVED) fails
func TestSubmitForReview_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "APPROVED", AssessedByUserID: "user_001",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	_, err := h.svc.SubmitForReview(context.Background(), "session_001", "tenant_001", "user_001")
	if err == nil {
		t.Fatal("expected error for invalid status transition, got nil")
	}
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

// E6 — Approve session from PENDING_REVIEW succeeds
func TestApproveSession_Success(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "PENDING_REVIEW",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	var approved bool
	h.repo.approveSessionFn = func(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt string) error {
		approved = true
		if reviewerUserID != "admin_001" {
			t.Errorf("expected reviewer 'admin_001', got %q", reviewerUserID)
		}
		return nil
	}

	result, err := h.svc.ApproveSession(context.Background(), "session_001", "tenant_001", "admin_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected ApproveSession to be called")
	}
	if result.Status != "APPROVED" {
		t.Errorf("expected status APPROVED, got %q", result.Status)
	}
	if result.ReviewedByUserID == nil || *result.ReviewedByUserID != "admin_001" {
		t.Errorf("expected reviewed_by to be set")
	}
}

// E7 — Approve from non-PENDING_REVIEW fails
func TestApproveSession_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "DRAFT",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	_, err := h.svc.ApproveSession(context.Background(), "session_001", "tenant_001", "admin_001")
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

// E8 — Reject session with reason succeeds
func TestRejectSession_Success(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "PENDING_REVIEW",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	var rejected bool
	h.repo.rejectSessionFn = func(ctx context.Context, sessionID, tenantID, reviewerUserID, reviewedAt, rejectionReason string) error {
		rejected = true
		if rejectionReason != "Incorrect rubric levels applied" {
			t.Errorf("expected reason 'Incorrect rubric levels applied', got %q", rejectionReason)
		}
		return nil
	}

	result, err := h.svc.RejectSession(context.Background(), "session_001", "tenant_001", "admin_001", "Incorrect rubric levels applied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rejected {
		t.Error("expected RejectSession to be called")
	}
	if result.Status != "REJECTED" {
		t.Errorf("expected status REJECTED, got %q", result.Status)
	}
	if result.RejectionReason == nil || *result.RejectionReason != "Incorrect rubric levels applied" {
		t.Errorf("expected rejection_reason to be set")
	}
}

// E9 — Reject without reason fails
func TestRejectSession_NoReason(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	_, err := h.svc.RejectSession(context.Background(), "session_001", "tenant_001", "admin_001", "")
	if err == nil {
		t.Fatal("expected error for empty reason, got nil")
	}
	if !errors.Is(err, ErrRejectionReasonRequired) {
		t.Fatalf("expected ErrRejectionReasonRequired, got %v", err)
	}
}

// E10 — Publish session from APPROVED succeeds
func TestPublishSession_Success(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "APPROVED",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	var published bool
	h.repo.publishSessionFn = func(ctx context.Context, sessionID, tenantID, publisherUserID, publishedAt string) error {
		published = true
		if publisherUserID != "admin_001" {
			t.Errorf("expected publisher 'admin_001', got %q", publisherUserID)
		}
		return nil
	}

	result, err := h.svc.PublishSession(context.Background(), "session_001", "tenant_001", "admin_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !published {
		t.Error("expected PublishSession to be called")
	}
	if result.Status != "PUBLISHED" {
		t.Errorf("expected status PUBLISHED, got %q", result.Status)
	}
	if result.PublishedByUserID == nil || *result.PublishedByUserID != "admin_001" {
		t.Errorf("expected published_by to be set")
	}
}

// E11 — Publish from non-APPROVED fails
func TestPublishSession_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "PENDING_REVIEW",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	_, err := h.svc.PublishSession(context.Background(), "session_001", "tenant_001", "admin_001")
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

// E12 — Batch publish sessions succeeds
func TestPublishSessions_Success(t *testing.T) {
	h := newTestHarness()

	nowFunc = func() string { return "2026-07-15T10:00:00Z" }

	h.repo.publishSessionsFn = func(ctx context.Context, sessionIDs []string, tenantID, publisherUserID, publishedAt string) (int, error) {
		if len(sessionIDs) != 2 {
			t.Errorf("expected 2 session IDs, got %d", len(sessionIDs))
		}
		return 2, nil
	}

	result, err := h.svc.PublishSessions(context.Background(), []string{"s1", "s2"}, "tenant_001", "admin_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PublishedCount != 2 {
		t.Errorf("expected 2 published, got %d", result.PublishedCount)
	}
	if len(result.FailedIDs) != 0 {
		t.Errorf("expected 0 failed, got %d", len(result.FailedIDs))
	}
}

// E13 — Batch publish with empty list fails
func TestPublishSessions_Empty(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.PublishSessions(context.Background(), []string{}, "tenant_001", "admin_001")
	if err == nil {
		t.Fatal("expected error for empty session list, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// E14 — BatchUpsertResults blocked when session is not DRAFT or REJECTED
func TestBatchUpsertResults_SessionLocked(t *testing.T) {
	h := newTestHarness()

	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "PENDING_REVIEW", BlueprintID: "bp_001",
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	payload := BatchUpsertResultsPayload{
		Results: []BatchUpsertResultInput{
			{StudentID: "stu_001", IndicatorID: "pi_001", ScoreType: "Rubric_Direct", RubricLevel: "ME"},
		},
	}

	_, err := h.svc.BatchUpsertResults(context.Background(), "session_001", "tenant_001", payload)
	if err == nil {
		t.Fatal("expected error for locked session, got nil")
	}
	if !errors.Is(err, ErrSessionLocked) {
		t.Fatalf("expected ErrSessionLocked, got %v", err)
	}
}

// E15 — BatchUpsertResults allowed on REJECTED session
func TestBatchUpsertResults_RejectedSessionAllowed(t *testing.T) {
	h := newTestHarness()

	reason := "Fix scores"
	session := &AssessmentSession{
		ID: "session_001", TenantID: "tenant_001",
		Status: "REJECTED", BlueprintID: "bp_001", ClassID: "class_001",
		RejectionReason: &reason,
	}
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AssessmentSession, error) {
		return session, nil
	}

	h.repo.listBlueprintIndicatorsFn = func(ctx context.Context, blueprintID string) ([]LinkedIndicator, error) {
		return []LinkedIndicator{{ID: "pi_001", Description: "Test indicator"}}, nil
	}

	h.csResolver.isStudentInClassFn = func(ctx context.Context, studentID, classID string) (bool, error) {
		return true, nil
	}

	payload := BatchUpsertResultsPayload{
		Results: []BatchUpsertResultInput{
			{StudentID: "stu_001", IndicatorID: "pi_001", ScoreType: "Rubric_Direct", RubricLevel: "ME"},
		},
	}

	count, err := h.svc.BatchUpsertResults(context.Background(), "session_001", "tenant_001", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 affected row, got %d", count)
	}
}

// E16 — Allowed transitions map is correct
func TestAllowedTransitions_Complete(t *testing.T) {
	// Verify the state machine covers all valid statuses
	for status := range ValidSessionStatuses {
		_, ok := AllowedTransitions[status]
		if !ok {
			t.Errorf("AllowedTransitions missing entry for status %q", status)
		}
	}

	// Verify terminal states
	if len(AllowedTransitions["PUBLISHED"]) != 0 {
		t.Error("PUBLISHED should be a terminal state with no outgoing transitions")
	}

	// Verify no illegal transitions exist
	if AllowedTransitions["DRAFT"]["APPROVED"] {
		t.Error("DRAFT should not be able to skip to APPROVED")
	}
	if AllowedTransitions["DRAFT"]["PUBLISHED"] {
		t.Error("DRAFT should not be able to skip to PUBLISHED")
	}
	if AllowedTransitions["PENDING_REVIEW"]["PUBLISHED"] {
		t.Error("PENDING_REVIEW should not be able to skip to PUBLISHED")
	}
	if AllowedTransitions["REJECTED"]["APPROVED"] {
		t.Error("REJECTED should not be able to skip to APPROVED")
	}
	if AllowedTransitions["APPROVED"]["PENDING_REVIEW"] {
		t.Error("APPROVED should not go back to PENDING_REVIEW")
	}
}

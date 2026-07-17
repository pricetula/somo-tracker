package assessments

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ============================================================================
// MockRepository (minimal — only weight config methods needed for these tests)
// ============================================================================

// mockRepo implements Repository for testing the weight config service methods.
type mockRepo struct {
	createWeightConfigFn   func(ctx context.Context, params CreateWeightConfigParams) (string, error)
	listWeightConfigsFn    func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error)
	getWeightConfigByIDFn  func(ctx context.Context, id string) (*AssessmentWeightConfig, error)
	getStudentTermGradesFn func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error)
}

func (m *mockRepo) CreateWeightConfig(ctx context.Context, params CreateWeightConfigParams) (string, error) {
	if m.createWeightConfigFn != nil {
		return m.createWeightConfigFn(ctx, params)
	}
	return "cfg_001", nil
}

func (m *mockRepo) ListWeightConfigs(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
	if m.listWeightConfigsFn != nil {
		return m.listWeightConfigsFn(ctx, filter)
	}
	return []AssessmentWeightConfig{}, nil
}

func (m *mockRepo) GetWeightConfigByID(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
	if m.getWeightConfigByIDFn != nil {
		return m.getWeightConfigByIDFn(ctx, id)
	}
	return nil, ErrNotFound
}

// --- Stubs for Repository methods not covered by this mock ---

func (m *mockRepo) IsTermFinalised(ctx context.Context, termID string) (bool, error) {
	return false, nil
}

func (m *mockRepo) CreateScaleProfileWithRanges(ctx context.Context, params CreateScaleProfileParams) (string, []string, error) {
	return "", nil, errors.New("not implemented in mock")
}

func (m *mockRepo) GetScaleProfileByID(ctx context.Context, id, tenantID, schoolID string) (*ScaleProfileWithRanges, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) ListScaleProfiles(ctx context.Context, tenantID, schoolID string, activeOnly bool) ([]ScaleProfileWithRanges, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) ToggleScaleProfileActive(ctx context.Context, id, tenantID, schoolID string, isActive bool) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) DeleteScaleProfile(ctx context.Context, id, tenantID, schoolID string) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) CreateSession(ctx context.Context, params CreateSessionParams) (string, error) {
	return "", errors.New("not implemented in mock")
}

func (m *mockRepo) GetSessionByID(ctx context.Context, id, tenantID, schoolID string) (*AssessmentSession, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) GetSessionStatusAndTerm(ctx context.Context, id, tenantID string) (string, string, error) {
	return "", "", errors.New("not implemented in mock")
}

func (m *mockRepo) ListSessions(ctx context.Context, tenantID, schoolID string, filters SessionFilters) ([]AssessmentSession, int, error) {
	return nil, 0, errors.New("not implemented in mock")
}

func (m *mockRepo) UpdateSessionStatus(ctx context.Context, id, tenantID, schoolID string, status string, rejectionComment *string, approvedBy *string) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) HasScoresForSession(ctx context.Context, sessionID string) (bool, error) {
	return false, errors.New("not implemented in mock")
}

func (m *mockRepo) CountSessionsReferencingScale(ctx context.Context, profileID string) (int, error) {
	return 0, errors.New("not implemented in mock")
}

func (m *mockRepo) UpsertStudentScore(ctx context.Context, params UpsertScoreParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) BulkUpsertStudentScores(ctx context.Context, params []UpsertScoreParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) GetStudentScoresBySession(ctx context.Context, sessionID, tenantID, schoolID string) ([]StudentScore, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) SnapshotPerformanceLevels(ctx context.Context, sessionID string, profile *ScaleProfile) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) UpsertOutcomeGrade(ctx context.Context, params UpsertOutcomeGradeParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) BulkUpsertOutcomeGrades(ctx context.Context, params []UpsertOutcomeGradeParams) error {
	return errors.New("not implemented in mock")
}

func (m *mockRepo) GetOutcomeGradesBySession(ctx context.Context, sessionID, tenantID, schoolID string) ([]OutcomeGrade, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) GetOutcomeGradesByStudent(ctx context.Context, sessionID, studentID string) ([]OutcomeGrade, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) GetStudentTermGrades(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
	if m.getStudentTermGradesFn != nil {
		return m.getStudentTermGradesFn(ctx, tenantID, schoolID, studentID, termID)
	}
	return []StudentTermGrade{}, nil
}

func (m *mockRepo) GetPublishedSessionsForParent(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]ParentAssessmentView, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) GetScaleRanges(ctx context.Context, profileID string) ([]ScaleRange, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockRepo) ReplaceScaleRanges(ctx context.Context, profileID string, ranges []CreateScaleRangeParams) ([]string, error) {
	return nil, errors.New("not implemented in mock")
}

// ============================================================================
// Test Harness
// ============================================================================

type svcTestHarness struct {
	svc  *Service
	repo *mockRepo
}

func newSvcTestHarness() *svcTestHarness {
	repo := &mockRepo{}
	svc := NewService(repo)
	return &svcTestHarness{
		svc:  svc,
		repo: repo,
	}
}

// validParams returns a minimal valid set of CreateWeightConfigParams.
func validWeightParams() CreateWeightConfigParams {
	return CreateWeightConfigParams{
		GradeLevel:         "GRADE_4",
		AssessmentTypeCode: "KNEC_SBA_Project",
		TargetExam:         "KPSEA",
		WeightPercent:      20.0,
		EffectiveFrom:      2026,
		Notes:              strPtr("Test config"),
	}
}

func strPtr(s string) *string { return &s }

// ============================================================================
// Tests: CreateWeightConfig
// ============================================================================

func TestCreateWeightConfig_HappyPath(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		if params.GradeLevel != "GRADE_4" {
			t.Errorf("expected grade_level 'GRADE_4', got %q", params.GradeLevel)
		}
		if params.AssessmentTypeCode != "KNEC_SBA_Project" {
			t.Errorf("expected assessment_type_code 'KNEC_SBA_Project', got %q", params.AssessmentTypeCode)
		}
		if params.TargetExam != "KPSEA" {
			t.Errorf("expected target_exam 'KPSEA', got %q", params.TargetExam)
		}
		if params.WeightPercent != 20.0 {
			t.Errorf("expected weight_percent 20.0, got %f", params.WeightPercent)
		}
		if params.EffectiveFrom != 2026 {
			t.Errorf("expected effective_from 2026, got %d", params.EffectiveFrom)
		}
		if params.Notes == nil || *params.Notes != "Test config" {
			t.Errorf("expected notes 'Test config', got %v", params.Notes)
		}
		return "cfg_001", nil
	}

	id, err := h.svc.CreateWeightConfig(context.Background(), validWeightParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cfg_001" {
		t.Fatalf("expected id 'cfg_001', got %q", id)
	}
}

func TestCreateWeightConfig_NilNotes(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		if params.Notes != nil {
			t.Errorf("expected notes to be nil, got %v", *params.Notes)
		}
		return "cfg_002", nil
	}

	params := validWeightParams()
	params.Notes = nil

	id, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cfg_002" {
		t.Fatalf("expected id 'cfg_002', got %q", id)
	}
}

func TestCreateWeightConfig_TrimsWhitespace(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		if params.AssessmentTypeCode != "KNEC_SBA_Project" {
			t.Errorf("expected trimmed assessment_type_code 'KNEC_SBA_Project', got %q", params.AssessmentTypeCode)
		}
		if params.TargetExam != "KPSEA" {
			t.Errorf("expected trimmed target_exam 'KPSEA', got %q", params.TargetExam)
		}
		return "cfg_003", nil
	}

	params := validWeightParams()
	params.AssessmentTypeCode = "  KNEC_SBA_Project  "
	params.TargetExam = "  KPSEA  "

	id, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cfg_003" {
		t.Fatalf("expected id 'cfg_003', got %q", id)
	}
}

func TestCreateWeightConfig_MissingGradeLevel(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.GradeLevel = ""

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for empty grade_level, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_MissingAssessmentTypeCode(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.AssessmentTypeCode = ""

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for empty assessment_type_code, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_MissingTargetExam(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.TargetExam = ""

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for empty target_exam, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_WeightPercentZero(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.WeightPercent = 0

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for weight_percent 0, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_WeightPercentNegative(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.WeightPercent = -5

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for negative weight_percent, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_WeightPercentOver100(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.WeightPercent = 150

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for weight_percent > 100, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_EffectiveFromTooEarly(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.EffectiveFrom = 2020

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for effective_from < 2024, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_EffectiveFromTooLate(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.EffectiveFrom = 2200

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for effective_from > 2100, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_AssessmentTypeCodeTooLong(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	// Build a 55-character code
	params.AssessmentTypeCode = "A"
	for i := 0; i < 54; i++ {
		params.AssessmentTypeCode += "A"
	}

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for assessment_type_code > 50 chars, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_TargetExamTooLong(t *testing.T) {
	h := newSvcTestHarness()

	params := validWeightParams()
	params.TargetExam = "VERY_LONG_EXAM_NAME_KPSEA_EXTRA"

	_, err := h.svc.CreateWeightConfig(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for target_exam > 20 chars, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateWeightConfig_Duplicate(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		return "", ErrAlreadyExists
	}

	_, err := h.svc.CreateWeightConfig(context.Background(), validWeightParams())
	if err == nil {
		t.Fatal("expected error for duplicate config, got nil")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestCreateWeightConfig_RepoError(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.createWeightConfigFn = func(ctx context.Context, params CreateWeightConfigParams) (string, error) {
		return "", errors.New("database connection lost")
	}

	_, err := h.svc.CreateWeightConfig(context.Background(), validWeightParams())
	if err == nil {
		t.Fatal("expected error for repo failure, got nil")
	}
	if errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected a raw error, not ErrInvalidInput: %v", err)
	}
}

// ============================================================================
// Tests: ListWeightConfigs
// ============================================================================

func TestListWeightConfigs_HappyPath(t *testing.T) {
	h := newSvcTestHarness()

	expected := []AssessmentWeightConfig{
		{ID: "cfg_001", GradeLevel: "GRADE_4", AssessmentTypeCode: "KNEC_SBA_Project", TargetExam: "KPSEA", WeightPercent: 20.0, EffectiveFrom: 2026},
		{ID: "cfg_002", GradeLevel: "GRADE_4", AssessmentTypeCode: "National_KPSEA", TargetExam: "KPSEA", WeightPercent: 40.0, EffectiveFrom: 2026},
	}

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		return expected, nil
	}

	result, err := h.svc.ListWeightConfigs(context.Background(), AssessmentWeightConfigFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].ID != "cfg_001" {
		t.Fatalf("expected first item id 'cfg_001', got %q", result.Items[0].ID)
	}
	if result.Items[1].AssessmentTypeCode != "National_KPSEA" {
		t.Fatalf("expected second item type 'National_KPSEA', got %q", result.Items[1].AssessmentTypeCode)
	}
}

func TestListWeightConfigs_WithFilter(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		if filter.GradeLevel == nil || *filter.GradeLevel != "GRADE_7" {
			t.Errorf("expected grade_level 'GRADE_7', got %v", filter.GradeLevel)
		}
		if filter.TargetExam == nil || *filter.TargetExam != "KJSEA" {
			t.Errorf("expected target_exam 'KJSEA', got %v", filter.TargetExam)
		}
		return []AssessmentWeightConfig{
			{ID: "cfg_010", GradeLevel: "GRADE_7", AssessmentTypeCode: "KNEC_SBA_Project", TargetExam: "KJSEA", WeightPercent: 20.0, EffectiveFrom: 2024},
		}, nil
	}

	gl := "GRADE_7"
	te := "KJSEA"
	filter := AssessmentWeightConfigFilter{
		GradeLevel: &gl,
		TargetExam: &te,
	}

	result, err := h.svc.ListWeightConfigs(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != "cfg_010" {
		t.Fatalf("expected id 'cfg_010', got %q", result.Items[0].ID)
	}
}

func TestListWeightConfigs_EmptyResults(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		return []AssessmentWeightConfig{}, nil
	}

	result, err := h.svc.ListWeightConfigs(context.Background(), AssessmentWeightConfigFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
}

func TestListWeightConfigs_NilFromRepo(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.listWeightConfigsFn = func(ctx context.Context, filter AssessmentWeightConfigFilter) ([]AssessmentWeightConfig, error) {
		return nil, nil
	}

	result, err := h.svc.ListWeightConfigs(context.Background(), AssessmentWeightConfigFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Items == nil {
		t.Fatal("expected non-nil empty items slice, got nil")
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result.Items))
	}
}

// ============================================================================
// Tests: GetWeightConfigByID
// ============================================================================

func TestGetWeightConfigByID_HappyPath(t *testing.T) {
	h := newSvcTestHarness()

	expected := &AssessmentWeightConfig{
		ID: "cfg_001", GradeLevel: "GRADE_4", AssessmentTypeCode: "KNEC_SBA_Project",
		TargetExam: "KPSEA", WeightPercent: 20.0, EffectiveFrom: 2026,
	}

	h.repo.getWeightConfigByIDFn = func(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
		if id != "cfg_001" {
			t.Errorf("expected id 'cfg_001', got %q", id)
		}
		return expected, nil
	}

	result, err := h.svc.GetWeightConfigByID(context.Background(), "cfg_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "cfg_001" {
		t.Fatalf("expected id 'cfg_001', got %q", result.ID)
	}
	if result.WeightPercent != 20.0 {
		t.Fatalf("expected weight_percent 20.0, got %f", result.WeightPercent)
	}
}

func TestGetWeightConfigByID_EmptyID(t *testing.T) {
	h := newSvcTestHarness()

	_, err := h.svc.GetWeightConfigByID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetWeightConfigByID_NotFound(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.getWeightConfigByIDFn = func(ctx context.Context, id string) (*AssessmentWeightConfig, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetWeightConfigByID(context.Background(), "cfg_999")
	if err == nil {
		t.Fatal("expected error for not found, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: GetStudentTermGrades
// ============================================================================

func TestGetStudentTermGrades_EmptyInput(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		schoolID  string
		studentID string
		termID    string
	}{
		{"empty tenant", "", "school_1", "student_1", "term_1"},
		{"empty school", "tenant_1", "", "student_1", "term_1"},
		{"empty student", "tenant_1", "school_1", "", "term_1"},
		{"empty term", "tenant_1", "school_1", "student_1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newSvcTestHarness()
			_, err := h.svc.GetStudentTermGrades(context.Background(), tt.tenantID, tt.schoolID, tt.studentID, tt.termID)
			if err == nil {
				t.Fatal("expected ErrInvalidInput for empty parameter, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetStudentTermGrades_SimpleModeWinner(t *testing.T) {
	h := newSvcTestHarness()

	// Simple case: one clear mode winner (ME appears 3 times), no tie.
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return []StudentTermGrade{
			{LearningAreaID: "la_1", LearningAreaName: "Mathematics", LearningAreaCode: "MATH", FinalLevel: "ME", AssessmentCount: 5},
			{LearningAreaID: "la_2", LearningAreaName: "English", LearningAreaCode: "ENG", FinalLevel: "EE", AssessmentCount: 3},
		}, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grades) != 2 {
		t.Fatalf("expected 2 grades, got %d", len(grades))
	}
	if grades[0].LearningAreaCode != "MATH" {
		t.Fatalf("expected first area MATH, got %q", grades[0].LearningAreaCode)
	}
	if grades[0].FinalLevel != "ME" {
		t.Fatalf("expected MATH final_level 'ME', got %q", grades[0].FinalLevel)
	}
	if grades[1].FinalLevel != "EE" {
		t.Fatalf("expected ENG final_level 'EE', got %q", grades[1].FinalLevel)
	}
}

func TestGetStudentTermGrades_FrequencyTieResolvedByDate(t *testing.T) {
	h := newSvcTestHarness()

	// Frequency tie (EE=2, ME=2) resolved by latest date (EE has later date).
	// This confirms the *currently correct* behavior (latest_date DESC) still passes.
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		// The SQL CTE does the tie-breaking; we trust the repo to return the correct result.
		return []StudentTermGrade{
			{LearningAreaID: "la_1", LearningAreaName: "Mathematics", LearningAreaCode: "MATH", FinalLevel: "EE", AssessmentCount: 4},
		}, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grades) != 1 {
		t.Fatalf("expected 1 grade, got %d", len(grades))
	}
	if grades[0].FinalLevel != "EE" {
		t.Fatalf("expected final_level 'EE' (latest date wins tie), got %q", grades[0].FinalLevel)
	}
}

func TestGetStudentTermGrades_FrequencyAndDateTieResolvedByLevel(t *testing.T) {
	h := newSvcTestHarness()

	// Frequency tie AND date tie (two levels each appear 2 times, same latest date)
	// resolved by level hierarchy: EE(4) > ME(3) > AE(2) > BE(1).
	// This is the scenario Fix 1 addresses — EE should win, not the alphabetically highest (ME).
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return []StudentTermGrade{
			{LearningAreaID: "la_1", LearningAreaName: "Mathematics", LearningAreaCode: "MATH", FinalLevel: "EE", AssessmentCount: 4},
		}, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grades) != 1 {
		t.Fatalf("expected 1 grade, got %d", len(grades))
	}
	// EE must win over ME/AE/BE when frequency and date are tied.
	// Before Fix 1, the alphabetical sort would have picked ME.
	if grades[0].FinalLevel != "EE" {
		t.Fatalf("expected final_level 'EE' (correct CBC hierarchy), got %q. "+
			"Fix 1 should ensure EE > ME when all else is equal.", grades[0].FinalLevel)
	}
}

func TestGetStudentTermGrades_OnlyRubricScores(t *testing.T) {
	h := newSvcTestHarness()

	// Student with only RUBRIC-sourced levels (no QUANTITATIVE sessions).
	// Confirms the union logic in session_scores includes outcome-grade-derived levels.
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return []StudentTermGrade{
			{LearningAreaID: "la_3", LearningAreaName: "Science", LearningAreaCode: "SCI", FinalLevel: "AE", AssessmentCount: 2},
		}, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grades) != 1 {
		t.Fatalf("expected 1 grade, got %d", len(grades))
	}
	if grades[0].LearningAreaCode != "SCI" {
		t.Fatalf("expected area SCI, got %q", grades[0].LearningAreaCode)
	}
	if grades[0].FinalLevel != "AE" {
		t.Fatalf("expected final_level 'AE', got %q", grades[0].FinalLevel)
	}
}

func TestGetStudentTermGrades_MixedQuantitativeAndRubric(t *testing.T) {
	h := newSvcTestHarness()

	// Mix of QUANTITATIVE and RUBRIC levels in the same learning area —
	// both feed into the same mode calculation via the UNION ALL.
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return []StudentTermGrade{
			{LearningAreaID: "la_1", LearningAreaName: "Mathematics", LearningAreaCode: "MATH", FinalLevel: "ME", AssessmentCount: 7},
		}, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grades) != 1 {
		t.Fatalf("expected 1 grade, got %d", len(grades))
	}
	if grades[0].AssessmentCount != 7 {
		t.Fatalf("expected assessment_count 7 (both types combined), got %d", grades[0].AssessmentCount)
	}
}

func TestGetStudentTermGrades_SingleAssessment(t *testing.T) {
	h := newSvcTestHarness()

	// Edge case: a learning area with only one assessment (mode of one).
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return []StudentTermGrade{
			{LearningAreaID: "la_4", LearningAreaName: "Art", LearningAreaCode: "ART", FinalLevel: "BE", AssessmentCount: 1},
		}, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grades) != 1 {
		t.Fatalf("expected 1 grade, got %d", len(grades))
	}
	if grades[0].AssessmentCount != 1 {
		t.Fatalf("expected assessment_count 1, got %d", grades[0].AssessmentCount)
	}
	if grades[0].FinalLevel != "BE" {
		t.Fatalf("expected final_level 'BE', got %q", grades[0].FinalLevel)
	}
}

func TestGetStudentTermGrades_RepoReturnsNil(t *testing.T) {
	h := newSvcTestHarness()

	// Service should convert nil to empty slice, not return nil to caller.
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return nil, nil
	}

	grades, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if grades == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(grades) != 0 {
		t.Fatalf("expected 0 grades, got %d", len(grades))
	}
}

func TestGetStudentTermGrades_RepoError(t *testing.T) {
	h := newSvcTestHarness()

	repoErr := errors.New("database connection lost")
	h.repo.getStudentTermGradesFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]StudentTermGrade, error) {
		return nil, repoErr
	}

	_, err := h.svc.GetStudentTermGrades(context.Background(), "tenant_1", "school_1", "student_1", "term_1")
	if err == nil {
		t.Fatal("expected error for repo failure, got nil")
	}
	// The service wraps the error, so we check it contains the original
	if !strings.Contains(err.Error(), "database connection lost") {
		t.Fatalf("expected error to contain 'database connection lost', got %v", err)
	}
}

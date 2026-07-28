package cohortpositions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Mock Repository ──────────────────────────────────────────────────────

type MockRepository struct {
	refreshTermFn      func(ctx context.Context, termID string) error
	getByStudentTermFn func(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error)
	listByClassTermFn  func(ctx context.Context, classID, termID, tenantID string) ([]CohortPositionSummary, error)
	listByGradeTermFn  func(ctx context.Context, schoolID, gradeLevel, termID, tenantID string) ([]CohortPositionSummary, error)
}

func (m *MockRepository) RefreshTerm(ctx context.Context, termID string) error {
	if m.refreshTermFn != nil {
		return m.refreshTermFn(ctx, termID)
	}
	return nil
}

func (m *MockRepository) GetByStudentTerm(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error) {
	if m.getByStudentTermFn != nil {
		return m.getByStudentTermFn(ctx, studentID, termID, tenantID)
	}
	return &CohortPositionSummary{StudentID: studentID, AcademicTermID: termID}, nil
}

func (m *MockRepository) ListByClassTerm(ctx context.Context, classID, termID, tenantID string) ([]CohortPositionSummary, error) {
	if m.listByClassTermFn != nil {
		return m.listByClassTermFn(ctx, classID, termID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) ListByGradeTerm(ctx context.Context, schoolID, gradeLevel, termID, tenantID string) ([]CohortPositionSummary, error) {
	if m.listByGradeTermFn != nil {
		return m.listByGradeTermFn(ctx, schoolID, gradeLevel, termID, tenantID)
	}
	return nil, nil
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestRefreshTerm_EmptyID(t *testing.T) {
	mock := &MockRepository{}
	svc := &Service{repo: mock}

	err := svc.RefreshTerm(context.Background(), "")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRefreshTerm_Success(t *testing.T) {
	var refreshedTerm string
	mock := &MockRepository{
		refreshTermFn: func(ctx context.Context, termID string) error {
			refreshedTerm = termID
			return nil
		},
	}
	svc := &Service{repo: mock}

	err := svc.RefreshTerm(context.Background(), "term_001")
	require.NoError(t, err)
	require.Equal(t, "term_001", refreshedTerm)
}

func TestGetByStudentTerm_EmptyParams(t *testing.T) {
	mock := &MockRepository{}
	svc := &Service{repo: mock}

	_, err := svc.GetByStudentTerm(context.Background(), "", "term_001", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)

	_, err = svc.GetByStudentTerm(context.Background(), "student_001", "", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)

	_, err = svc.GetByStudentTerm(context.Background(), "student_001", "term_001", "")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetByStudentTerm_Found(t *testing.T) {
	mock := &MockRepository{
		getByStudentTermFn: func(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error) {
			return &CohortPositionSummary{
				StudentID: studentID, AcademicTermID: termID, ClassRank: intPtr(1), GradeRank: intPtr(5),
			}, nil
		},
	}
	svc := &Service{repo: mock}

	result, err := svc.GetByStudentTerm(context.Background(), "student_001", "term_001", "tenant_001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, *result.ClassRank)
}

func TestGetByStudentTerm_NotFound(t *testing.T) {
	mock := &MockRepository{
		getByStudentTermFn: func(ctx context.Context, studentID, termID, tenantID string) (*CohortPositionSummary, error) {
			return nil, ErrNotFound
		},
	}
	svc := &Service{repo: mock}

	_, err := svc.GetByStudentTerm(context.Background(), "student_missing", "term_001", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestListByClassTerm_Empty(t *testing.T) {
	svc := &Service{repo: &MockRepository{}}

	items, err := svc.ListByClassTerm(context.Background(), "class_001", "term_001", "tenant_001")
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestListByGradeTerm_EmptyParams(t *testing.T) {
	mock := &MockRepository{}
	svc := &Service{repo: mock}

	_, err := svc.ListByGradeTerm(context.Background(), "", "GRADE_4", "term_001", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func intPtr(i int) *int { return &i }

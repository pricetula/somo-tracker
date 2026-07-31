package teacherperformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Mock Repository ──────────────────────────────────────────────────────

type MockRepository struct {
	refreshComputationFn       func(ctx context.Context, termID string) error
	listByTeacherFn            func(ctx context.Context, tenantID, schoolID, userID, termID string, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error)
	listByTermFn               func(ctx context.Context, tenantID, schoolID, termID string, classID, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error)
	getByTeacherClassSubjectFn func(ctx context.Context, userID, learningAreaID, classID, termID string) (*TeacherSubjectPerformanceSummary, error)
}

func (m *MockRepository) RefreshComputation(ctx context.Context, termID string) error {
	if m.refreshComputationFn != nil {
		return m.refreshComputationFn(ctx, termID)
	}
	return nil
}

func (m *MockRepository) ListByTeacher(ctx context.Context, tenantID, schoolID, userID, termID string, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error) {
	if m.listByTeacherFn != nil {
		return m.listByTeacherFn(ctx, tenantID, schoolID, userID, termID, learningAreaID)
	}
	return nil, nil
}

func (m *MockRepository) ListByTerm(ctx context.Context, tenantID, schoolID, termID string, classID, learningAreaID *string) ([]TeacherSubjectPerformanceSummary, error) {
	if m.listByTermFn != nil {
		return m.listByTermFn(ctx, tenantID, schoolID, termID, classID, learningAreaID)
	}
	return nil, nil
}

func (m *MockRepository) GetByTeacherClassSubject(ctx context.Context, userID, learningAreaID, classID, termID string) (*TeacherSubjectPerformanceSummary, error) {
	if m.getByTeacherClassSubjectFn != nil {
		return m.getByTeacherClassSubjectFn(ctx, userID, learningAreaID, classID, termID)
	}
	return &TeacherSubjectPerformanceSummary{UserID: userID, LearningAreaID: learningAreaID, ClassID: classID}, nil
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestRefreshComputation_EmptyID(t *testing.T) {
	mock := &MockRepository{}
	svc := NewService(mock)

	err := svc.RefreshComputation(context.Background(), "")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestRefreshComputation_Success(t *testing.T) {
	var capturedTerm string
	mock := &MockRepository{
		refreshComputationFn: func(ctx context.Context, termID string) error {
			capturedTerm = termID
			return nil
		},
	}
	svc := NewService(mock)

	err := svc.RefreshComputation(context.Background(), "term_001")
	require.NoError(t, err)
	require.Equal(t, "term_001", capturedTerm)
}

func TestListByTeacher_EmptyParams(t *testing.T) {
	mock := &MockRepository{}
	svc := NewService(mock)

	_, err := svc.ListByTeacher(context.Background(), "", "school_001", "user_001", "term_001", nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestListByTeacher_Success(t *testing.T) {
	mock := &MockRepository{
		listByTeacherFn: func(ctx context.Context, tenantID, schoolID, userID, termID string, laID *string) ([]TeacherSubjectPerformanceSummary, error) {
			return []TeacherSubjectPerformanceSummary{
				{UserID: userID, LearningAreaID: "la_001", SubjectMeanScore: float64Ptr(78.5)},
			}, nil
		},
	}
	svc := NewService(mock)

	resp, err := svc.ListByTeacher(context.Background(), "tenant_001", "school_001", "user_001", "term_001", nil)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, 1, resp.Total)
}

func TestListByTerm_EmptyParams(t *testing.T) {
	mock := &MockRepository{}
	svc := NewService(mock)

	_, err := svc.ListByTerm(context.Background(), "", "school_001", "term_001", nil, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestListByTerm_Success(t *testing.T) {
	mock := &MockRepository{
		listByTermFn: func(ctx context.Context, tenantID, schoolID, termID string, classID, laID *string) ([]TeacherSubjectPerformanceSummary, error) {
			return []TeacherSubjectPerformanceSummary{
				{UserID: "user_001", SubjectMeanScore: float64Ptr(82.0)},
			}, nil
		},
	}
	svc := NewService(mock)

	resp, err := svc.ListByTerm(context.Background(), "tenant_001", "school_001", "term_001", nil, nil)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, 1, resp.Total)
}

func TestGetByTeacherClassSubject_Found(t *testing.T) {
	mock := &MockRepository{
		getByTeacherClassSubjectFn: func(ctx context.Context, userID, laID, classID, termID string) (*TeacherSubjectPerformanceSummary, error) {
			return &TeacherSubjectPerformanceSummary{
				UserID: userID, LearningAreaID: laID, ClassID: classID,
				SubjectMeanScore: float64Ptr(75.0),
			}, nil
		},
	}
	svc := NewService(mock)

	result, err := svc.GetByTeacherClassSubject(context.Background(), "user_001", "la_001", "class_001", "term_001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 75.0, *result.SubjectMeanScore)
}

func TestGetByTeacherClassSubject_NotFound(t *testing.T) {
	mock := &MockRepository{
		getByTeacherClassSubjectFn: func(ctx context.Context, userID, laID, classID, termID string) (*TeacherSubjectPerformanceSummary, error) {
			return nil, ErrNotFound
		},
	}
	svc := NewService(mock)

	_, err := svc.GetByTeacherClassSubject(context.Background(), "user_missing", "la_001", "class_001", "term_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func float64Ptr(f float64) *float64 { return &f }

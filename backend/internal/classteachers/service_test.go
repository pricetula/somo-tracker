package classteachers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Mock Repository ──────────────────────────────────────────────────────

type MockRepository struct {
	createFn           func(ctx context.Context, params CreateClassTeacherParams) (string, error)
	getByIDFn          func(ctx context.Context, id, tenantID string) (*ClassTeacher, error)
	listByClassFn      func(ctx context.Context, classID, tenantID string) ([]ClassTeacher, error)
	listByTeacherFn    func(ctx context.Context, userID, tenantID string) ([]ClassTeacher, error)
	deleteFn           func(ctx context.Context, id, tenantID string) error
	countPrimaryFn     func(ctx context.Context, classID, tenantID string) (int, error)
	existsForSubjectFn func(ctx context.Context, classID, userID, learningAreaID, tenantID string) (bool, error)
}

func (m *MockRepository) Create(ctx context.Context, params CreateClassTeacherParams) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, params)
	}
	return "ct_001", nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID)
	}
	return &ClassTeacher{ID: id, TenantID: tenantID, UserID: "user_001", TeacherRole: "PRIMARY_CLASS_TEACHER"}, nil
}

func (m *MockRepository) ListByClass(ctx context.Context, classID, tenantID string) ([]ClassTeacher, error) {
	if m.listByClassFn != nil {
		return m.listByClassFn(ctx, classID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) ListByTeacher(ctx context.Context, userID, tenantID string) ([]ClassTeacher, error) {
	if m.listByTeacherFn != nil {
		return m.listByTeacherFn(ctx, userID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, tenantID)
	}
	return nil
}

func (m *MockRepository) CountPrimaryForClass(ctx context.Context, classID, tenantID string) (int, error) {
	if m.countPrimaryFn != nil {
		return m.countPrimaryFn(ctx, classID, tenantID)
	}
	return 0, nil
}

func (m *MockRepository) ExistsForSubject(ctx context.Context, classID, userID, learningAreaID, tenantID string) (bool, error) {
	if m.existsForSubjectFn != nil {
		return m.existsForSubjectFn(ctx, classID, userID, learningAreaID, tenantID)
	}
	return false, nil
}

// ── Test Harness ─────────────────────────────────────────────────────────

type testHarness struct {
	svc  *Service
	repo *MockRepository
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	svc := NewService(repo)
	return &testHarness{svc: svc, repo: repo}
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestCreate_ValidPrimaryTeacher(t *testing.T) {
	h := newTestHarness()

	h.repo.countPrimaryFn = func(ctx context.Context, classID, tenantID string) (int, error) {
		return 0, nil
	}
	h.repo.createFn = func(ctx context.Context, params CreateClassTeacherParams) (string, error) {
		return "ct_001", nil
	}
	h.repo.getByIDFn = func(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
		return &ClassTeacher{
			ID: id, TenantID: tenantID, UserID: "user_001",
			ClassID: "class_001", TeacherRole: "PRIMARY_CLASS_TEACHER",
		}, nil
	}

	result, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "PRIMARY_CLASS_TEACHER", result.TeacherRole)
}

func TestCreate_PrimaryAlreadyAssigned(t *testing.T) {
	h := newTestHarness()

	h.repo.countPrimaryFn = func(ctx context.Context, classID, tenantID string) (int, error) {
		return 1, nil // already assigned
	}

	_, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPrimaryAlreadyAssigned)
}

func TestCreate_SubjectTeacherRequiresLearningArea(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "SUBJECT_TEACHER",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreate_InvalidRole(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "INVALID_ROLE",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreate_PrimaryWithLearningAreaRejected(t *testing.T) {
	h := newTestHarness()

	la := "la_001"
	_, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:         "user_001",
		ClassID:        "class_001",
		TeacherRole:    "PRIMARY_CLASS_TEACHER",
		LearningAreaID: &la,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreate_EmptyUserID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:      "",
		ClassID:     "class_001",
		TeacherRole: "SUBSTITUTE_TEACHER",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreate_EmptyTenant(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.Create(context.Background(), "", CreateClassTeacherPayload{
		UserID:      "user_001",
		ClassID:     "class_001",
		TeacherRole: "SUBSTITUTE_TEACHER",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetByID(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
		return &ClassTeacher{ID: id, TenantID: tenantID, TeacherRole: "PRIMARY_CLASS_TEACHER"}, nil
	}

	result, err := h.svc.GetByID(context.Background(), "ct_001", "tenant_001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "ct_001", result.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getByIDFn = func(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetByID(context.Background(), "ct_missing", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetByID_EmptyID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetByID(context.Background(), "", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestListByClass(t *testing.T) {
	h := newTestHarness()

	h.repo.listByClassFn = func(ctx context.Context, classID, tenantID string) ([]ClassTeacher, error) {
		return []ClassTeacher{
			{ID: "ct_001", ClassID: classID, TeacherRole: "PRIMARY_CLASS_TEACHER"},
			{ID: "ct_002", ClassID: classID, TeacherRole: "SUBJECT_TEACHER"},
		}, nil
	}

	resp, err := h.svc.ListByClass(context.Background(), "class_001", "tenant_001")
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	require.Equal(t, 2, resp.Total)
}

func TestListByTeacher(t *testing.T) {
	h := newTestHarness()

	h.repo.listByTeacherFn = func(ctx context.Context, userID, tenantID string) ([]ClassTeacher, error) {
		return []ClassTeacher{
			{ID: "ct_001", UserID: userID, ClassID: "class_001"},
		}, nil
	}

	resp, err := h.svc.ListByTeacher(context.Background(), "user_001", "tenant_001")
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
}

func TestDelete(t *testing.T) {
	h := newTestHarness()

	var deletedID string
	h.repo.deleteFn = func(ctx context.Context, id, tenantID string) error {
		deletedID = id
		return nil
	}

	err := h.svc.Delete(context.Background(), "ct_001", "tenant_001")
	require.NoError(t, err)
	require.Equal(t, "ct_001", deletedID)
}

func TestDelete_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.deleteFn = func(ctx context.Context, id, tenantID string) error {
		return ErrNotFound
	}

	err := h.svc.Delete(context.Background(), "ct_missing", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreate_RepoReturnsAlreadyExists(t *testing.T) {
	h := newTestHarness()

	h.repo.countPrimaryFn = func(ctx context.Context, classID, tenantID string) (int, error) {
		return 0, nil
	}
	h.repo.createFn = func(ctx context.Context, params CreateClassTeacherParams) (string, error) {
		return "", ErrAlreadyExists
	}

	la := "la_001"
	_, err := h.svc.Create(context.Background(), "tenant_001", CreateClassTeacherPayload{
		UserID:         "user_001",
		ClassID:        "class_001",
		TeacherRole:    "SUBJECT_TEACHER",
		LearningAreaID: &la,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAlreadyExists)
}

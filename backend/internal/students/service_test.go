package students

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Mock Repository ──────────────────────────────────────────────────────

type MockRepository struct {
	createFn       func(ctx context.Context, student *Student) (string, error)
	getByIDFn      func(ctx context.Context, id, tenantID, schoolID string) (*Student, error)
	listFn         func(ctx context.Context, filter ListFilter) ([]Student, int, error)
	updateFn       func(ctx context.Context, student *Student) error
	deleteFn       func(ctx context.Context, id, tenantID, schoolID string) error
	getDetailFn    func(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error)
	createEnrollFn func(ctx context.Context, enrollment *Enrollment) (string, error)
	createBatchFn  func(ctx context.Context, students []*Student) ([]string, error)
	batchEnrollFn  func(ctx context.Context, enrollments []*Enrollment, tenantID, schoolID string) ([]string, error)
	listEnrollFn   func(ctx context.Context, studentID, tenantID string) ([]Enrollment, error)
	isEnrolledFn   func(ctx context.Context, studentID, academicTermID, tenantID string) (bool, error)
}

func (m *MockRepository) Create(ctx context.Context, student *Student) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, student)
	}
	return "student_001", nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, schoolID string) (*Student, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID, schoolID)
	}
	return &Student{ID: id, FullName: "Test Student"}, nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter) ([]Student, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *MockRepository) Update(ctx context.Context, student *Student) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, student)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID, schoolID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, tenantID, schoolID)
	}
	return nil
}

func (m *MockRepository) GetDetail(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error) {
	if m.getDetailFn != nil {
		return m.getDetailFn(ctx, id, tenantID, schoolID)
	}
	return &StudentDetail{Student: Student{ID: id, FullName: "Detail"}}, nil
}

func (m *MockRepository) CreateBatch(ctx context.Context, students []*Student) ([]string, error) {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, students)
	}
	return nil, nil
}

func (m *MockRepository) CreateEnrollment(ctx context.Context, enrollment *Enrollment) (string, error) {
	if m.createEnrollFn != nil {
		return m.createEnrollFn(ctx, enrollment)
	}
	return "enroll_001", nil
}

func (m *MockRepository) CreateBatchEnrollments(ctx context.Context, enrollments []*Enrollment, tenantID, schoolID string) ([]string, error) {
	if m.batchEnrollFn != nil {
		return m.batchEnrollFn(ctx, enrollments, tenantID, schoolID)
	}
	return nil, nil
}

func (m *MockRepository) ListEnrollments(ctx context.Context, studentID, tenantID string) ([]Enrollment, error) {
	if m.listEnrollFn != nil {
		return m.listEnrollFn(ctx, studentID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) IsEnrolledInTerm(ctx context.Context, studentID, academicTermID, tenantID string) (bool, error) {
	if m.isEnrolledFn != nil {
		return m.isEnrolledFn(ctx, studentID, academicTermID, tenantID)
	}
	return false, nil
}

// ── Test Harness ─────────────────────────────────────────────────────────

type svcTestHarness struct {
	svc  *Service
	repo *MockRepository
}

func newSvcTestHarness() *svcTestHarness {
	repo := &MockRepository{}
	svc := NewService(repo)
	return &svcTestHarness{svc: svc, repo: repo}
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestCreateStudent(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.createFn = func(ctx context.Context, student *Student) (string, error) {
		return "student_001", nil
	}

	result, err := h.svc.Create(context.Background(), "tenant_001", "school_001", CreateStudentPayload{
		FullName: "John",
		Gender:   "M",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "student_001", result.ID)
	require.Equal(t, "John", result.FullName)
}

func TestCreateStudent_MissingFields(t *testing.T) {
	h := newSvcTestHarness()

	_, err := h.svc.Create(context.Background(), "tenant_001", "school_001", CreateStudentPayload{
		FullName: "",
		Gender:   "M",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateStudent_EmptyTenant(t *testing.T) {
	h := newSvcTestHarness()

	_, err := h.svc.Create(context.Background(), "", "school_001", CreateStudentPayload{
		FullName: "John", Gender: "M",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreateStudent_InvalidGender(t *testing.T) {
	h := newSvcTestHarness()

	_, err := h.svc.Create(context.Background(), "tenant_001", "school_001", CreateStudentPayload{
		FullName: "John", Gender: "X",
	})
	require.Error(t, err)
}

func TestGetDetail(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.getDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error) {
		return &StudentDetail{Student: Student{ID: id, FullName: "Jane"}}, nil
	}

	result, err := h.svc.GetDetail(context.Background(), "student_001", "tenant_001", "school_001")
	require.NoError(t, err)
	require.Equal(t, "Jane", result.FullName)
}

func TestGetDetail_NotFound(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.getDetailFn = func(ctx context.Context, id, tenantID, schoolID string) (*StudentDetail, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetDetail(context.Background(), "student_missing", "tenant_001", "school_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestListStudents(t *testing.T) {
	h := newSvcTestHarness()

	h.repo.listFn = func(ctx context.Context, filter ListFilter) ([]Student, int, error) {
		return []Student{
			{ID: "s1", FullName: "Alice"},
			{ID: "s2", FullName: "Bob"},
		}, 2, nil
	}

	resp, err := h.svc.ListStudents(context.Background(), ListFilter{
		TenantID: "tenant_001", SchoolID: "school_001", Page: 1, Limit: 50,
	})
	require.NoError(t, err)
	require.Equal(t, 2, resp.Total)
	require.Len(t, resp.Items, 2)
}

func TestUpdateStudent(t *testing.T) {
	h := newSvcTestHarness()

	var updatedStudent *Student
	h.repo.updateFn = func(ctx context.Context, student *Student) error {
		updatedStudent = student
		return nil
	}

	err := h.svc.Update(context.Background(), "student_001", "tenant_001", "school_001", UpdateStudentPayload{
		FullName: strPtr("Updated Name"),
	})
	require.NoError(t, err)
	require.NotNil(t, updatedStudent)
	require.Equal(t, "student_001", updatedStudent.ID)
}

func TestDeleteStudent(t *testing.T) {
	h := newSvcTestHarness()

	var deletedID string
	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		deletedID = id
		return nil
	}

	err := h.svc.Delete(context.Background(), "student_001", "tenant_001", "school_001")
	require.NoError(t, err)
	require.Equal(t, "student_001", deletedID)
}

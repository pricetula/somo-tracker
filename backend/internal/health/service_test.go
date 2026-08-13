package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── Mock Repository ──────────────────────────────────────────────────────

type MockRepository struct {
	createIncidentFn         func(ctx context.Context, params CreateIncidentParams) (string, error)
	getIncidentByIDFn        func(ctx context.Context, id, tenantID string) (*MedicalIncident, error)
	listIncidentsByStudentFn func(ctx context.Context, studentID, tenantID string) ([]MedicalIncident, error)
	listIncidentsBySchoolFn  func(ctx context.Context, tenantID, schoolID string, limit, offset int) ([]MedicalIncident, int, error)
	updateIncidentFn         func(ctx context.Context, id, tenantID string, payload UpdateMedicalIncidentPayload) error
	deleteIncidentFn         func(ctx context.Context, id, tenantID string) error
	upsertProfileFn          func(ctx context.Context, params UpsertProfileParams) (*StudentHealthProfile, error)
	getProfileByStudentFn    func(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error)
	getProfileByIDFn         func(ctx context.Context, id, tenantID string) (*StudentHealthProfile, error)
}

func (m *MockRepository) CreateIncident(ctx context.Context, params CreateIncidentParams) (string, error) {
	if m.createIncidentFn != nil {
		return m.createIncidentFn(ctx, params)
	}
	return "inc_001", nil
}

func (m *MockRepository) GetIncidentByID(ctx context.Context, id, tenantID string) (*MedicalIncident, error) {
	if m.getIncidentByIDFn != nil {
		return m.getIncidentByIDFn(ctx, id, tenantID)
	}
	return &MedicalIncident{ID: id, Symptoms: "Fever", ActionTaken: "Rested"}, nil
}

func (m *MockRepository) ListIncidentsByStudent(ctx context.Context, studentID, tenantID string) ([]MedicalIncident, error) {
	if m.listIncidentsByStudentFn != nil {
		return m.listIncidentsByStudentFn(ctx, studentID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) ListIncidentsBySchool(ctx context.Context, tenantID, schoolID string, limit, offset int) ([]MedicalIncident, int, error) {
	if m.listIncidentsBySchoolFn != nil {
		return m.listIncidentsBySchoolFn(ctx, tenantID, schoolID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockRepository) UpdateIncident(ctx context.Context, id, tenantID string, payload UpdateMedicalIncidentPayload) error {
	if m.updateIncidentFn != nil {
		return m.updateIncidentFn(ctx, id, tenantID, payload)
	}
	return nil
}

func (m *MockRepository) DeleteIncident(ctx context.Context, id, tenantID string) error {
	if m.deleteIncidentFn != nil {
		return m.deleteIncidentFn(ctx, id, tenantID)
	}
	return nil
}

func (m *MockRepository) UpsertProfile(ctx context.Context, params UpsertProfileParams) (*StudentHealthProfile, error) {
	if m.upsertProfileFn != nil {
		return m.upsertProfileFn(ctx, params)
	}
	bg := "O+"
	return &StudentHealthProfile{ID: "prof_001", StudentID: params.StudentID, BloodGroup: &bg}, nil
}

func (m *MockRepository) GetProfileByStudent(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error) {
	if m.getProfileByStudentFn != nil {
		return m.getProfileByStudentFn(ctx, studentID, tenantID)
	}
	return nil, ErrNotFound
}

func (m *MockRepository) GetProfileByID(ctx context.Context, id, tenantID string) (*StudentHealthProfile, error) {
	if m.getProfileByIDFn != nil {
		return m.getProfileByIDFn(ctx, id, tenantID)
	}
	bg := "A+"
	return &StudentHealthProfile{ID: id, BloodGroup: &bg}, nil
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

func TestCreateIncident(t *testing.T) {
	h := newTestHarness()

	h.repo.createIncidentFn = func(ctx context.Context, params CreateIncidentParams) (string, error) {
		return "inc_001", nil
	}
	h.repo.getIncidentByIDFn = func(ctx context.Context, id, tenantID string) (*MedicalIncident, error) {
		return &MedicalIncident{
			ID: id, StudentID: "student_001",
			Symptoms: "Headache", ActionTaken: "Rested",
		}, nil
	}

	result, err := h.svc.CreateIncident(context.Background(), "tenant_001", "user_001", CreateMedicalIncidentPayload{
		StudentID:   "student_001",
		Symptoms:    "Headache",
		ActionTaken: "Rested",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Headache", result.Symptoms)
}

func TestCreateIncident_MissingFields(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateIncident(context.Background(), "tenant_001", "user_001", CreateMedicalIncidentPayload{
		StudentID:   "",
		Symptoms:    "Headache",
		ActionTaken: "Rested",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)

	_, err = h.svc.CreateIncident(context.Background(), "tenant_001", "user_001", CreateMedicalIncidentPayload{
		StudentID:   "student_001",
		Symptoms:    "",
		ActionTaken: "Rested",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)

	_, err = h.svc.CreateIncident(context.Background(), "tenant_001", "user_001", CreateMedicalIncidentPayload{
		StudentID:   "student_001",
		Symptoms:    "Headache",
		ActionTaken: "",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetIncidentByID(t *testing.T) {
	h := newTestHarness()

	h.repo.getIncidentByIDFn = func(ctx context.Context, id, tenantID string) (*MedicalIncident, error) {
		return &MedicalIncident{ID: id, Symptoms: "Fever"}, nil
	}

	result, err := h.svc.GetIncidentByID(context.Background(), "inc_001", "tenant_001")
	require.NoError(t, err)
	require.Equal(t, "inc_001", result.ID)
}

func TestGetIncidentByID_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getIncidentByIDFn = func(ctx context.Context, id, tenantID string) (*MedicalIncident, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetIncidentByID(context.Background(), "inc_missing", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestListIncidentsByStudent(t *testing.T) {
	h := newTestHarness()

	h.repo.listIncidentsByStudentFn = func(ctx context.Context, studentID, tenantID string) ([]MedicalIncident, error) {
		return []MedicalIncident{
			{ID: "inc_001", Symptoms: "Fever"},
			{ID: "inc_002", Symptoms: "Injury"},
		}, nil
	}

	resp, err := h.svc.ListIncidentsByStudent(context.Background(), "student_001", "tenant_001")
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
}

func TestListIncidentsBySchool(t *testing.T) {
	h := newTestHarness()

	h.repo.listIncidentsBySchoolFn = func(ctx context.Context, tenantID, schoolID string, limit, offset int) ([]MedicalIncident, int, error) {
		return []MedicalIncident{{ID: "inc_001", Symptoms: "Fever"}}, 1, nil
	}

	resp, err := h.svc.ListIncidentsBySchool(context.Background(), "tenant_001", "school_001", 1, 50)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, 1, resp.Total)
}

func TestUpdateIncident(t *testing.T) {
	h := newTestHarness()

	var updated bool
	h.repo.updateIncidentFn = func(ctx context.Context, id, tenantID string, payload UpdateMedicalIncidentPayload) error {
		updated = true
		return nil
	}

	newSymptoms := "Severe fever"
	err := h.svc.UpdateIncident(context.Background(), "inc_001", "tenant_001", UpdateMedicalIncidentPayload{Symptoms: &newSymptoms})
	require.NoError(t, err)
	require.True(t, updated)
}

func TestUpdateIncident_NoFields(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateIncident(context.Background(), "inc_001", "tenant_001", UpdateMedicalIncidentPayload{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestDeleteIncident(t *testing.T) {
	h := newTestHarness()

	var deletedID string
	h.repo.deleteIncidentFn = func(ctx context.Context, id, tenantID string) error {
		deletedID = id
		return nil
	}

	err := h.svc.DeleteIncident(context.Background(), "inc_001", "tenant_001")
	require.NoError(t, err)
	require.Equal(t, "inc_001", deletedID)
}

func TestUpsertProfile(t *testing.T) {
	h := newTestHarness()

	h.repo.upsertProfileFn = func(ctx context.Context, params UpsertProfileParams) (*StudentHealthProfile, error) {
		bg := "A+"
		return &StudentHealthProfile{
			StudentID: params.StudentID, BloodGroup: &bg,
			Allergies: params.Allergies,
		}, nil
	}

	result, err := h.svc.UpsertProfile(context.Background(), "tenant_001", "student_001", "user_001", UpsertHealthProfilePayload{
		BloodGroup: strPtr("A+"),
		Allergies:  []string{"Peanuts"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "A+", *result.BloodGroup)
}

func TestGetProfileByStudent(t *testing.T) {
	h := newTestHarness()

	h.repo.getProfileByStudentFn = func(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error) {
		bg := "B+"
		return &StudentHealthProfile{StudentID: studentID, BloodGroup: &bg}, nil
	}

	result, err := h.svc.GetProfileByStudent(context.Background(), "student_001", "tenant_001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "B+", *result.BloodGroup)
}

func TestGetStudentHealth(t *testing.T) {
	h := newTestHarness()

	h.repo.getProfileByStudentFn = func(ctx context.Context, studentID, tenantID string) (*StudentHealthProfile, error) {
		bg := "O+"
		return &StudentHealthProfile{StudentID: studentID, BloodGroup: &bg}, nil
	}
	h.repo.listIncidentsByStudentFn = func(ctx context.Context, studentID, tenantID string) ([]MedicalIncident, error) {
		return []MedicalIncident{{ID: "inc_001", Symptoms: "Fever"}}, nil
	}

	result, err := h.svc.GetStudentHealth(context.Background(), "student_001", "tenant_001")
	require.NoError(t, err)
	require.NotNil(t, result.Profile)
	require.Len(t, result.Incidents, 1)
}

func strPtr(s string) *string { return &s }

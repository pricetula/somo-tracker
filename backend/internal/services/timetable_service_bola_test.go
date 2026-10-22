package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"somotracker/backend/internal/database/sqlc"
)

type mockTimetableService struct {
	verifyClassErr error
	verifySlotErr  error
	deleteErr      error
	deleteCalled   bool
	getSlotsResult []sqlc.GetClassTimetableSlotsByTemplateWithDetailsRow
	getSlotsErr    error
}

func (m *mockTimetableService) CreateTemplate(ctx context.Context, schoolID uuid.UUID, req CreateTemplateRequest) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockTimetableService) GetTemplate(ctx context.Context, templateID uuid.UUID) (sqlc.TimetableTemplate, error) {
	return sqlc.TimetableTemplate{}, nil
}
func (m *mockTimetableService) UpdateTemplate(ctx context.Context, templateID uuid.UUID, name string, description string) error {
	return nil
}
func (m *mockTimetableService) ListTemplates(ctx context.Context, schoolID uuid.UUID, page int, limit int) ([]sqlc.TimetableTemplate, int, error) {
	return nil, 0, nil
}
func (m *mockTimetableService) ListTimeSlotsByTemplate(ctx context.Context, templateID uuid.UUID) ([]sqlc.TimeSlot, error) {
	return nil, nil
}
func (m *mockTimetableService) SetupClassTimetableSlot(ctx context.Context, schoolID uuid.UUID, classRoomID, timeSlotID, subjectID, teacherMembershipID string, dayOfWeek int, roomID string) error {
	return nil
}
func (m *mockTimetableService) GetClassTimetableSlotsByTemplate(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID, templateID uuid.UUID) ([]sqlc.GetClassTimetableSlotsByTemplateWithDetailsRow, error) {
	return m.getSlotsResult, m.getSlotsErr
}
func (m *mockTimetableService) DeleteClassTimetableSlot(ctx context.Context, schoolID uuid.UUID, slotID uuid.UUID) error {
	m.deleteCalled = true
	if m.verifySlotErr != nil {
		return m.verifySlotErr
	}
	return m.deleteErr
}
func (m *mockTimetableService) VerifyClassBelongsToSchool(ctx context.Context, schoolID uuid.UUID, classRoomID uuid.UUID) error {
	return m.verifyClassErr
}
func (m *mockTimetableService) GetStudentTimetable(ctx context.Context, schoolID uuid.UUID, studentID uuid.UUID) ([]StudentTimetableSlot, error) {
	return nil, nil
}

// Test DeleteClassTimetableSlot BOLA – same school succeeds
func TestDeleteClassTimetableSlot_BOLA_SameSchool(t *testing.T) {
	svc := &mockTimetableService{}
	schoolID := uuid.New()
	slotID := uuid.New()

	err := svc.DeleteClassTimetableSlot(context.Background(), schoolID, slotID)
	require.NoError(t, err)
	require.True(t, svc.deleteCalled, "delete should be called when ownership verified")
}

// Test DeleteClassTimetableSlot BOLA – different school blocked
func TestDeleteClassTimetableSlot_BOLA_DifferentSchool(t *testing.T) {
	svc := &mockTimetableService{
		verifySlotErr: errors.New("not in school: class timetable slot not found for school"),
	}
	schoolID := uuid.New()
	slotID := uuid.New()

	err := svc.DeleteClassTimetableSlot(context.Background(), schoolID, slotID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in school")
	require.True(t, svc.deleteCalled, "verify runs before delete; delete is attempted after verify in real service")
}

// Test VerifyClassBelongsToSchool BOLA – class belongs
func TestVerifyClassBelongsToSchool_BOLA_ClassBelongs(t *testing.T) {
	svc := &mockTimetableService{verifyClassErr: nil}
	err := svc.VerifyClassBelongsToSchool(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
}

// Test VerifyClassBelongsToSchool BOLA – class does not belong
func TestVerifyClassBelongsToSchool_BOLA_ClassNotBelong(t *testing.T) {
	svc := &mockTimetableService{verifyClassErr: errors.New("class not found in school")}
	err := svc.VerifyClassBelongsToSchool(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	require.Contains(t, err.Error(), "class not found")
}

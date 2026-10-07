package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/services"
)

type mockStudentImportService struct {
	jobs   map[uuid.UUID]*services.BulkJob
	items  map[uuid.UUID]*services.BulkJobItem
	insert func(ctx context.Context, schoolID uuid.UUID, admissionNumber, fullName string, dob interface{}, gender string, metadata json.RawMessage) error
}

func (m *mockStudentImportService) CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []services.StudentImportItem) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockStudentImportService) GetJob(ctx context.Context, jobID uuid.UUID) (*services.BulkJob, error) {
	return m.jobs[jobID], nil
}
func (m *mockStudentImportService) GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]services.BulkJobItem, error) {
	var out []services.BulkJobItem
	for _, it := range m.items {
		if it.JobID == jobID {
			out = append(out, *it)
		}
	}
	return out, nil
}
func (m *mockStudentImportService) GetItemByID(ctx context.Context, itemID uuid.UUID) (*services.BulkJobItem, error) {
	return m.items[itemID], nil
}
func (m *mockStudentImportService) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	m.items[itemID].Status = status
	return nil
}
func (m *mockStudentImportService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	if m.jobs[jobID] != nil {
		m.jobs[jobID].Status = status
	}
	return nil
}
func (m *mockStudentImportService) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error {
	if j, ok := m.jobs[jobID]; ok {
		j.SucceededCount += succeeded
		j.FailedCount += failed
		j.DeferredCount += deferred
	}
	return nil
}
func (m *mockStudentImportService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*services.BulkJob, error) {
	return nil, nil
}
func (m *mockStudentImportService) UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockStudentImportService) RecomputeGenderCounts(ctx context.Context, schoolID uuid.UUID) error {
	return nil
}
func (m *mockStudentImportService) InsertStudent(ctx context.Context, schoolID uuid.UUID, admissionNumber, fullName string, dob interface{}, gender string, metadata json.RawMessage) error {
	if m.insert != nil {
		return m.insert(ctx, schoolID, admissionNumber, fullName, dob, gender, metadata)
	}
	return nil
}
func (m *mockStudentImportService) TryFinalizeJob(ctx context.Context, jobID uuid.UUID) (bool, error) {
	return true, nil
}

func TestStudentImportWorker_Success(t *testing.T) {
	// Arrange
	jobID := uuid.New()
	schoolID := uuid.New()
	svc := &mockStudentImportService{
		jobs: map[uuid.UUID]*services.BulkJob{
			jobID: {ID: jobID, SchoolID: schoolID, TotalRecords: 2, Status: "QUEUED"},
		},
		items: map[uuid.UUID]*services.BulkJobItem{},
	}
	// TODO: build payload, call processor.ProcessTask, assert items marked SUCCEEDED and counters incremented
	require.NotNil(t, svc)
}

func TestStudentImportWorker_SuccessWithErrors(t *testing.T) {
	// Arrange duplicate admission_number -> InsertStudent returns unique violation
	jobID := uuid.New()
	schoolID := uuid.New()
	svc := &mockStudentImportService{
		jobs: map[uuid.UUID]*services.BulkJob{
			jobID: {ID: jobID, SchoolID: schoolID, TotalRecords: 2, Status: "QUEUED"},
		},
		items: map[uuid.UUID]*services.BulkJobItem{},
		insert: func(ctx context.Context, schoolID uuid.UUID, admissionNumber, fullName string, dob interface{}, gender string, metadata json.RawMessage) error {
			if admissionNumber == "DUP" {
				return &mockUniqueError{msg: "duplicate key"}
			}
			return nil
		},
	}
	// TODO: process batch with one duplicate and one good row, assert FAILED + SUCCEEDED
	require.NotNil(t, svc)
}

func TestStudentImportWorker_DuplicateWithinFile(t *testing.T) {
	// Handler already dedupes within file; worker should never see duplicate payloads from same job
	// Test that first occurrence wins and second is skipped at handler level
	t.Skip("handler dedupe tested in handler tests")
}

type mockUniqueError struct{ msg string }

func (e *mockUniqueError) Error() string { return e.msg }

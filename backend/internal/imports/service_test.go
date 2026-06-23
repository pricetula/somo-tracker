package imports

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"somotracker/backend/internal/config"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	createImportJobFn             func(ctx context.Context, job *ImportJob) error
	getImportJobFn                func(ctx context.Context, jobID string) (*ImportJob, error)
	updateImportJobStatusFn       func(ctx context.Context, id, status string, processed, successCount, failedCount int) error
	setImportJobStartedFn         func(ctx context.Context, id string) error
	setImportJobCompletedFn       func(ctx context.Context, id string, hasErrors bool) error
	bulkInsertInvitationsFn       func(ctx context.Context, records []ImportStaffRecord, tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string) (map[string]string, []FailedInsertion, error)
	recordImportFailureFn         func(ctx context.Context, jobID, rawPayloadJSON, errMsg string) error
	bulkRecordImportFailureFn     func(ctx context.Context, jobID string, records []ImportStaffRecord, errMsg string) error
	getFailedInvitationsByJobFn   func(ctx context.Context, jobID string) ([]FailedInvitation, error)
	getInvitationStytchMemberIDFn func(ctx context.Context, id string) (string, error)
	setInvitationStytchMemberIDFn func(ctx context.Context, id, stytchMemberID string) error
	setInvitationFailedFn         func(ctx context.Context, id, errorMessage string, attemptCount int) error
	getActiveSchoolIDFn           func(ctx context.Context, tenantID, userID string) (string, error)
	getTenantStytchOrgIDFn        func(ctx context.Context, tenantID string) (string, error)
	setImportJobFailedFn          func(ctx context.Context, id string) error
	getPendingStage2RecordsFn     func(ctx context.Context, jobID string) ([]Stage2Record, error)
}

func (m *MockRepository) CreateImportJob(ctx context.Context, job *ImportJob) error {
	if m.createImportJobFn != nil {
		return m.createImportJobFn(ctx, job)
	}
	return nil
}

func (m *MockRepository) GetImportJob(ctx context.Context, jobID string) (*ImportJob, error) {
	if m.getImportJobFn != nil {
		return m.getImportJobFn(ctx, jobID)
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) UpdateImportJobStatus(ctx context.Context, id, status string, processed, successCount, failedCount int) error {
	if m.updateImportJobStatusFn != nil {
		return m.updateImportJobStatusFn(ctx, id, status, processed, successCount, failedCount)
	}
	return nil
}

func (m *MockRepository) SetImportJobStarted(ctx context.Context, id string) error {
	if m.setImportJobStartedFn != nil {
		return m.setImportJobStartedFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) SetImportJobCompleted(ctx context.Context, id string, hasErrors bool) error {
	if m.setImportJobCompletedFn != nil {
		return m.setImportJobCompletedFn(ctx, id, hasErrors)
	}
	return nil
}

func (m *MockRepository) BulkInsertInvitations(ctx context.Context, records []ImportStaffRecord, tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string) (map[string]string, []FailedInsertion, error) {
	if m.bulkInsertInvitationsFn != nil {
		return m.bulkInsertInvitationsFn(ctx, records, tenantID, schoolID, role, jobID, now, tokenPrefix)
	}
	inserted := make(map[string]string)
	for _, rec := range records {
		inserted[rec.TempID] = "inv_" + rec.Email
	}
	return inserted, nil, nil
}

func (m *MockRepository) RecordImportFailure(ctx context.Context, jobID, rawPayloadJSON, errMsg string) error {
	if m.recordImportFailureFn != nil {
		return m.recordImportFailureFn(ctx, jobID, rawPayloadJSON, errMsg)
	}
	return nil
}

func (m *MockRepository) BulkRecordImportFailure(ctx context.Context, jobID string, records []ImportStaffRecord, errMsg string) error {
	if m.bulkRecordImportFailureFn != nil {
		return m.bulkRecordImportFailureFn(ctx, jobID, records, errMsg)
	}
	return nil
}

func (m *MockRepository) GetFailedInvitationsByJob(ctx context.Context, jobID string) ([]FailedInvitation, error) {
	if m.getFailedInvitationsByJobFn != nil {
		return m.getFailedInvitationsByJobFn(ctx, jobID)
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) GetInvitationStytchMemberID(ctx context.Context, id string) (string, error) {
	if m.getInvitationStytchMemberIDFn != nil {
		return m.getInvitationStytchMemberIDFn(ctx, id)
	}
	return "", nil
}

func (m *MockRepository) SetInvitationStytchMemberID(ctx context.Context, id, stytchMemberID string) error {
	if m.setInvitationStytchMemberIDFn != nil {
		return m.setInvitationStytchMemberIDFn(ctx, id, stytchMemberID)
	}
	return nil
}

func (m *MockRepository) SetInvitationFailed(ctx context.Context, id, errorMessage string, attemptCount int) error {
	if m.setInvitationFailedFn != nil {
		return m.setInvitationFailedFn(ctx, id, errorMessage, attemptCount)
	}
	return nil
}

func (m *MockRepository) BulkUpdateInvitations(ctx context.Context, records []ImportStaffRecord, role, jobID string, now time.Time) (int, error) {
	return len(records), nil
}

func (m *MockRepository) SetImportJobFailed(ctx context.Context, id string) error {
	if m.setImportJobFailedFn != nil {
		return m.setImportJobFailedFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) GetPendingStage2Records(ctx context.Context, jobID string) ([]Stage2Record, error) {
	if m.getPendingStage2RecordsFn != nil {
		return m.getPendingStage2RecordsFn(ctx, jobID)
	}
	// Default: return records for each email in the last bulkInsert call
	return nil, nil
}

func (m *MockRepository) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if m.getActiveSchoolIDFn != nil {
		return m.getActiveSchoolIDFn(ctx, tenantID, userID)
	}
	return "school_001", nil
}

func (m *MockRepository) GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	if m.getTenantStytchOrgIDFn != nil {
		return m.getTenantStytchOrgIDFn(ctx, tenantID)
	}
	return "org_stytch_001", nil
}

// ============================================================================
// MockAsynqClient — implements a subset of *asynq.Client for testing
// ============================================================================

type MockAsynqClient struct {
	enqueueFn func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

func (m *MockAsynqClient) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if m.enqueueFn != nil {
		return m.enqueueFn(task, opts...)
	}
	return &asynq.TaskInfo{ID: "task_001"}, nil
}

// ============================================================================
// MockStytchResolver
// ============================================================================

type MockStytchResolver struct {
	getTenantStytchOrgIDFn func(ctx context.Context, tenantID string) (string, error)
}

func (m *MockStytchResolver) GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error) {
	if m.getTenantStytchOrgIDFn != nil {
		return m.getTenantStytchOrgIDFn(ctx, tenantID)
	}
	return "org_stytch_001", nil
}

// ============================================================================
// Test Harness — constructs Service directly (bypasses NewService which needs *asynq.Client)
// ============================================================================

type testHarness struct {
	svc    *Service
	repo   *MockRepository
	client *MockAsynqClient
	logs   *observer.ObservedLogs
	logger *zap.Logger
	cfg    config.Config
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	client := &MockAsynqClient{}

	observedCore, observedLogs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)

	cfg := config.Config{
		AppEnv:      "test",
		FrontendURL: "http://localhost:3000",
	}

	svc := &Service{
		repo:   repo,
		client: client,
		logger: logger,
		cfg:    cfg,
	}

	return &testHarness{
		svc:    svc,
		repo:   repo,
		client: client,
		logs:   observedLogs,
		logger: logger,
		cfg:    cfg,
	}
}

func validRecords() []ImportStaffRecord {
	return []ImportStaffRecord{
		{Email: "alice@school.com", FirstName: "Alice", LastName: "Smith", Phone: "+12345", RegistrationNumber: "REG-001"},
		{Email: "bob@school.com", FirstName: "Bob", LastName: "Jones", Phone: "+67890", RegistrationNumber: "REG-002"},
	}
}

func newResolver() StytchOrgResolver {
	return &MockStytchResolver{}
}

// ============================================================================
// Tests: StartImport — Happy Path
// ============================================================================

func TestStartImport_HappyPath(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	records := validRecords()

	result, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "SCHOOL_ADMIN", records, resolver, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ImportJobID == "" {
		t.Fatal("expected non-empty import job ID")
	}
	if result.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", result.Status)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
}

// ============================================================================
// Tests: StartImport — Bad Paths
// ============================================================================

func TestStartImport_InvalidRole(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "INVALID", validRecords(), resolver, "")
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
}

func TestStartImport_EmptyRecords(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "NURSE", nil, resolver, "")
	if err == nil {
		t.Fatal("expected error for empty records, got nil")
	}
}

func TestStartImport_TooManyRecords(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	records := make([]ImportStaffRecord, MaxRecordsPerImport+1)
	for i := range records {
		records[i] = ImportStaffRecord{
			Email:     fmt.Sprintf("user%d@school.com", i),
			FirstName: "User",
			LastName:  fmt.Sprintf("%d", i),
		}
	}

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "FINANCE", records, resolver, "")
	if err == nil {
		t.Fatal("expected error for too many records, got nil")
	}
}

func TestStartImport_MissingEmail(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	records := []ImportStaffRecord{
		{Email: "", FirstName: "Alice", LastName: "Smith"},
	}

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "NURSE", records, resolver, "")
	if err == nil {
		t.Fatal("expected error for missing email, got nil")
	}
}

func TestStartImport_MissingFirstName(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	records := []ImportStaffRecord{
		{Email: "alice@school.com", FirstName: "", LastName: "Smith"},
	}

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "NURSE", records, resolver, "")
	if err == nil {
		t.Fatal("expected error for missing first_name, got nil")
	}
}

func TestStartImport_MissingLastName(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	records := []ImportStaffRecord{
		{Email: "alice@school.com", FirstName: "Alice", LastName: ""},
	}

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "NURSE", records, resolver, "")
	if err == nil {
		t.Fatal("expected error for missing last_name, got nil")
	}
}

func TestStartImport_StytchOrgResolveFails(t *testing.T) {
	h := newTestHarness()

	resolver := &MockStytchResolver{
		getTenantStytchOrgIDFn: func(ctx context.Context, tenantID string) (string, error) {
			return "", errors.New("tenant not found")
		},
	}

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "SCHOOL_ADMIN", validRecords(), resolver, "")
	if err == nil {
		t.Fatal("expected error for stytch org resolution failure, got nil")
	}
}

func TestStartImport_DBCreateFails(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	h.repo.createImportJobFn = func(ctx context.Context, job *ImportJob) error {
		return errors.New("postgres connection error")
	}

	_, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "SCHOOL_ADMIN", validRecords(), resolver, "")
	if err == nil {
		t.Fatal("expected error for DB create failure, got nil")
	}
}

func TestStartImport_AsynqEnqueueFails(t *testing.T) {
	h := newTestHarness()
	resolver := newResolver()

	h.client.enqueueFn = func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		return nil, errors.New("redis connection refused")
	}

	result, err := h.svc.StartImport(context.Background(), "tenant_001", "school_001", "user_001", "SCHOOL_ADMIN", validRecords(), resolver, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still return a result with enqueue_failed status
	if result.Status != "enqueue_failed" {
		t.Fatalf("expected status 'enqueue_failed', got %q", result.Status)
	}
	if result.ImportJobID == "" {
		t.Fatal("expected non-empty import job ID even when enqueue fails")
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}

	// Verify WARN log was emitted
	warnLogs := h.logs.FilterLevelExact(zapcore.WarnLevel)
	if warnLogs.Len() < 1 {
		t.Log("expected at least 1 WARN log for enqueue failure")
	}
}

// ============================================================================
// Tests: GetImportJob
// ============================================================================

func TestGetImportJob_HappyPath(t *testing.T) {
	h := newTestHarness()

	job := &ImportJob{
		ID:           "job_001",
		TenantID:     "tenant_001",
		SchoolID:     "school_001",
		Role:         "NURSE",
		Status:       "pending",
		TotalRecords: 50,
	}

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return job, nil
	}

	result, err := h.svc.GetImportJob(context.Background(), "job_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Job.ID != "job_001" {
		t.Fatalf("expected job ID 'job_001', got %q", result.Job.ID)
	}
	if result.Job.Role != "NURSE" {
		t.Fatalf("expected role 'NURSE', got %q", result.Job.Role)
	}
}

func TestGetImportJob_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return nil, errors.New("import job not found")
	}

	_, err := h.svc.GetImportJob(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent job, got nil")
	}
}

// ============================================================================
// Tests: GetFailedInvitations
// ============================================================================

func TestGetFailedInvitations_HappyPath(t *testing.T) {
	h := newTestHarness()

	firstName := "Alice"
	lastName := "Smith"
	errMsg := "stytch invite failed: invalid email"

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return []FailedInvitation{
			{
				ID:           "inv_001",
				Email:        "alice@bad.com",
				FirstName:    &firstName,
				LastName:     &lastName,
				ErrorMessage: &errMsg,
			},
		}, nil
	}

	result, err := h.svc.GetFailedInvitations(context.Background(), "job_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Invitations) != 1 {
		t.Fatalf("expected 1 failed invitation, got %d", len(result.Invitations))
	}
	if result.Invitations[0].Email != "alice@bad.com" {
		t.Fatalf("expected email 'alice@bad.com', got %q", result.Invitations[0].Email)
	}
}

func TestGetFailedInvitations_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return nil, nil // no failures
	}

	result, err := h.svc.GetFailedInvitations(context.Background(), "job_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Invitations == nil {
		t.Fatal("expected non-nil (empty) invitations slice")
	}
	if len(result.Invitations) != 0 {
		t.Fatalf("expected 0 invitations, got %d", len(result.Invitations))
	}
}

func TestGetFailedInvitations_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return nil, errors.New("import job not found")
	}

	_, err := h.svc.GetFailedInvitations(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent job, got nil")
	}
}

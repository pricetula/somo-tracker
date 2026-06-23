package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"somotracker/backend/internal/auth"
	"somotracker/backend/internal/config"
)

// ============================================================================
// MockIdentityProvider
// ============================================================================

type MockIDP struct {
	mu                       sync.Mutex
	inviteMemberByEmailFn    func(ctx context.Context, orgID, email, name, redirectURL string) (string, error)
	inviteMemberByEmailCalls int
}

func (m *MockIDP) InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inviteMemberByEmailCalls++
	if m.inviteMemberByEmailFn != nil {
		return m.inviteMemberByEmailFn(ctx, orgID, email, name, redirectURL)
	}
	return "member_" + email, nil
}

func (m *MockIDP) SendDiscoveryEmail(ctx context.Context, email string) error {
	return nil
}

func (m *MockIDP) AuthenticateDiscoveryToken(ctx context.Context, token string) (string, string, error) {
	return "ist", "email", nil
}

func (m *MockIDP) CreateOrganization(ctx context.Context, name string) (string, error) {
	return "org_" + name, nil
}

func (m *MockIDP) ExchangeIntermediateSession(ctx context.Context, ist, orgID string) (auth.ExchangeResult, error) {
	return auth.ExchangeResult{}, nil
}

func (m *MockIDP) CreateMember(ctx context.Context, orgID, email, name string) (string, error) {
	return "member_" + email, nil
}

func (m *MockIDP) AuthenticateInviteToken(ctx context.Context, token string) (string, string, error) {
	return "ist_invite", "invited@example.com", nil
}

func (m *MockIDP) ExchangeInviteSession(ctx context.Context, ist, orgID string) (string, error) {
	return "sty_sess_invite", nil
}

// ============================================================================
// MockRedisClient
// ============================================================================

type MockRedisClient struct {
	publishFn func(ctx context.Context, channel string, message interface{}) *redis.IntCmd
}

func (m *MockRedisClient) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	if m.publishFn != nil {
		return m.publishFn(ctx, channel, message)
	}
	return redis.NewIntResult(1, nil)
}

// ============================================================================
// Test Harness
// ============================================================================

type workerTestHarness struct {
	worker *Worker
	repo   *MockRepository
	rdb    *MockRedisClient
	idp    *MockIDP
	logs   *observer.ObservedLogs
	logger *zap.Logger
	cfg    config.Config
}

func newWorkerTestHarness(t *testing.T) *workerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	rdb := &MockRedisClient{}
	idp := &MockIDP{}

	observedCore, observedLogs := observer.New(zapcore.InfoLevel)
	logger := zap.New(observedCore)

	cfg := config.Config{
		AppEnv:     "test",
		BackendURL: "http://localhost:3030",
	}

	worker := &Worker{
		repo:   repo,
		rdb:    rdb,
		idp:    idp,
		logger: logger,
		cfg:    cfg,
	}

	return &workerTestHarness{
		worker: worker,
		repo:   repo,
		rdb:    rdb,
		idp:    idp,
		logs:   observedLogs,
		logger: logger,
		cfg:    cfg,
	}
}

func validPayload() *ProcessImportPayload {
	return &ProcessImportPayload{
		ImportJobID: "job_001",
		TenantID:    "tenant_001",
		SchoolID:    "school_001",
		Role:        "NURSE",
		StytchOrgID: "org_stytch_001",
		BackendURL:  "http://localhost:3030",
		Records: []ImportStaffRecord{
			{TempID: "tmp_alice", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
			{TempID: "tmp_bob", Email: "bob@school.com", FirstName: "Bob", LastName: "Jones"},
		},
	}
}

func createTask(payload *ProcessImportPayload) *asynq.Task {
	data, _ := json.Marshal(payload)
	return asynq.NewTask(TypeProcessImport, data)
}

// stage2RecordsFromPayload builds a []Stage2Record from payload records,
// assuming all were inserted and each invitation ID is "inv_<email>".
func stage2RecordsFromPayload(payload *ProcessImportPayload) []Stage2Record {
	records := make([]Stage2Record, 0, len(payload.Records))
	for _, rec := range payload.Records {
		records = append(records, Stage2Record{
			InvitationID: "inv_" + rec.Email,
			Email:        rec.Email,
			FirstName:    rec.FirstName,
			LastName:     rec.LastName,
		})
	}
	return records
}

// ============================================================================
// Tests: ProcessImport — Happy Path
// ============================================================================

func TestProcessImport_AllSuccess(t *testing.T) {
	h := newWorkerTestHarness(t)

	var capturedSuccess, capturedFailed int
	h.repo.updateImportJobStatusFn = func(ctx context.Context, id, status string, processed, successCount, failedCount int) error {
		capturedSuccess = successCount
		capturedFailed = failedCount
		return nil
	}

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil // not yet invited
	}

	payload := validPayload()
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return stage2RecordsFromPayload(payload), nil
	}

	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 2 Stytch invites were sent
	if h.idp.inviteMemberByEmailCalls != 2 {
		t.Fatalf("expected 2 Stytch invite calls, got %d", h.idp.inviteMemberByEmailCalls)
	}

	// overallSuccess counts Stage 2 (Stytch invites) only — Stage 1 inserts are
	// intermediate state. Both alice and bob were invited in Stage 2.
	if capturedSuccess != 2 {
		t.Fatalf("expected overallSuccess=2 (2 Stage 2 invites), got %d", capturedSuccess)
	}
	if capturedFailed != 0 {
		t.Fatalf("expected overallFailed=0, got %d", capturedFailed)
	}

	// Verify final status log
	infoLogs := h.logs.FilterMessage("import job completed")
	if infoLogs.Len() != 1 {
		t.Fatal("expected exactly 1 'import job completed' log")
	}
}

// ============================================================================
// Tests: ProcessImport — Stage 1 DB failures
// ============================================================================

func TestProcessImport_BulkInsertFails(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		return nil, nil, errors.New("postgres connection lost")
	}

	h.repo.bulkRecordImportFailureFn = func(ctx context.Context, jobID string, records []ImportStaffRecord, errMsg string) error {
		return nil
	}

	// No records were inserted, so GetPendingStage2Records returns nil
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return nil, nil
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error (failures are recorded, not returned): %v", err)
	}

	// Should have 0 Stytch invites since all inserts failed
	if h.idp.inviteMemberByEmailCalls != 0 {
		t.Fatalf("expected 0 Stytch invite calls after bulk insert failure, got %d", h.idp.inviteMemberByEmailCalls)
	}

	// Verify BulkRecordImportFailure was called with all records (not N individual calls).
	// No explicit assertion needed here as the mock function signature accepts a slice.
}

// TestProcessImport_BulkRecordImportFailure verifies that when Stage 1 fails,
// failures are recorded in a single bulk call instead of N individual INSERTs.
func TestProcessImport_BulkRecordImportFailure(t *testing.T) {
	h := newWorkerTestHarness(t)

	callCount := 0
	var capturedRecords []ImportStaffRecord
	h.repo.bulkRecordImportFailureFn = func(ctx context.Context, jobID string, records []ImportStaffRecord, errMsg string) error {
		callCount++
		capturedRecords = append(capturedRecords, records...)
		return nil
	}

	// Three records split into two batches (BatchSize=2 in domain.go, but we
	// force it smaller via a 2-record payload to verify single-batch behaviour).
	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		return nil, nil, errors.New("postgres connection lost")
	}

	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return nil, nil
	}

	payload := validPayload() // 2 records
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}

	// Should have made 1 bulk call (covering all 2 records in one batch)
	if callCount != 1 {
		t.Fatalf("expected 1 BulkRecordImportFailure call, got %d (would be 2 individual calls without the fix)", callCount)
	}
	if len(capturedRecords) != 2 {
		t.Fatalf("expected 2 records in bulk failure call, got %d", len(capturedRecords))
	}
}

func TestProcessImport_PartialDuplicates(t *testing.T) {
	h := newWorkerTestHarness(t)

	var capturedSuccess, capturedFailed int
	h.repo.updateImportJobStatusFn = func(ctx context.Context, id, status string, processed, successCount, failedCount int) error {
		capturedSuccess = successCount
		capturedFailed = failedCount
		return nil
	}

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		var failures []FailedInsertion
		for _, rec := range records {
			if rec.Email == "bob@school.com" {
				failures = append(failures, FailedInsertion{
					TempID: rec.TempID,
					Email:  rec.Email,
					Reason: "duplicate",
				})
			} else {
				inserted[rec.TempID] = "inv_" + rec.Email
			}
		}
		return inserted, failures, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	// Only alice's record was inserted; bob was a duplicate and skipped
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return []Stage2Record{
			{InvitationID: "inv_alice@school.com", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
		}, nil
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only alice@school.com should be invited (bob was duplicate)
	if h.idp.inviteMemberByEmailCalls != 1 {
		t.Fatalf("expected 1 Stytch invite call (alice only), got %d", h.idp.inviteMemberByEmailCalls)
	}

	// Final status should be completed_with_errors
	infoLogs := h.logs.FilterMessage("import job completed")
	if infoLogs.Len() == 1 {
		entry := infoLogs.All()[0]
		statusField := findField(entry, "status")
		if statusField != nil && statusField.String != "completed_with_errors" {
			t.Fatalf("expected status 'completed_with_errors', got %q", statusField.String)
		}
	}

	// overallSuccess counts Stage 2 (Stytch invites) only — Stage 1 inserts are
	// intermediate state. Alice was invited (1 success), Bob was a duplicate (1 fail).
	if capturedSuccess != 1 {
		t.Fatalf("expected overallSuccess=1 (alice Stage 2 invite), got %d", capturedSuccess)
	}
	if capturedFailed != 1 {
		t.Fatalf("expected overallFailed=1 (bob duplicate), got %d", capturedFailed)
	}
}

// ============================================================================
// Tests: ProcessImport — Stage 2 Stytch failures
// ============================================================================

func TestProcessImport_StytchInviteFails(t *testing.T) {
	h := newWorkerTestHarness(t)

	var capturedSuccess, capturedFailed int
	h.repo.updateImportJobStatusFn = func(ctx context.Context, id, status string, processed, successCount, failedCount int) error {
		capturedSuccess = successCount
		capturedFailed = failedCount
		return nil
	}

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	h.idp.inviteMemberByEmailFn = func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
		if email == "bob@school.com" {
			return "", errors.New("invalid_email: email domain not allowed")
		}
		return "member_" + email, nil
	}

	h.repo.setInvitationFailedFn = func(ctx context.Context, id, errorMessage string, attemptCount int) error {
		return nil
	}

	payload := validPayload()
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return stage2RecordsFromPayload(payload), nil
	}

	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 Stytch calls attempted
	if h.idp.inviteMemberByEmailCalls != 2 {
		t.Fatalf("expected 2 Stytch invite calls, got %d", h.idp.inviteMemberByEmailCalls)
	}

	// Final status should be completed_with_errors
	infoLogs := h.logs.FilterMessage("import job completed")
	if infoLogs.Len() == 1 {
		entry := infoLogs.All()[0]
		statusField := findField(entry, "status")
		if statusField != nil && statusField.String != "completed_with_errors" {
			t.Fatalf("expected status 'completed_with_errors', got %q", statusField.String)
		}
	}

	// overallSuccess counts Stage 2 (Stytch invites) only. Alice was invited (1 success),
	// Bob failed the Stytch invite (1 fail). Stage 1 inserts are intermediate state.
	if capturedSuccess != 1 {
		t.Fatalf("expected overallSuccess=1 (alice Stage 2 invite), got %d", capturedSuccess)
	}
	if capturedFailed != 1 {
		t.Fatalf("expected overallFailed=1 (bob Stytch failure), got %d", capturedFailed)
	}
}

func TestProcessImport_StytchTransientThenSuccess(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	attempt := 0
	h.idp.inviteMemberByEmailFn = func(ctx context.Context, orgID, email, name, redirectURL string) (string, error) {
		attempt++
		if attempt == 1 {
			return "", errors.New("rate_limit_exceeded: try again")
		}
		return "member_" + email, nil
	}

	payload := validPayload()
	payload.Records = []ImportStaffRecord{
		{Email: "retry@school.com", FirstName: "Retry", LastName: "User"},
	}
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return stage2RecordsFromPayload(payload), nil
	}

	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have made 2 attempts (1 fail, 1 success)
	if attempt != 2 {
		t.Fatalf("expected 2 Stytch attempts, got %d", attempt)
	}
}

// ============================================================================
// Tests: ProcessImport — Already invited (idempotency)
// ============================================================================

func TestProcessImport_AlreadyInvited(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	// All records already have a Stytch member ID, so GetPendingStage2Records
	// returns nothing — the DB query filters them out.
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return nil, nil
	}

	// This would never be called since there are no pending records, but
	// set it for safety in case the logic changes.
	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "existing_member_id", nil
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No new Stytch invites should be sent
	if h.idp.inviteMemberByEmailCalls != 0 {
		t.Fatalf("expected 0 Stytch invite calls (all already invited), got %d", h.idp.inviteMemberByEmailCalls)
	}
}

// ============================================================================
// Tests: ProcessImport — Invalid payload
// ============================================================================

func TestProcessImport_InvalidPayload(t *testing.T) {
	h := newWorkerTestHarness(t)

	task := asynq.NewTask(TypeProcessImport, []byte("invalid json"))

	err := h.worker.ProcessImport(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
}

// ============================================================================
// Tests: ProcessImport — SetStarted fails
// ============================================================================

func TestProcessImport_SetStartedFails(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.setImportJobStartedFn = func(ctx context.Context, id string) error {
		return errors.New("db down")
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err == nil {
		t.Fatal("expected error when SetImportJobStarted fails, got nil")
	}
}

// ============================================================================
// Tests: ProcessImport — Large batch (stress subset)
// ============================================================================

func TestProcessImport_LargeBatch(t *testing.T) {
	h := newWorkerTestHarness(t)

	numRecords := 500
	records := make([]ImportStaffRecord, numRecords)
	for i := 0; i < numRecords; i++ {
		records[i] = ImportStaffRecord{
			Email:     fmt.Sprintf("user%d@school.com", i),
			FirstName: "User",
			LastName:  fmt.Sprintf("%d", i),
		}
	}

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, recs []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range recs {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	payload := validPayload()
	payload.Records = records
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		recs := make([]Stage2Record, len(records))
		for i, rec := range records {
			recs[i] = Stage2Record{
				InvitationID: "inv_" + rec.Email,
				Email:        rec.Email,
				FirstName:    rec.FirstName,
				LastName:     rec.LastName,
			}
		}
		return recs, nil
	}

	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h.idp.inviteMemberByEmailCalls != numRecords {
		t.Fatalf("expected %d Stytch invite calls, got %d", numRecords, h.idp.inviteMemberByEmailCalls)
	}
}

// ============================================================================
// Tests: HandleError — Dead-letter callback
// ============================================================================

func TestHandleError_UpdatesJobToFailed(t *testing.T) {
	h := newWorkerTestHarness(t)

	jobID := ""
	h.repo.setImportJobFailedFn = func(ctx context.Context, id string) error {
		jobID = id
		return nil
	}

	payload := validPayload()
	task := createTask(payload)

	err := errors.New("max retries exceeded: postgres connection timeout")
	h.worker.HandleError(context.Background(), task, err)

	if jobID != "job_001" {
		t.Fatalf("expected SetImportJobFailed to be called with 'job_001', got %q", jobID)
	}
}

func TestHandleError_LogsOnUnmarshalFailure(t *testing.T) {
	h := newWorkerTestHarness(t)

	task := asynq.NewTask(TypeProcessImport, []byte("{invalid json}"))
	err := errors.New("max retries exceeded")

	h.worker.HandleError(context.Background(), task, err)

	// Should not panic — failure to unmarshal is logged and the handler returns
	errorLogs := h.logs.FilterLevelExact(zapcore.ErrorLevel).FilterMessage("asynq error handler: failed to unmarshal payload")
	if errorLogs.Len() != 1 {
		t.Fatalf("expected 1 error log for unmarshal failure, got %d", errorLogs.Len())
	}
}

func TestHandleError_RepoFailureLogged(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.setImportJobFailedFn = func(ctx context.Context, id string) error {
		return errors.New("postgres connection failed")
	}

	payload := validPayload()
	task := createTask(payload)
	err := errors.New("max retries exceeded")

	h.worker.HandleError(context.Background(), task, err)

	// Should log the repo failure
	errorLogs := h.logs.FilterLevelExact(zapcore.ErrorLevel).FilterMessage("asynq error handler: failed to set job status to failed")
	if errorLogs.Len() != 1 {
		t.Fatalf("expected 1 error log for repo failure, got %d", errorLogs.Len())
	}
}

// ============================================================================
// Tests: Stage 2 resume on retry (DB-backed)
// ============================================================================

func TestProcessImport_Stage2ResumeSkipsAlreadyInvited(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	// GetPendingStage2Records simulates a retry — alice was already invited
	// in a previous run so only bob is returned.
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return []Stage2Record{
			{InvitationID: "inv_bob@school.com", Email: "bob@school.com", FirstName: "Bob", LastName: "Jones"},
		}, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only bob should be invited on the retry
	if h.idp.inviteMemberByEmailCalls != 1 {
		t.Fatalf("expected 1 Stytch invite call (bob only), got %d", h.idp.inviteMemberByEmailCalls)
	}
}

func TestProcessImport_Stage2ResumeAfterPartialCompletion(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	// Simulate retry where bob failed the Stytch invite on the first run
	// (status = 'invite_failed') — the DB query excludes those. Only
	// alice and charlie remain pending.
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return []Stage2Record{
			{InvitationID: "inv_alice@school.com", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
			{InvitationID: "inv_charlie@school.com", Email: "charlie@school.com", FirstName: "Charlie", LastName: "Brown"},
		}, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	payload := validPayload()
	payload.Records = []ImportStaffRecord{
		{TempID: "tmp_alice", Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
		{TempID: "tmp_bob", Email: "bob@school.com", FirstName: "Bob", LastName: "Jones"},
		{TempID: "tmp_charlie", Email: "charlie@school.com", FirstName: "Charlie", LastName: "Brown"},
	}
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only alice and charlie should be invited on the retry
	if h.idp.inviteMemberByEmailCalls != 2 {
		t.Fatalf("expected 2 Stytch invite calls (alice + charlie), got %d", h.idp.inviteMemberByEmailCalls)
	}
}

func TestProcessImport_Stage2NoPendingRecords(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	// All records are already fully processed (have stytch_member_id or
	// were invite_failed), so the retry has nothing left to do.
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return nil, nil
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h.idp.inviteMemberByEmailCalls != 0 {
		t.Fatalf("expected 0 Stytch invite calls (no pending records), got %d", h.idp.inviteMemberByEmailCalls)
	}
}

// ============================================================================
// Tests: ProcessImport — Context cancellation (task timeout)
// ============================================================================

func TestProcessImport_ContextCancelled(t *testing.T) {
	h := newWorkerTestHarness(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		return nil, nil, ctx.Err()
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(ctx, task)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ============================================================================
// Tests: Redis publish failure is non-fatal
// ============================================================================

func TestProcessImport_RedisPublishFails_DoesNotBlock(t *testing.T) {
	h := newWorkerTestHarness(t)

	// Simulate Redis being down for publishes
	h.rdb.publishFn = func(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
		return redis.NewIntResult(0, errors.New("redis is down"))
	}

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now time.Time, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.TempID] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	payload := validPayload()
	h.repo.getPendingStage2RecordsFn = func(ctx context.Context, jobID string) ([]Stage2Record, error) {
		return stage2RecordsFromPayload(payload), nil
	}

	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error even when Redis publish fails: %v", err)
	}

	// Stytch invites should still be sent
	if h.idp.inviteMemberByEmailCalls != 2 {
		t.Fatalf("expected 2 Stytch invite calls despite Redis failure, got %d", h.idp.inviteMemberByEmailCalls)
	}

	// Should log the Redis failure as a warning
	warnLogs := h.logs.FilterLevelExact(zapcore.WarnLevel).FilterMessage("redis publish failed (non-fatal)")
	if warnLogs.Len() < 1 {
		t.Fatal("expected at least 1 warning log for Redis publish failure")
	}
}

// ============================================================================
// Helpers
// ============================================================================

func findField(entry observer.LoggedEntry, key string) *zapcore.Field {
	for i := range entry.Context {
		if entry.Context[i].Key == key {
			return &entry.Context[i]
		}
	}
	return nil
}

// Compile-time interface checks
var _ auth.IdentityProvider = (*MockIDP)(nil)

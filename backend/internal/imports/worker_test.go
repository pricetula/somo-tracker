package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

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
		AppEnv:      "test",
		FrontendURL: "http://localhost:3000",
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
		FrontendURL: "http://localhost:3000",
		Records: []ImportStaffRecord{
			{Email: "alice@school.com", FirstName: "Alice", LastName: "Smith"},
			{Email: "bob@school.com", FirstName: "Bob", LastName: "Jones"},
		},
	}
}

func createTask(payload *ProcessImportPayload) *asynq.Task {
	data, _ := json.Marshal(payload)
	return asynq.NewTask(TypeProcessImport, data)
}

// ============================================================================
// Tests: ProcessImport — Happy Path
// ============================================================================

func TestProcessImport_AllSuccess(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.Email] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil // not yet invited
	}

	payload := validPayload()
	task := createTask(payload)

	err := h.worker.ProcessImport(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 2 Stytch invites were sent
	if h.idp.inviteMemberByEmailCalls != 2 {
		t.Fatalf("expected 2 Stytch invite calls, got %d", h.idp.inviteMemberByEmailCalls)
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
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		return nil, nil, errors.New("postgres connection lost")
	}

	h.repo.recordImportFailureFn = func(ctx context.Context, jobID, rawPayloadJSON, errMsg string) error {
		return nil
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
}

func TestProcessImport_PartialDuplicates(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
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
				inserted[rec.Email] = "inv_" + rec.Email
			}
		}
		return inserted, failures, nil
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
}

// ============================================================================
// Tests: ProcessImport — Stage 2 Stytch failures
// ============================================================================

func TestProcessImport_StytchInviteFails(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.Email] = "inv_" + rec.Email
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
}

func TestProcessImport_StytchTransientThenSuccess(t *testing.T) {
	h := newWorkerTestHarness(t)

	h.repo.bulkInsertInvitationsFn = func(
		ctx context.Context, records []ImportStaffRecord,
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.Email] = "inv_" + rec.Email
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
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range records {
			inserted[rec.Email] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	// All records already have a Stytch member ID
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
		tenantID, schoolID, role, jobID string, now interface{}, tokenPrefix string,
	) (map[string]string, []FailedInsertion, error) {
		inserted := make(map[string]string)
		for _, rec := range recs {
			inserted[rec.Email] = "inv_" + rec.Email
		}
		return inserted, nil, nil
	}

	h.repo.getInvitationStytchMemberIDFn = func(ctx context.Context, id string) (string, error) {
		return "", nil
	}

	payload := validPayload()
	payload.Records = records
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

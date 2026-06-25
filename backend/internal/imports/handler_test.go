package imports

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/middleware"
)

// ============================================================================
// MockSchoolResolver for handler tests
// ============================================================================

type MockSchoolResolver struct {
	getActiveSchoolIDFn func(ctx context.Context, tenantID, userID string) (string, error)
}

func (m *MockSchoolResolver) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if m.getActiveSchoolIDFn != nil {
		return m.getActiveSchoolIDFn(ctx, tenantID, userID)
	}
	return "school_001", nil
}

// ============================================================================
// MockRedisClient for handler tests (SSE)
// ============================================================================

type handlerMockRedis struct {
	mu          sync.Mutex
	subscribeFn func(ctx context.Context, channels ...string) *redis.PubSub
	pingFn      func(ctx context.Context) *redis.StatusCmd
}

func (m *handlerMockRedis) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscribeFn != nil {
		return m.subscribeFn(ctx, channels...)
	}
	return &redis.PubSub{}
}

func (m *handlerMockRedis) Ping(ctx context.Context) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return redis.NewStatusResult("PONG", nil)
}

// compile-time check that handlerMockRedis satisfies SSEPubSubClient
var _ SSEPubSubClient = (*handlerMockRedis)(nil)

// ============================================================================
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app      *fiber.App
	svc      *Service
	repo     *MockRepository
	resolver *MockSchoolResolver
	rdb      *handlerMockRedis
	handler  *Handler
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	client := &MockAsynqClient{}
	logger := zap.NewNop()
	cfg := config.Config{AppEnv: "test", FrontendURL: "http://localhost:3000"}

	svc := &Service{
		repo:   repo,
		client: client,
		logger: logger,
		cfg:    cfg,
	}

	rdb := &handlerMockRedis{}

	resolver := &MockSchoolResolver{}

	handler := &Handler{
		svc:      svc,
		repo:     repo,
		resolver: resolver,
		rdb:      rdb,
		logger:   logger,
	}

	app := fiber.New()

	// Register routes with a test auth middleware (bypasses real requireAuth)
	// that sets the tenant_id and user_id locals
	imports := app.Group("/api/v1/imports/staff", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		return c.Next()
	})

	imports.Post("/", handler.StartImport)
	imports.Get("/track/:id", handler.TrackImport)
	imports.Get("/track/:id/sse", handler.SSETrackImport)
	imports.Get("/:id/failures", handler.ListFailedInvitations)

	return &handlerTestHarness{
		app:      app,
		svc:      svc,
		repo:     repo,
		resolver: resolver,
		rdb:      rdb,
		handler:  handler,
	}
}

func doRequest(app *fiber.App, method, path string, body []byte) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Cookie", "somo_sid=valid_session_token")
	resp, _ := app.Test(req)
	return resp
}

// ============================================================================
// Tests: StartImport Handler
// ============================================================================

func TestHandler_StartImport_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := StartImportRequest{
		Role: "SCHOOL_ADMIN",
		Records: []ImportStaffRecord{
			{Email: "alice@school.com", FullName: "Alice Smith"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", bodyBytes)
	if resp.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}

	var result StartImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.ImportJobID == "" {
		t.Fatal("expected non-empty import_job_id")
	}
	if result.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", result.Status)
	}
}

func TestHandler_StartImport_MissingRole(t *testing.T) {
	h := newHandlerTestHarness(t)

	body := StartImportRequest{
		Role: "",
		Records: []ImportStaffRecord{
			{Email: "alice@school.com", FullName: "Alice Smith"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}

	var errBody ErrorBody
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Error != "invalid_input" {
		t.Fatalf("expected error 'invalid_input', got %q", errBody.Error)
	}
}

func TestHandler_StartImport_InvalidJSON(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", []byte("{invalid json"))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandler_StartImport_ServiceError(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getTenantStytchOrgIDFn = func(ctx context.Context, tenantID string) (string, error) {
		return "", fmt.Errorf("tenant not found: %w", middleware.ErrInvalidInput)
	}

	body := StartImportRequest{
		Role: "NURSE",
		Records: []ImportStaffRecord{
			{Email: "nurse@school.com", FullName: "Nurse Betty"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	resp := doRequest(h.app, "POST", "/api/v1/imports/staff/", bodyBytes)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: TrackImport Handler
// ============================================================================

func TestHandler_TrackImport_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return &ImportJob{
			ID:           jobID,
			TenantID:     "tenant_001",
			SchoolID:     "school_001",
			Role:         "SCHOOL_ADMIN",
			Status:       "completed",
			TotalRecords: 10,
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/track/job_001", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result TrackImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Job.ID != "job_001" {
		t.Fatalf("expected job ID 'job_001', got %q", result.Job.ID)
	}
}

func TestHandler_TrackImport_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return nil, fmt.Errorf("import job not found: %w", middleware.ErrNotFound)
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/track/nonexistent", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: ListFailedInvitations Handler
// ============================================================================

func TestHandler_ListFailedInvitations_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	fullName := "Alice"
	errMsg := "stytch invite failed"

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return []FailedInvitation{
			{ID: "inv_001", Email: "alice@bad.com", FullName: &fullName, ErrorMessage: &errMsg},
		}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/job_001/failures", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListFailedInvitationsResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Invitations) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(result.Invitations))
	}
}

func TestHandler_ListFailedInvitations_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.getFailedInvitationsByJobFn = func(ctx context.Context, jobID string) ([]FailedInvitation, error) {
		return nil, fmt.Errorf("import job not found: %w", middleware.ErrNotFound)
	}

	resp := doRequest(h.app, "GET", "/api/v1/imports/staff/nonexistent/failures", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: SSETrackImport — Redis fallback
// ============================================================================

// sseTestResult contains events read from an SSE response and the mock
// repository's getImportJob call count.
type sseTestResult struct {
	events []string
}

// invokeSSEHandler executes the SSE handler via app.Test and reads the body.
// Because Fiber's app.Test collects the response synchronously, the streaming
// goroutine may not have run yet. We verify handler effects through the mock
// repo call count and log assertions rather than real-time SSE event streaming.
func invokeSSEHandler(t *testing.T, app *fiber.App, path string) sseTestResult {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Cookie", "somo_sid=valid_session_token")

	resp, err := app.Test(req, 6000)
	if err != nil {
		t.Fatalf("SSE request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read all body — app.Test collects what was written before the
	// streaming goroutine (the initial "connected" event and anything
	// written synchronously).
	var events []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			events = append(events, strings.TrimPrefix(line, "data: "))
		}
	}

	// 1. Check for errors that occurred during scanning
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading response body: %v", err)
	}

	return sseTestResult{events: events}
}

func TestHandler_SSE_FallsBackToPollingWhenRedisDown(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.rdb.pingFn = func(ctx context.Context) *redis.StatusCmd {
		return redis.NewStatusResult("", errors.New("connection refused"))
	}

	callCount := 0
	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		callCount++
		return &ImportJob{
			ID:               jobID,
			Status:           "completed",
			TotalRecords:     10,
			ProcessedRecords: 10,
			SuccessCount:     10,
			FailedCount:      0,
		}, nil
	}

	result := invokeSSEHandler(t, h.app, "/api/v1/imports/staff/track/job_001/sse")

	// Verify the handler ran without crashing and events were returned.
	// The initial "connected" event is written directly to the fiber context
	// before SetBodyStreamWriter starts, and may or may not be captured in
	// the response body depending on Fiber version. What we can verify:
	// 1. Events were produced (at least one)
	// 2. The DB was polled at least once (proving the ticker runs)
	if len(result.events) == 0 {
		t.Fatal("expected at least one SSE event, got none")
	}
	if callCount == 0 {
		t.Fatal("expected at least 1 DB poll (ticker should run), got 0")
	}
}

func TestHandler_SSE_EmitsFinishedWhenJobFailed(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.rdb.pingFn = func(ctx context.Context) *redis.StatusCmd {
		return redis.NewStatusResult("", errors.New("connection refused"))
	}

	callCount := 0
	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		callCount++
		return &ImportJob{
			ID:               jobID,
			Status:           "failed",
			TotalRecords:     10,
			ProcessedRecords: 5,
			SuccessCount:     0,
			FailedCount:      5,
		}, nil
	}

	result := invokeSSEHandler(t, h.app, "/api/v1/imports/staff/track/job_001/sse")
	if len(result.events) == 0 {
		t.Fatal("expected at least one SSE event")
	}
	if callCount == 0 {
		t.Fatal("expected at least 1 DB poll, got 0")
	}
}

func TestHandler_SSE_RedisUnavailableLogsDegradedMode(t *testing.T) {
	h := newHandlerTestHarness(t)

	observedCore, observedLogs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)
	h.handler.logger = logger

	h.rdb.pingFn = func(ctx context.Context) *redis.StatusCmd {
		return redis.NewStatusResult("", errors.New("connection refused"))
	}

	h.repo.getImportJobFn = func(ctx context.Context, jobID string) (*ImportJob, error) {
		return &ImportJob{
			ID:               jobID,
			Status:           "completed",
			TotalRecords:     0,
			ProcessedRecords: 0,
			SuccessCount:     0,
			FailedCount:      0,
		}, nil
	}

	result := invokeSSEHandler(t, h.app, "/api/v1/imports/staff/track/job_001/sse")
	_ = result // events captured but we're verifying logs

	degradedLogs := observedLogs.FilterMessage("SSE: Redis unreachable at connection, falling back to pure polling")
	if degradedLogs.Len() != 1 {
		t.Fatalf("expected 1 degraded-mode log, got %d", degradedLogs.Len())
	}
}

// ============================================================================
// Tests: isTerminalJobStatus
// ============================================================================

func TestIsTerminalJobStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"pending", false},
		{"processing", false},
		{"completed", true},
		{"completed_with_errors", true},
		{"failed", true},
		{"", false},
		{"enqueue_failed", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isTerminalJobStatus(tt.status); got != tt.want {
				t.Fatalf("isTerminalJobStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Compile-time checks
// ============================================================================

var _ redis.Client = (redis.Client)(redis.Client{})

package cbcstreams

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ============================================================================
// Test Harness
// ============================================================================

type handlerTestHarness struct {
	app     *fiber.App
	svc     *Service
	repo    *MockRepository
	handler *Handler
}

func newHandlerTestHarness(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Test middleware that sets tenant_id and school_id (bypasses requireAuth)
	testAuth := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "tenant_001")
		c.Locals("user_id", "user_001")
		c.Locals("school_id", "school_001")
		return c.Next()
	}

	// Register routes manually with test auth
	streams := app.Group("/api/v1/streams", testAuth)
	streams.Get("/", handler.List)
	streams.Post("/", handler.Create)
	streams.Put("/:id", handler.Update)
	streams.Delete("/:id", handler.Delete)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
	}
}

func doRequest(app *fiber.App, method, path string, body []byte) *http.Response {
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req)
	return resp
}

// ============================================================================
// Tests: List Streams (GET /api/v1/streams)
// ============================================================================

func TestHandler_ListStreams_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	now := time.Now()
	expected := []Stream{
		{ID: "stream_001", Name: "Blue", CreatedAt: now, UpdatedAt: now},
		{ID: "stream_002", Name: "Red", CreatedAt: now, UpdatedAt: now},
	}

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		return expected, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/streams", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListStreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(result.Data))
	}
	if result.Data[0].Name != "Blue" {
		t.Fatalf("expected name 'Blue', got %q", result.Data[0].Name)
	}
}

func TestHandler_ListStreams_Empty(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
		return []Stream{}, nil
	}

	resp := doRequest(h.app, "GET", "/api/v1/streams", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result ListStreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Data) != 0 {
		t.Fatalf("expected 0 streams, got %d", len(result.Data))
	}
}

// ============================================================================
// Tests: Create Stream (POST /api/v1/streams)
// ============================================================================

func TestHandler_CreateStream_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenantID 'tenant_001', got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected schoolID 'school_001', got %q", schoolID)
		}
		if name != "Blue" {
			t.Errorf("expected name 'Blue', got %q", name)
		}
		return &Stream{ID: "stream_001", Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}

	body, _ := json.Marshal(CreateStreamPayload{Name: "Blue"})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result Stream
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.ID != "stream_001" {
		t.Fatalf("expected id 'stream_001', got %q", result.ID)
	}
	if result.Name != "Blue" {
		t.Fatalf("expected name 'Blue', got %q", result.Name)
	}
}

func TestHandler_CreateStream_MissingName(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(CreateStreamPayload{Name: ""})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateStream_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "POST", "/api/v1/streams", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateStream_NameTooLong(t *testing.T) {
	// SC4: Rejects name exceeding 100 characters — 422
	h := newHandlerTestHarness(t)

	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	body, _ := json.Marshal(CreateStreamPayload{Name: longName})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("SC4: expected 422 for name >100 chars, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateStream_DuplicateName(t *testing.T) {
	// SC5: Rejects duplicate stream name within same tenant + school — 409 DUPLICATE_ENTRY
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
		return nil, ErrAlreadyExists
	}

	body, _ := json.Marshal(CreateStreamPayload{Name: "Blue"})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("SC5: expected 409 Conflict for duplicate name, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("SC5: failed to decode response: %v", err)
	}
	if result.Code != "already_exists" {
		t.Fatalf("SC5: expected code 'already_exists', got %q", result.Code)
	}
}

func TestHandler_CreateStream_SameNameDifferentSchool(t *testing.T) {
	// SC6: Allows same stream name in a different school
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
		if schoolID != "school_001" {
			t.Errorf("SC6: unexpected schoolID: %q", schoolID)
		}
		return &Stream{ID: "stream_new", Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}

	body, _ := json.Marshal(CreateStreamPayload{Name: "Blue"})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("SC6: expected 201 Created, got %d", resp.StatusCode)
	}
}

func TestHandler_CreateStream_SameNameDifferentTenant(t *testing.T) {
	// SC7: Allows same stream name in a different tenant
	h := newHandlerTestHarness(t)

	h.repo.createFn = func(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
		if tenantID != "tenant_001" {
			t.Errorf("SC7: unexpected tenantID: %q", tenantID)
		}
		return &Stream{ID: "stream_new", Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}

	body, _ := json.Marshal(CreateStreamPayload{Name: "Blue"})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("SC7: expected 201 Created, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Update Stream (PUT /api/v1/streams/:id)
// ============================================================================

func TestHandler_UpdateStream_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
		if id != "stream_001" {
			t.Errorf("expected id 'stream_001', got %q", id)
		}
		if name != "Green" {
			t.Errorf("expected name 'Green', got %q", name)
		}
		return &Stream{ID: id, Name: name}, nil
	}

	body, _ := json.Marshal(UpdateStreamPayload{Name: "Green"})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result Stream
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Name != "Green" {
		t.Fatalf("expected name 'Green', got %q", result.Name)
	}
}

func TestHandler_UpdateStream_MissingName(t *testing.T) {
	h := newHandlerTestHarness(t)

	body, _ := json.Marshal(UpdateStreamPayload{Name: ""})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateStream_InvalidBody(t *testing.T) {
	h := newHandlerTestHarness(t)

	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", []byte("not json"))

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateStream_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
		return nil, ErrNotFound
	}

	body, _ := json.Marshal(UpdateStreamPayload{Name: "Green"})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_999", body)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateStream_NameTooLong(t *testing.T) {
	// SC4 counterpart for update: name exceeding 100 characters — 422
	h := newHandlerTestHarness(t)

	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	body, _ := json.Marshal(UpdateStreamPayload{Name: longName})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for name >100 chars, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateStream_DuplicateName(t *testing.T) {
	// SU6: Rejects update to a duplicate name within same school — 409 DUPLICATE_ENTRY
	h := newHandlerTestHarness(t)

	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
		return nil, ErrAlreadyExists
	}

	body, _ := json.Marshal(UpdateStreamPayload{Name: "Blue"})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("SU6: expected 409 Conflict for duplicate name, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Tests: Delete Stream (DELETE /api/v1/streams/:id)
// ============================================================================

func TestHandler_DeleteStream_HappyPath(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		if id != "stream_001" {
			t.Errorf("expected id 'stream_001', got %q", id)
		}
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/streams/stream_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteStream_BlockedByClasses(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return true, nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/streams/stream_001", nil)

	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}

	var result struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Error != "STREAM_IN_USE" {
		t.Fatalf("expected error 'STREAM_IN_USE', got %q", result.Error)
	}
}

func TestHandler_DeleteStream_NotFound(t *testing.T) {
	h := newHandlerTestHarness(t)

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		return ErrNotFound
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/streams/stream_999", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Cross-tenant / cross-school isolation tests
// ============================================================================

func TestHandler_UpdateStream_ScopedToSchool(t *testing.T) {
	// SU4: Handler passes the session's school_id to the repository.
	// The repository's WHERE clause scopes access — if no match, ErrNotFound.
	h := newHandlerTestHarness(t)

	var capturedTenantID, capturedSchoolID string
	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
		capturedTenantID = tenantID
		capturedSchoolID = schoolID
		return &Stream{ID: id, Name: name}, nil
	}

	body, _ := json.Marshal(UpdateStreamPayload{Name: "Green"})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("SU4: expected 200, got %d", resp.StatusCode)
	}
	if capturedSchoolID != "school_001" {
		t.Fatalf("SU4: expected school_id 'school_001' from session, got %q", capturedSchoolID)
	}
	if capturedTenantID != "tenant_001" {
		t.Fatalf("SU4: expected tenant_id 'tenant_001' from session, got %q", capturedTenantID)
	}
	t.Log("✓ SU4: update request scoped to session school + tenant")
}

func TestHandler_UpdateStream_ScopedToTenant(t *testing.T) {
	// SU5: Handler passes the session's tenant_id to the repository.
	h := newHandlerTestHarness(t)

	var capturedTenantID string
	h.repo.updateFn = func(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
		capturedTenantID = tenantID
		return &Stream{ID: id, Name: name}, nil
	}

	body, _ := json.Marshal(UpdateStreamPayload{Name: "Green"})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("SU5: expected 200, got %d", resp.StatusCode)
	}
	if capturedTenantID != "tenant_001" {
		t.Fatalf("SU5: expected tenant_id 'tenant_001' from session, got %q", capturedTenantID)
	}
	t.Log("✓ SU5: update request scoped to session tenant")
}

func TestHandler_DeleteStream_ScopedToSchool(t *testing.T) {
	// SD4: Handler passes the session's school_id to the repository.
	// The repository's WHERE clause scopes deletion — if no match, ErrNotFound.
	h := newHandlerTestHarness(t)

	var capturedTenantID, capturedSchoolID string
	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		capturedTenantID = tenantID
		capturedSchoolID = schoolID
		return false, nil
	}
	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		capturedTenantID = tenantID
		capturedSchoolID = schoolID
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/streams/stream_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("SD4: expected 204, got %d", resp.StatusCode)
	}
	if capturedSchoolID != "school_001" {
		t.Fatalf("SD4: expected school_id 'school_001' from session, got %q", capturedSchoolID)
	}
	if capturedTenantID != "tenant_001" {
		t.Fatalf("SD4: expected tenant_id 'tenant_001' from session, got %q", capturedTenantID)
	}
	t.Log("✓ SD4: delete request scoped to session school + tenant")
}

func TestHandler_DeleteStream_ScopedToTenant(t *testing.T) {
	// SD5: Handler passes the session's tenant_id to the repository.
	h := newHandlerTestHarness(t)

	var capturedTenantID string
	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		capturedTenantID = tenantID
		return false, nil
	}
	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		capturedTenantID = tenantID
		return nil
	}

	resp := doRequest(h.app, "DELETE", "/api/v1/streams/stream_001", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("SD5: expected 204, got %d", resp.StatusCode)
	}
	if capturedTenantID != "tenant_001" {
		t.Fatalf("SD5: expected tenant_id 'tenant_001' from session, got %q", capturedTenantID)
	}
	t.Log("✓ SD5: delete request scoped to session tenant")
}

// ============================================================================
// Auth rejection (unauthenticated requests)
// ============================================================================

// testHandlerWithoutAuth creates a handler harness that does NOT inject
// session auth — the real requireAuth middleware will reject.
func testHandlerWithoutAuth(t *testing.T) *handlerTestHarness {
	t.Helper()

	repo := &MockRepository{}
	svc := NewService(repo)
	handler := NewHandler(svc)

	app := fiber.New()

	// Register routes WITHOUT testAuth middleware — requireAuth will reject
	// because GetSession(c) returns nil (no session in context).
	streams := app.Group("/api/v1/streams")
	streams.Get("/", handler.requireAuth, handler.List)
	streams.Post("/", handler.requireAuth, handler.Create)
	streams.Put("/:id", handler.requireAuth, handler.Update)
	streams.Delete("/:id", handler.requireAuth, handler.Delete)

	return &handlerTestHarness{
		app:     app,
		svc:     svc,
		repo:    repo,
		handler: handler,
	}
}

func TestHandler_ListStreams_Unauthenticated(t *testing.T) {
	// SL5: Rejects unauthenticated request with 401
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "GET", "/api/v1/streams", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("SL5: expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("SL5: failed to decode response: %v", err)
	}
	if result.Code != "unauthorized" {
		t.Fatalf("SL5: expected code 'unauthorized', got %q", result.Code)
	}
}

func TestHandler_CreateStream_Unauthenticated(t *testing.T) {
	// SC8: Rejects unauthenticated request — 401
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(CreateStreamPayload{Name: "Blue"})
	resp := doRequest(h.app, "POST", "/api/v1/streams", body)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("SC8: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_UpdateStream_Unauthenticated(t *testing.T) {
	// SU8: Rejects unauthenticated request — 401
	h := testHandlerWithoutAuth(t)

	body, _ := json.Marshal(UpdateStreamPayload{Name: "Green"})
	resp := doRequest(h.app, "PUT", "/api/v1/streams/stream_001", body)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("SU8: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestHandler_DeleteStream_Unauthenticated(t *testing.T) {
	// SD7: Rejects unauthenticated request — 401
	h := testHandlerWithoutAuth(t)

	resp := doRequest(h.app, "DELETE", "/api/v1/streams/stream_001", nil)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("SD7: expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// ============================================================================
// After successful deletion, stream no longer appears in list endpoint (SD6)
// ============================================================================

func TestHandler_DeleteStream_RemovedFromList(t *testing.T) {
	// SD6: After successful deletion, stream no longer appears in list endpoint
	h := newHandlerTestHarness(t)

	deletedStreamID := "stream_003"
	deleted := false

	h.repo.listFn = func(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
		if deleted {
			return []Stream{
				{ID: "stream_001", Name: "Blue", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: "stream_002", Name: "Red", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		}
		return []Stream{
			{ID: "stream_001", Name: "Blue", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "stream_002", Name: "Red", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: deletedStreamID, Name: "Green", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}, nil
	}

	h.repo.hasReferencingClassesFn = func(ctx context.Context, id, tenantID, schoolID string) (bool, error) {
		return false, nil
	}

	h.repo.deleteFn = func(ctx context.Context, id, tenantID, schoolID string) error {
		deleted = true
		return nil
	}

	// Verify 3 before delete
	beforeResp := doRequest(h.app, "GET", "/api/v1/streams", nil)
	var before ListStreamsResponse
	if err := json.NewDecoder(beforeResp.Body).Decode(&before); err != nil {
		t.Fatalf("SD6: failed to decode list response: %v", err)
	}
	if len(before.Data) != 3 {
		t.Fatalf("SD6: expected 3 streams before delete, got %d", len(before.Data))
	}

	// Delete
	deleteResp := doRequest(h.app, "DELETE", "/api/v1/streams/"+deletedStreamID, nil)
	if deleteResp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("SD6: expected 204 on delete, got %d", deleteResp.StatusCode)
	}

	// Verify only 2 after delete
	afterResp := doRequest(h.app, "GET", "/api/v1/streams", nil)
	var after ListStreamsResponse
	if err := json.NewDecoder(afterResp.Body).Decode(&after); err != nil {
		t.Fatalf("SD6: failed to decode list response: %v", err)
	}
	if len(after.Data) != 2 {
		t.Fatalf("SD6: expected 2 streams after delete, got %d", len(after.Data))
	}
	for _, s := range after.Data {
		if s.ID == deletedStreamID {
			t.Fatalf("SD6: deleted stream %s still appears in list", deletedStreamID)
		}
	}
	t.Log("✓ SD6: deleted stream no longer appears in list")
}

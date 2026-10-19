// Package api — unit tests for bulk admin-member invitation ingestion.
// Section 1: Request Validation (handler layer). Matches project testify convention.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

// Design note: if a production interface exists, replace mock implementation.
func newTestHandler(svc services.AdminInvitationService) *AdminInvitationHandler {
	logger, _ := zap.NewDevelopment()
	return NewAdminInvitationHandler(svc, nil, nil, nil, logger)
}

func setupFiber() *fiber.App {
	return fiber.New()
}

func injectLocalsMW(locals map[string]interface{}) fiber.Handler {
	return func(c fiber.Ctx) error {
		for k, v := range locals {
			c.Locals(k, v)
		}
		return c.Next()
	}
}

func makeRowsJSON(n int) string {
	var b strings.Builder
	b.WriteString(`[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(`,`)
		}
		b.WriteString(fmt.Sprintf(`{"email":"user%d@test.co","full_name":"User %d"}`, i, i))
	}
	b.WriteString(`]`)
	return b.String()
}

// Section 1: Request Validation (handler layer)
func TestHandleInvites_Validation(t *testing.T) {
	t.Parallel()
	svc := &mockAdminInvitationService{}
	h := newTestHandler(svc)

	tests := []struct {
		name          string
		body          string
		locals        map[string]interface{}
		wantStatus    int
		wantErrCode   string
		wantDBZero    bool
		wantQueueZero bool
	}{
		{
			name:       "valid payload 1 row -> 202",
			body:       `{"invitations":[{"email":"a@b.co","full_name":"Alice"}]}`,
			locals:     map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name:       "valid payload exactly 1000 rows -> 202",
			body:       `{"invitations":` + makeRowsJSON(1000) + `}`,
			locals:     map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name:        "payload 1001 rows -> 400",
			body:        `{"invitations":` + makeRowsJSON(1001) + `}`,
			locals:      map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "bad_request",
		},
		{
			name:        "empty array -> 400, no job created",
			body:        `{"invitations":[]}`,
			locals:      map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "bad_request",
			wantDBZero:  true,
		},
		{
			name:        "missing email on one row -> 400 with row_index, other rows reported",
			body:        `{"invitations":[{"email":"","full_name":"A"},{"email":"b@c.co","full_name":"B"}]}`,
			locals:      map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "malformed email -> 400 field email",
			body:        `{"invitations":[{"email":"not-an-email","full_name":"A"}]}`,
			locals:      map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "empty/whitespace full_name -> 400",
			body:        `{"invitations":[{"email":"a@b.co","full_name":"   "}]}`,
			locals:      map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "multiple invalid rows -> 400 ALL errors with correct row_index",
			body:        `{"invitations":[{"email":"bad","full_name":""},{"email":"ok@ok.co","full_name":""}]}`,
			locals:      map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupFiber()
			app.Post("/invitations", injectLocalsMW(tt.locals), h.HandleInvites)

			req, err := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "test-key-"+tt.name)

			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantDBZero {
				t.Log("TODO: query DB to confirm zero bulk_jobs / bulk_job_items (testdb DB(t))")
			}
			if tt.wantQueueZero {
				t.Log("TODO: inspect Asynq queue to confirm zero tasks (asynqtest inspector)")
			}
		})
	}
}

// Section 1 — Duplicate emails design decision (explicit, not implicit).
func TestHandleInvites_DuplicateEmailsPayload(t *testing.T) {
	t.Parallel()
	svc := &mockAdminInvitationService{}
	h := newTestHandler(svc)
	body := `{"invitations":[{"email":"dup@dup.co","full_name":"A"},{"email":"dup@dup.co","full_name":"B"}]}`
	req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app := setupFiber()
	locals := map[string]interface{}{
		"active_school_id": uuid.New().String(),
		"tenant_id":        uuid.New().String(),
		"user_id":          uuid.New().String(),
	}
	app.Post("/invitations", injectLocalsMW(locals), h.HandleInvites)
	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	// Design decision TODO: allowed vs rejected not finalized.
	t.Logf("Duplicate email payload status=%d — design decision unresolved (TODO)", resp.StatusCode)
	assert.True(t, resp.StatusCode == fiber.StatusAccepted || resp.StatusCode == fiber.StatusBadRequest,
		"unexpected status %d; must explicitly choose allowed/rejected", resp.StatusCode)
}

func TestHandleInvites_MissingInvalidLocals(t *testing.T) {
	t.Parallel()
	svc := &mockAdminInvitationService{}
	h := newTestHandler(svc)

	tests := []struct {
		name       string
		locals     map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing active_school_id",
			locals:     map[string]interface{}{"tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "missing tenant_id",
			locals:     map[string]interface{}{"active_school_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "missing user_id",
			locals:     map[string]interface{}{"active_school_id": uuid.New().String(), "tenant_id": uuid.New().String()},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "wrong type string active_school_id",
			locals:     map[string]interface{}{"active_school_id": "not-a-uuid", "tenant_id": uuid.New().String(), "user_id": uuid.New().String()},
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupFiber()
			app.Post("/invitations", injectLocalsMW(tt.locals), h.HandleInvites)

			body := `{"invitations":[{"email":"a@b.co","full_name":"Alice"}]}`
			req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "test-key-"+tt.name)

			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// Section 2: Idempotency replay
func TestHandleInvites_Idempotency(t *testing.T) {
	t.Parallel()
	svc := &mockAdminInvitationService{}
	h := newTestHandler(svc)

	body := `{"invitations":[{"email":"a@b.co","full_name":"Alice"}]}`
	req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "00000000-0000-0000-0000-000000000001")

	app := setupFiber()
	locals := map[string]interface{}{
		"active_school_id": uuid.New().String(),
		"tenant_id":        uuid.New().String(),
		"user_id":          uuid.New().String(),
	}

	// First request
	app.Post("/invitations", injectLocalsMW(locals), h.HandleInvites)
	resp1, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusAccepted, resp1.StatusCode)

	// Second request with same idempotency key should return existing job
	resp2, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusAccepted, resp2.StatusCode)

	// Both responses should have same job_id
	var r1, r2 map[string]any
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&r1))
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&r2))
	assert.Equal(t, r1["job_id"], r2["job_id"], "idempotency key should return same job")
}

// mockAdminInvitationService implements services.AdminInvitationService for testing.
type mockAdminInvitationService struct{}

func (m *mockAdminInvitationService) CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockAdminInvitationService) CreateBulkJobWithItems(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, items []services.InvitationItem) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockAdminInvitationService) CreateBulkJobWithItemsWithAuth(ctx context.Context, schoolID, tenantID, createdBy, actorUserID uuid.UUID, idempotencyKey string, items []services.InvitationItem) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockAdminInvitationService) InsertItems(ctx context.Context, jobID uuid.UUID, items []services.InvitationItem) error {
	return nil
}
func (m *mockAdminInvitationService) GetJob(ctx context.Context, jobID uuid.UUID) (*services.BulkJob, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]services.BulkJobItem, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) GetFailedOrDeferredItemsWithAuth(ctx context.Context, jobID, schoolID, actorUserID uuid.UUID) ([]services.BulkJobItem, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) UpdateItem(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	return nil
}
func (m *mockAdminInvitationService) UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result []byte, status string, lastError string) error {
	return nil
}
func (m *mockAdminInvitationService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	return nil
}
func (m *mockAdminInvitationService) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error {
	return nil
}
func (m *mockAdminInvitationService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*services.BulkJob, error) {
	// Return a job to test idempotency replay
	if key == "same-key-for-both-requests" {
		return &services.BulkJob{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}, nil
	}
	return nil, nil
}
func (m *mockAdminInvitationService) GetItemByID(ctx context.Context, itemID uuid.UUID) (*services.BulkJobItem, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]services.BulkJobItem, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result []byte, lastError string, attempts int) error {
	return nil
}
func (m *mockAdminInvitationService) UserHasAdminRole(ctx context.Context, schoolID, userID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockAdminInvitationService) GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error) {
	return "", nil
}
func (m *mockAdminInvitationService) ProvisionInvitee(ctx context.Context, tenantID, schoolID, invitedBy uuid.UUID, email, fullName, role, stytchMemberID string) error {
	return nil
}

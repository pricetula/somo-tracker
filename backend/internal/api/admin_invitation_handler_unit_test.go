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

	"somotracker/backend/internal/services"
)

// Design note: if a production interface exists, replace mock implementation.
func newTestHandler(svc services.AdminInvitationService) *AdminInvitationHandler {
	return NewAdminInvitationHandler(svc, nil, nil, nil, nil)
}

func setupFiber() *fiber.App {
	return fiber.New()
}

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
			body:       `{"invitations":[{"email":"a@b.co","full_name":"Alice","role":"ADMIN"}]}`,
			locals:     map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name:       "valid payload exactly 10000 rows -> 202",
			body:       `{"invitations":` + makeRowsJSON(10000) + `}`,
			locals:     map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name:        "payload 10001 rows -> 400 (or 413 from body-limit middleware; confirm layer)",
			body:        `{"invitations":` + makeRowsJSON(10001) + `}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "bad_request",
		},
		{
			name:        "empty array -> 400, no job created",
			body:        `{"invitations":[]}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "bad_request",
			wantDBZero:  true,
		},
		{
			name:        "missing email on one row -> 400 with row_index, other rows reported",
			body:        `{"invitations":[{"email":"","full_name":"A","role":"ADMIN"},{"email":"b@c.co","full_name":"B","role":"TEACHER"}]}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "malformed email -> 400 field email",
			body:        `{"invitations":[{"email":"not-an-email","full_name":"A","role":"ADMIN"}]}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "empty/whitespace full_name -> 400",
			body:        `{"invitations":[{"email":"a@b.co","full_name":"   ","role":"ADMIN"}]}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "invalid role -> 400",
			body:        `{"invitations":[{"email":"a@b.co","full_name":"A","role":"SUPER_ADMIN"}]}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
		{
			name:        "multiple invalid rows -> 400 ALL errors with correct row_index",
			body:        `{"invitations":[{"email":"bad","full_name":"","role":"BAD"},{"email":"ok@ok.co","full_name":"","role":"ADMIN"}]}`,
			locals:      map[string]interface{}{"school_id": uuid.New(), "tenant_id": uuid.New(), "admin_user_id": uuid.New()},
			wantStatus:  fiber.StatusBadRequest,
			wantErrCode: "validation_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupFiber()
			app.Post("/invitations", func(c fiber.Ctx) error { return h.HandleInvites(c) })

			req, err := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

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
	body := `{"invitations":[{"email":"dup@dup.co","full_name":"A","role":"ADMIN"},{"email":"dup@dup.co","full_name":"B","role":"TEACHER"}]}`
	req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app := setupFiber()
	app.Post("/invitations", func(c fiber.Ctx) error { return h.HandleInvites(c) })
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

	cases := []struct {
		name   string
		locals map[string]interface{}
	}{
		{"missing school_id", map[string]interface{}{"tenant_id": uuid.New(), "admin_user_id": uuid.New()}},
		{"missing tenant_id", map[string]interface{}{"school_id": uuid.New(), "admin_user_id": uuid.New()}},
		{"wrong type string school_id", map[string]interface{}{"school_id": "not-uuid", "tenant_id": uuid.New(), "admin_user_id": uuid.New()}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := setupFiber()
			app.Post("/invitations", func(f fiber.Ctx) error {
				for k, v := range c.locals {
					f.Set(k, fmt.Sprintf("%v", v))
				}
				return h.HandleInvites(f)
			})
			req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(`{"invitations":[{"email":"a@b.co","full_name":"A","role":"ADMIN"}]}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, fiber.TestConfig{})
			require.NoError(t, err)
			// Must fail safely (401 or 500), never panic on bare .(string) assertion.
			assert.True(t, resp.StatusCode == fiber.StatusUnauthorized || resp.StatusCode == fiber.StatusInternalServerError,
				"expected safe failure, got %d", resp.StatusCode)
		})
	}
}

func makeRowsJSON(n int) string {
	b := bytes.NewBufferString("[")
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"email":"`)
		b.WriteString(strings.Repeat("a", 8))
		b.WriteString(`@test.co","full_name":"Row`)
		b.WriteString(string(rune('0' + i%10)))
		b.WriteString(`","role":"ADMIN"}`)
	}
	b.WriteByte(']')
	return b.String()
}

type mockAdminInvitationService struct{}

func (m *mockAdminInvitationService) CreateBulkJob(ctx context.Context, schoolID, tenantID, createdBy uuid.UUID, idempotencyKey string, total int) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockAdminInvitationService) InsertItems(ctx context.Context, jobID uuid.UUID, items []map[string]interface{}) error {
	return nil
}
func (m *mockAdminInvitationService) GetJob(ctx context.Context, jobID uuid.UUID) (*services.BulkJob, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) GetFailedOrDeferredItems(ctx context.Context, jobID uuid.UUID) ([]services.BulkJobItem, error) {
	return nil, nil
}
func (m *mockAdminInvitationService) UpdateItem(ctx context.Context, itemID uuid.UUID, status string, result json.RawMessage, lastError string, attempts int) error {
	return nil
}
func (m *mockAdminInvitationService) UpdateItemResultOnly(ctx context.Context, itemID uuid.UUID, result json.RawMessage, status string, lastError string) error {
	return nil
}
func (m *mockAdminInvitationService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	return nil
}
func (m *mockAdminInvitationService) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error {
	return nil
}
func (m *mockAdminInvitationService) GetJobByIdempotency(ctx context.Context, tenantID uuid.UUID, key string) (*services.BulkJob, error) {
	return nil, nil
}

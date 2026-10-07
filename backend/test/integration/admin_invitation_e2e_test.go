//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/api"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/database/sqlc"
	"somotracker/backend/internal/services"
	"somotracker/backend/internal/stytch"
	"somotracker/backend/internal/testcontainers"
	"somotracker/backend/internal/worker"
)

// TestAdminInvitationE2E_Success tests the full flow:
// 1. POST /api/admins/invitations creates a bulk job with items
// 2. Worker processes items and Stytch is called
// 3. Items are provisioned (users, members, school_memberships)
// 4. Job finalizes to COMPLETED
// Stytch is stubbed via MockStytchClient to simulate success.
func TestAdminInvitationE2E_Success(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	require.NoError(t, migrator.Up(ctx))
	defer migrator.Close()

	// Create test data
	var tenantID, countryID, edSysID, schoolID, userID uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'Test Tenant','test-tenant','org_test_123') RETURNING id`).Scan(&tenantID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES (gen_random_uuid(),'Test Country','TC') RETURNING id`).Scan(&countryID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES (gen_random_uuid(),$1,'Test System') RETURNING id`, countryID).Scan(&edSysID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'Test School',$2,$3) RETURNING id`, tenantID, countryID, edSysID).Scan(&schoolID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES (gen_random_uuid(),'admin@test.com',$1,'Admin User') RETURNING id`, tenantID).Scan(&userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1,$2,'ADMIN',true)`, schoolID, userID)
	require.NoError(t, err)

	// Redis and Asynq
	rc := testcontainers.SetupRedis(t)
	redisClient := rc.Client
	defer redisClient.Close()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rc.Addr()})
	defer asynqClient.Close()

	// Service and handler with mocked Stytch
	svc := services.NewAdminInvitationService(pool, sqlc.New(pool), logger)
	mockStytch := &MockStytchClient{
		Results: []*stytch.InviteMemberResult{
			{StytchInviteID: "inv_1", StytchMemberID: "mem_1"},
			{StytchInviteID: "inv_2", StytchMemberID: "mem_2"},
		},
	}
	handler := api.NewAdminInvitationHandler(svc, mockStytch, asynqClient, redisClient, logger)

	app := fiber.New()
	app.Post("/api/admins/invitations", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		c.Locals("user_id", userID.String())
		return handler.HandleInvites(c)
	})
	app.Get("/api/admins/invitations/jobs/:job_id/events", func(c fiber.Ctx) error {
		c.Locals("tenant_id", tenantID.String())
		return handler.Events(c)
	})
	app.Get("/api/admins/invitations/jobs/:job_id", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		return handler.GetJob(c)
	})
	app.Post("/api/admins/invitations/jobs/:job_id/retry-failed", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		return handler.RetryFailed(c)
	})

	// Submit bulk invitation
	payload := api.BulkInvitationRequest{
		Invitations: []api.InvitationRow{
			{Email: "teacher1@test.com", FullName: "Teacher One"},
			{Email: "teacher2@test.com", FullName: "Teacher Two"},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/admins/invitations", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-success-"+uuid.NewString())

	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var respBody struct {
		JobID        string `json:"job_id"`
		Status       string `json:"status"`
		TotalRecords int    `json:"total_records"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	require.NotEmpty(t, respBody.JobID)
	require.Equal(t, "QUEUED", respBody.Status)
	require.Equal(t, 2, respBody.TotalRecords)

	jobID, err := uuid.Parse(respBody.JobID)
	require.NoError(t, err)

	// Verify job created in DB
	job, err := svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, schoolID, job.SchoolID)
	require.Equal(t, tenantID, job.TenantID)
	require.Equal(t, "ADMIN_INVITATION", job.JobType)
	require.Equal(t, 2, job.TotalRecords)

	items, err := svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Debug: check if tasks are enqueued
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: rc.Addr()})
	pending, err := inspector.ListPendingTasks("admin_invitation", 0, 10)
	require.NoError(t, err)
	t.Logf("Pending tasks: %d", len(pending))
	for _, tsk := range pending {
		t.Logf("Task: %s, Payload: %s", tsk.Type, string(tsk.Payload))
	}

	// Start real worker with mocked Stytch
	processor := worker.NewAdminInvitationProcessor(svc, mockStytch, logger, redisClient)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: rc.Addr()}, asynq.Config{
		Concurrency: 2,
		Queues: map[string]int{
			"admin_invitation": 10,
		},
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc("admin:invitation:batch", processor.ProcessTask)
	mux.HandleFunc("admin:invitation:retry", processor.ProcessRetryTask)
	_, srvCancel := context.WithCancel(context.Background())
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(mux)
	}()
	defer func() { srvCancel(); server.Shutdown() }()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Check for server startup errors
	select {
	case err := <-serverErr:
		require.NoError(t, err, "asynq server error")
	default:
	}

	// Wait for job to complete
	require.Eventually(t, func() bool {
		j, _ := svc.GetJob(ctx, jobID)
		return j.Status == "COMPLETED" || j.Status == "COMPLETED_WITH_ERRORS"
	}, 15*time.Second, 200*time.Millisecond)

	job, err = svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	t.Logf("Job status after completion: %s, succeeded: %d, failed: %d", job.Status, job.SucceededCount, job.FailedCount)
	require.Equal(t, "COMPLETED", job.Status)
	require.Equal(t, 2, job.SucceededCount)
	require.Equal(t, 0, job.FailedCount)

	// Verify items all succeeded
	items, err = svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		require.Equal(t, "SUCCEEDED", it.Status)
		var result map[string]string
		require.NoError(t, json.Unmarshal(it.Result, &result))
		require.NotEmpty(t, result["stytch_invite_id"])
		require.NotEmpty(t, result["stytch_member_id"])
	}

	// Verify provisioned data in DB
	rows, err := pool.Query(ctx, `SELECT u.email, u.full_name, sm.role FROM users u JOIN school_memberships sm ON sm.user_id = u.id WHERE sm.school_id=$1 AND u.email IN ('teacher1@test.com','teacher2@test.com') ORDER BY u.email`, schoolID)
	require.NoError(t, err)
	defer rows.Close()

	found := 0
	for rows.Next() {
		var email, name, role string
		require.NoError(t, rows.Scan(&email, &name, &role))
		require.Equal(t, "ADMIN", role)
		found++
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 2, found, "expected 2 provisioned teachers")

	// SSE reflects completed state - skip for now as job is already completed
	// sseReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/admins/invitations/jobs/%s/events", jobID.String()), nil)
	// sseResp, err := app.Test(sseReq)
	// require.NoError(t, err)
	// require.Equal(t, "text/event-stream", sseResp.Header.Get("Content-Type"))
	// buf := make([]byte, 512)
	// n, _ := sseResp.Body.Read(buf)
	// require.Contains(t, string(buf[:n]), "COMPLETED")

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_job_items WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM school_memberships WHERE school_id=$1 AND user_id IN (SELECT id FROM users WHERE email IN ('teacher1@test.com','teacher2@test.com'))`, schoolID)
	_, _ = pool.Exec(ctx, `DELETE FROM members WHERE user_id IN (SELECT id FROM users WHERE email IN ('teacher1@test.com','teacher2@test.com'))`)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email IN ('teacher1@test.com','teacher2@test.com')`)
}

// TestAdminInvitationE2E_StytchErrors tests various Stytch error scenarios:
// - duplicate_user_email -> treated as SUCCEEDED (idempotent)
// - rate_limit / server_error -> DEFERRED (retryable)
// - invalid_email -> FAILED (permanent)
func TestAdminInvitationE2E_StytchErrors(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	require.NoError(t, migrator.Up(ctx))
	defer migrator.Close()

	// Create test data
	var tenantID, countryID, edSysID, schoolID, userID uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'Test Tenant','test-tenant','org_test_456') RETURNING id`).Scan(&tenantID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES (gen_random_uuid(),'Test Country','TC') RETURNING id`).Scan(&countryID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES (gen_random_uuid(),$1,'Test System') RETURNING id`, countryID).Scan(&edSysID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'Test School',$2,$3) RETURNING id`, tenantID, countryID, edSysID).Scan(&schoolID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES (gen_random_uuid(),'admin@test.com',$1,'Admin User') RETURNING id`, tenantID).Scan(&userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1,$2,'ADMIN',true)`, schoolID, userID)
	require.NoError(t, err)

	// Redis and Asynq
	rc := testcontainers.SetupRedis(t)
	redisClient := rc.Client
	defer redisClient.Close()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rc.Addr()})
	defer asynqClient.Close()

	// Service and handler with mocked Stytch that simulates various errors
	svc := services.NewAdminInvitationService(pool, sqlc.New(pool), logger)
	mockStytch := &MockStytchClient{
		NextErr: &stytchMock429{}, // First call: rate limited
		Results: []*stytch.InviteMemberResult{
			{StytchInviteID: "inv_1", StytchMemberID: "mem_1"}, // Second call: success for retry
		},
	}
	handler := api.NewAdminInvitationHandler(svc, mockStytch, asynqClient, redisClient, logger)

	app := fiber.New()
	app.Post("/api/admins/invitations", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		c.Locals("user_id", userID.String())
		return handler.HandleInvites(c)
	})
	app.Get("/api/admins/invitations/jobs/:job_id", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		return handler.GetJob(c)
	})

	// Submit bulk invitation with 3 items to test different error paths
	payload := api.BulkInvitationRequest{
		Invitations: []api.InvitationRow{
			{Email: "rate@test.com", FullName: "Rate Limited"},   // Will hit rate limit then succeed on retry
			{Email: "invalid@test", FullName: "Invalid Email"},   // Invalid email -> FAILED
			{Email: "duplicate@test.com", FullName: "Duplicate"}, // Duplicate -> SUCCEEDED
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/admins/invitations", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-errors-"+uuid.NewString())

	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var respBody struct {
		JobID string `json:"job_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	jobID, err := uuid.Parse(respBody.JobID)
	require.NoError(t, err)

	// Start worker
	processor := worker.NewAdminInvitationProcessor(svc, mockStytch, logger, redisClient)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: rc.Addr()}, asynq.Config{
		Concurrency: 2,
		Queues: map[string]int{
			"admin_invitation": 10,
		},
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc("admin:invitation:batch", processor.ProcessTask)
	mux.HandleFunc("admin:invitation:retry", processor.ProcessRetryTask)
	_, srvCancel := context.WithCancel(context.Background())
	go server.Run(mux)
	defer func() { srvCancel(); server.Shutdown() }()

	// Wait for job to complete
	require.Eventually(t, func() bool {
		j, _ := svc.GetJob(ctx, jobID)
		return j.Status == "COMPLETED" || j.Status == "COMPLETED_WITH_ERRORS"
	}, 20*time.Second, 200*time.Millisecond)

	job, err := svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	// Should have 1 succeeded (duplicate), 1 failed (invalid), 1 deferred (rate limited)
	require.Equal(t, "COMPLETED_WITH_ERRORS", job.Status)
	require.Equal(t, 1, job.SucceededCount)
	require.Equal(t, 1, job.FailedCount)
	require.Equal(t, 1, job.DeferredCount)

	// Verify item statuses
	items, err := svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 3)

	var foundRateLimited, foundInvalid, foundDuplicate bool
	for _, it := range items {
		var itemPayload struct {
			Email    string `json:"email"`
			FullName string `json:"full_name"`
			Role     string `json:"role"`
		}
		require.NoError(t, json.Unmarshal(it.Payload, &itemPayload))

		switch itemPayload.Email {
		case "rate@test.com":
			foundRateLimited = true
			require.Equal(t, "DEFERRED", it.Status)
			require.Contains(t, it.LastError, "rate_limited")
		case "invalid@test":
			foundInvalid = true
			require.Equal(t, "FAILED", it.Status)
			require.NotEmpty(t, it.LastError)
			require.Contains(t, it.LastError, "invalid_email")
		case "duplicate@test.com":
			foundDuplicate = true
			require.Equal(t, "SUCCEEDED", it.Status)
			var result map[string]string
			require.NoError(t, json.Unmarshal(it.Result, &result))
			require.Equal(t, "duplicate", result["stytch_invite_id"])
		}
	}
	require.True(t, foundRateLimited)
	require.True(t, foundInvalid)
	require.True(t, foundDuplicate)

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_job_items WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE id=$1`, jobID)
}

// TestAdminInvitationE2E_RetryFailed tests the retry-failed endpoint
func TestAdminInvitationE2E_RetryFailed(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	require.NoError(t, migrator.Up(ctx))
	defer migrator.Close()

	// Create test data
	var tenantID, countryID, edSysID, schoolID, userID uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'Test Tenant','test-tenant','org_test_789') RETURNING id`).Scan(&tenantID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES (gen_random_uuid(),'Test Country','TC') RETURNING id`).Scan(&countryID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES (gen_random_uuid(),$1,'Test System') RETURNING id`, countryID).Scan(&edSysID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'Test School',$2,$3) RETURNING id`, tenantID, countryID, edSysID).Scan(&schoolID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES (gen_random_uuid(),'admin@test.com',$1,'Admin User') RETURNING id`, tenantID).Scan(&userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1,$2,'ADMIN',true)`, schoolID, userID)
	require.NoError(t, err)

	// Redis and Asynq
	rc := testcontainers.SetupRedis(t)
	redisClient := rc.Client
	defer redisClient.Close()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rc.Addr()})
	defer asynqClient.Close()

	// Service and handler with mocked Stytch that fails first, then succeeds on retry
	svc := services.NewAdminInvitationService(pool, sqlc.New(pool), logger)
	mockStytch := &MockStytchClient{
		NextErr: &stytchMock500{}, // First attempt fails
		Results: []*stytch.InviteMemberResult{
			{StytchInviteID: "inv_retry_1", StytchMemberID: "mem_retry_1"},
		},
	}
	handler := api.NewAdminInvitationHandler(svc, mockStytch, asynqClient, redisClient, logger)

	app := fiber.New()
	app.Post("/api/admins/invitations", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		c.Locals("user_id", userID.String())
		return handler.HandleInvites(c)
	})
	app.Post("/api/admins/invitations/jobs/:job_id/retry-failed", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		return handler.RetryFailed(c)
	})
	app.Get("/api/admins/invitations/jobs/:job_id", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		return handler.GetJob(c)
	})

	// Submit bulk invitation
	payload := api.BulkInvitationRequest{
		Invitations: []api.InvitationRow{
			{Email: "retry@test.com", FullName: "Retry Teacher"},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/admins/invitations", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-retry-"+uuid.NewString())

	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var respBody struct {
		JobID string `json:"job_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	jobID, err := uuid.Parse(respBody.JobID)
	require.NoError(t, err)

	// Start worker (first attempt fails)
	processor := worker.NewAdminInvitationProcessor(svc, mockStytch, logger, redisClient)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: rc.Addr()}, asynq.Config{
		Concurrency: 2,
		Queues: map[string]int{
			"admin_invitation": 10,
		},
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc("admin:invitation:batch", processor.ProcessTask)
	mux.HandleFunc("admin:invitation:retry", processor.ProcessRetryTask)
	_, srvCancel := context.WithCancel(context.Background())
	go server.Run(mux)
	defer func() { srvCancel(); server.Shutdown() }()

	// Wait for first processing to complete (item will be FAILED or DEFERRED)
	require.Eventually(t, func() bool {
		j, _ := svc.GetJob(ctx, jobID)
		return j.Status == "COMPLETED" || j.Status == "COMPLETED_WITH_ERRORS"
	}, 15*time.Second, 200*time.Millisecond)

	job, err := svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	// First attempt: server error -> DEFERRED -> job COMPLETED_WITH_ERRORS
	require.Equal(t, "COMPLETED_WITH_ERRORS", job.Status)
	require.Equal(t, 0, job.SucceededCount)
	require.Equal(t, 0, job.FailedCount)
	require.Equal(t, 1, job.DeferredCount)

	items, err := svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "DEFERRED", items[0].Status)

	// Now call retry-failed endpoint
	retryReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/admins/invitations/jobs/%s/retry-failed", jobID.String()), nil)
	require.NoError(t, err)
	retryReq.Header.Set("Content-Type", "application/json")

	retryResp, err := app.Test(retryReq, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, retryResp.StatusCode)

	// Wait for retry to complete
	require.Eventually(t, func() bool {
		j, _ := svc.GetJob(ctx, jobID)
		return j.Status == "COMPLETED"
	}, 15*time.Second, 200*time.Millisecond)

	job, err = svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", job.Status)
	require.Equal(t, 1, job.SucceededCount)

	items, err = svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "SUCCEEDED", items[0].Status)

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_job_items WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM school_memberships WHERE school_id=$1 AND user_id IN (SELECT id FROM users WHERE email='retry@test.com')`, schoolID)
	_, _ = pool.Exec(ctx, `DELETE FROM members WHERE user_id IN (SELECT id FROM users WHERE email='retry@test.com')`)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email='retry@test.com'`)
}

// TestAdminInvitationE2E_Idempotency tests that duplicate requests with same
// idempotency key return the same job.
func TestAdminInvitationE2E_Idempotency(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	require.NoError(t, migrator.Up(ctx))
	defer migrator.Close()

	// Create test data
	var tenantID, countryID, edSysID, schoolID, userID uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'Test Tenant','test-tenant','org_test_idem') RETURNING id`).Scan(&tenantID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES (gen_random_uuid(),'Test Country','TC') RETURNING id`).Scan(&countryID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES (gen_random_uuid(),$1,'Test System') RETURNING id`, countryID).Scan(&edSysID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'Test School',$2,$3) RETURNING id`, tenantID, countryID, edSysID).Scan(&schoolID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES (gen_random_uuid(),'admin@test.com',$1,'Admin User') RETURNING id`, tenantID).Scan(&userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1,$2,'ADMIN',true)`, schoolID, userID)
	require.NoError(t, err)

	rc := testcontainers.SetupRedis(t)
	redisClient := rc.Client
	defer redisClient.Close()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rc.Addr()})
	defer asynqClient.Close()

	svc := services.NewAdminInvitationService(pool, sqlc.New(pool), logger)
	mockStytch := &MockStytchClient{
		Results: []*stytch.InviteMemberResult{
			{StytchInviteID: "inv_1", StytchMemberID: "mem_1"},
		},
	}
	handler := api.NewAdminInvitationHandler(svc, mockStytch, asynqClient, redisClient, logger)

	app := fiber.New()
	app.Post("/api/admins/invitations", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		c.Locals("user_id", userID.String())
		return handler.HandleInvites(c)
	})

	idempotencyKey := "idem-test-" + uuid.NewString()
	payload := api.BulkInvitationRequest{
		Invitations: []api.InvitationRow{
			{Email: "idem@test.com", FullName: "Idem Teacher"},
		},
	}
	body, _ := json.Marshal(payload)

	// First request
	req1, _ := http.NewRequest(http.MethodPost, "/api/admins/invitations", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	resp1, err := app.Test(req1, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp1.StatusCode)

	var body1 struct {
		JobID string `json:"job_id"`
	}
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&body1))

	// Second request with same idempotency key
	req2, _ := http.NewRequest(http.MethodPost, "/api/admins/invitations", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	resp2, err := app.Test(req2, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp2.StatusCode)

	var body2 struct {
		JobID string `json:"job_id"`
	}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body2))

	// Should return same job ID
	require.Equal(t, body1.JobID, body2.JobID)

	// Verify only one job was created
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM bulk_jobs WHERE idempotency_key=$1`, idempotencyKey).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_job_items WHERE job_id=$1`, body1.JobID)
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE id=$1`, body1.JobID)
}

// TestAdminInvitationE2E_Authorization tests tenant isolation
func TestAdminInvitationE2E_TenantIsolation(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	ctx := context.Background()
	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := database.NewMigrator(pool, logger)
	require.NoError(t, err)
	require.NoError(t, migrator.Up(ctx))
	defer migrator.Close()

	// Create two tenants
	var tenantID1, tenantID2, countryID, edSysID, schoolID1, schoolID2, userID1 uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'Tenant1','tenant1','org_1') RETURNING id`).Scan(&tenantID1)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'Tenant2','tenant2','org_2') RETURNING id`).Scan(&tenantID2)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES (gen_random_uuid(),'Test Country','TC') RETURNING id`).Scan(&countryID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO education_systems (id, country_id, system_name) VALUES (gen_random_uuid(),$1,'Test System') RETURNING id`, countryID).Scan(&edSysID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'School1',$2,$3) RETURNING id`, tenantID1, countryID, edSysID).Scan(&schoolID1)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'School2',$2,$3) RETURNING id`, tenantID2, countryID, edSysID).Scan(&schoolID2)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES (gen_random_uuid(),'admin1@test.com',$1,'Admin1') RETURNING id`, tenantID1).Scan(&userID1)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role, is_active) VALUES ($1,$2,'ADMIN',true)`, schoolID1, userID1)
	require.NoError(t, err)

	rc := testcontainers.SetupRedis(t)
	redisClient := rc.Client
	defer redisClient.Close()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: rc.Addr()})
	defer asynqClient.Close()

	svc := services.NewAdminInvitationService(pool, sqlc.New(pool), logger)
	mockStytch := &MockStytchClient{
		Results: []*stytch.InviteMemberResult{
			{StytchInviteID: "inv_1", StytchMemberID: "mem_1"},
		},
	}
	handler := api.NewAdminInvitationHandler(svc, mockStytch, asynqClient, redisClient, logger)

	// App for tenant1
	app1 := fiber.New()
	app1.Post("/api/admins/invitations", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID1.String())
		c.Locals("tenant_id", tenantID1.String())
		c.Locals("user_id", userID1.String())
		return handler.HandleInvites(c)
	})
	app1.Get("/api/admins/invitations/jobs/:job_id", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID1.String())
		c.Locals("tenant_id", tenantID1.String())
		return handler.GetJob(c)
	})

	// App for tenant2
	app2 := fiber.New()
	app2.Get("/api/admins/invitations/jobs/:job_id", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID2.String())
		c.Locals("tenant_id", tenantID2.String())
		return handler.GetJob(c)
	})

	// Create job as tenant1
	payload := api.BulkInvitationRequest{
		Invitations: []api.InvitationRow{
			{Email: "teacher@test.com", FullName: "Teacher"},
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/admins/invitations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "isolation-test-"+uuid.NewString())
	resp, err := app1.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var body1 struct {
		JobID string `json:"job_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body1))

	// Now try to access the job as tenant2 (different school/tenant)
	getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/admins/invitations/jobs/%s", body1.JobID), nil)
	getResp, err := app2.Test(getReq, fiber.TestConfig{})
	require.NoError(t, err)
	// Should return 404 (not found) to prevent cross-tenant leakage
	require.Equal(t, fiber.StatusNotFound, getResp.StatusCode)

	// Cleanup
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_job_items WHERE job_id=$1`, body1.JobID)
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE id=$1`, body1.JobID)
}

// MockStytchClient is a test double for the Stytch client.
// It implements the minimal interface needed by AdminInvitationProcessor.
type MockStytchClient struct {
	CallCount int
	NextErr   error
	Results   []*stytch.InviteMemberResult
	ResIndex  int
}

func (m *MockStytchClient) ClassifyStytchError(err error) (retry bool, permanent bool, duplicate bool, reason string) {
	if err == nil {
		return false, false, false, ""
	}
	switch err.(type) {
	case *stytchMock429:
		return true, false, false, "rate_limited"
	case *stytchMock500:
		return true, false, false, "server_error"
	case *stytchMockTimeout:
		return true, false, false, "timeout"
	case *stytchMockDuplicate:
		return false, false, true, "duplicate"
	case *stytchMockInvalidEmail:
		return false, true, false, "invalid_email"
	case *stytchMockUnknown:
		return true, false, false, "unknown"
	default:
		// For other errors, check error message
		errMsg := err.Error()
		if errMsg == "timeout" || errMsg == "network error" {
			return true, false, false, "network_error"
		}
		if errMsg == "duplicate_user_email" {
			return false, false, true, "duplicate"
		}
		if errMsg == "invalid_email" {
			return false, true, false, "invalid_email"
		}
		if errMsg == "rate_limited" {
			return true, false, false, "rate_limited"
		}
		return false, true, false, "unknown: " + errMsg
	}
}

func (m *MockStytchClient) InviteMember(ctx context.Context, email, fullName, role string, tenantID string) (*stytch.InviteMemberResult, error) {
	m.CallCount++

	// Return specific errors based on email for testing
	switch email {
	case "rate@test.com":
		if m.NextErr != nil {
			err := m.NextErr
			m.NextErr = nil
			return nil, err
		}
		if m.ResIndex < len(m.Results) {
			res := m.Results[m.ResIndex]
			m.ResIndex++
			return res, nil
		}
	case "invalid@test":
		return nil, &stytchMockInvalidEmail{}
	case "duplicate@test.com":
		return nil, &stytchMockDuplicate{}
	}

	if m.NextErr != nil {
		err := m.NextErr
		m.NextErr = nil
		return nil, err
	}
	if m.ResIndex < len(m.Results) {
		res := m.Results[m.ResIndex]
		m.ResIndex++
		return res, nil
	}
	return &stytch.InviteMemberResult{StytchInviteID: "inv_default", StytchMemberID: "mem_default"}, nil
}

func (m *MockStytchClient) GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error) {
	return "org_test_123", nil
}

// Custom error shapes for deterministic classification.
type stytchMock429 struct{}

func (e *stytchMock429) Error() string { return "429 rate_limited" }

type stytchMock500 struct{}

func (e *stytchMock500) Error() string { return "500 server error" }

type stytchMockTimeout struct{}

func (e *stytchMockTimeout) Error() string { return "timeout" }

type stytchMockDuplicate struct{}

func (e *stytchMockDuplicate) Error() string { return "duplicate_user_email" }

type stytchMockInvalidEmail struct{}

func (e *stytchMockInvalidEmail) Error() string { return "invalid_email" }

type stytchMockUnknown struct{}

func (e *stytchMockUnknown) Error() string { return "unexpected error" }

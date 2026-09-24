//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/api"
	"somotracker/backend/internal/database"
	"somotracker/backend/internal/services"
	"somotracker/backend/internal/testcontainers"
	"somotracker/backend/internal/worker"
)

func TestStudentsImportFullE2E_EnqueueWorkerSSE(t *testing.T) {
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

	// Seed minimal master data for FK constraints
	var tenantID, countryID, edSysID, schoolID, userID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO tenants (id, name, slug, stytch_org_id)
		VALUES (gen_random_uuid(), 'Test Tenant', 'test-tenant', 'org-test')
		RETURNING id
	`).Scan(&tenantID)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO countries (id, country_name, country_code)
		VALUES (gen_random_uuid(), 'Testland', 'TL')
		RETURNING id
	`).Scan(&countryID)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO education_systems (id, country_id, system_name)
		VALUES (gen_random_uuid(), $1, 'CBE')
		RETURNING id
	`, countryID).Scan(&edSysID)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id)
		VALUES (gen_random_uuid(), $1, 'Test School', $2, $3)
		RETURNING id
	`, tenantID, countryID, edSysID).Scan(&schoolID)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO users (id, email, tenant_id, full_name)
		VALUES (gen_random_uuid(), 'admin@test.com', $1, 'Admin User')
		RETURNING id
	`, tenantID).Scan(&userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role) VALUES ($1,$2,'ADMIN')`, schoolID, userID)
	require.NoError(t, err)

	// Redis & Asynq
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	require.Eventually(t, func() bool { return redisClient.Ping(ctx).Err() == nil }, 5e9, 100e6)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	defer asynqClient.Close()

	svc := services.NewStudentImportService(pool, logger)
	handler := api.NewStudentsImportHandler(svc, asynqClient, redisClient, logger)

	app := fiber.New()
	app.Post("/students/add", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID.String())
		c.Locals("tenant_id", tenantID.String())
		c.Locals("user_id", userID.String())
		return handler.HandleImport(c)
	})
	app.Get("/students/jobs/:job_id/events", func(c fiber.Ctx) error {
		c.Locals("tenant_id", tenantID.String())
		return handler.Events(c)
	})

	// Submit import
	payload := api.BulkStudentRequest{
		Students: []api.StudentImportRow{
			{AdmissionNumber: "A001", FullName: "Alice Test", DateOfBirth: "2006-01-01", Gender: "F"},
			{AdmissionNumber: "A002", FullName: "Bob Test", DateOfBirth: "2006-02-02", Gender: "M"},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/students/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var respBody struct {
		JobID string `json:"job_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	jobID, err := uuid.Parse(respBody.JobID)
	require.NoError(t, err)

	// Verify job created
	job, err := svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, int(2), job.TotalRecords)

	// Start real Asynq server to process the job
	processor := worker.NewStudentImportProcessor(svc, logger, redisClient)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: "localhost:6379"}, asynq.Config{
		Concurrency: 2,
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc("student:import:batch", processor.ProcessTask)
	_, serverCancel := context.WithCancel(context.Background())
	go func() {
		if err := server.Run(mux); err != nil {
			t.Logf("asynq server error: %v", err)
		}
	}()
	defer func() {
		serverCancel()
		server.Shutdown()
	}()

	// Wait for job to complete
	require.Eventually(t, func() bool {
		j, err := svc.GetJob(ctx, jobID)
		if err != nil {
			return false
		}
		return j.Status == "COMPLETED"
	}, 10e9, 200e6)

	job, err = svc.GetJob(ctx, jobID)
	require.NoError(t, err)

	job, err = svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", job.Status)

	// Verify students persisted
	var cnt int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM students WHERE school_id=$1 AND full_name IN ('Alice Test','Bob Test')`, schoolID).Scan(&cnt)
	require.NoError(t, err)
	require.Equal(t, 2, cnt)

	// SSE verification
	sseReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/students/jobs/%s/events", jobID.String()), nil)
	sseResp, err := app.Test(sseReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, sseResp.StatusCode)
	require.Equal(t, "text/event-stream", sseResp.Header.Get("Content-Type"))
	// Read first event frame
	buf := make([]byte, 512)
	n, _ := sseResp.Body.Read(buf)
	frame := string(buf[:n])
	require.Contains(t, frame, "event: progress")
	require.Contains(t, frame, `"status":"COMPLETED"`)
}

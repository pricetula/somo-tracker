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

func TestStudentsImportFullE2E_DuplicateAdmissionNumber(t *testing.T) {
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

	var tenantID, countryID, edSysID, schoolID, userID uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES (gen_random_uuid(),'T','t','o') RETURNING id`).Scan(&tenantID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO countries (id, country_name, country_code) VALUES (gen_random_uuid(),'C','CC') RETURNING id`).Scan(&countryID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO education_systems (id, name) VALUES (gen_random_uuid(),'S') RETURNING id`).Scan(&edSysID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO schools (id, tenant_id, school_name, country_id, education_system_id) VALUES (gen_random_uuid(),$1,'S',$2,$3) RETURNING id`, tenantID, countryID, edSysID).Scan(&schoolID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES (gen_random_uuid(),'a@b.com',$1,'A') RETURNING id`, tenantID).Scan(&userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO school_memberships (school_id, user_id, role) VALUES ($1,$2,'ADMIN')`, schoolID, userID)
	require.NoError(t, err)

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

	// Pre-insert a student with admission number DUP-001
	_, err = pool.Exec(ctx, `INSERT INTO students (school_id, admission_number, full_name, date_of_birth, gender) VALUES ($1,'DUP-001','Existing','2000-01-01','M')`, schoolID)
	require.NoError(t, err)

	payload := api.BulkStudentRequest{
		Students: []api.StudentImportRow{
			{AdmissionNumber: "DUP-001", FullName: "Should Fail", DateOfBirth: "2006-01-01", Gender: "F"},
			{AdmissionNumber: "NEW-001", FullName: "Should Succeed", DateOfBirth: "2006-02-02", Gender: "F"},
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
	jobID, _ := uuid.Parse(respBody.JobID)

	// Run real worker
	processor := worker.NewStudentImportProcessor(svc, logger, redisClient)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: "localhost:6379"}, asynq.Config{Concurrency: 2})
	mux := asynq.NewServeMux()
	mux.HandleFunc("student:import:batch", processor.ProcessTask)
	server.Mux = mux
	srvCtx, srvCancel := context.WithCancel(context.Background())
	go server.Run(srvCtx)
	defer func() { srvCancel(); server.Shutdown() }()

	require.Eventually(t, func() bool {
		j, _ := svc.GetJob(ctx, jobID)
		return j.Status == "COMPLETED_WITH_ERRORS" || j.Status == "COMPLETED"
	}, 15e9, 200e6)

	job, err := svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, "COMPLETED_WITH_ERRORS", job.Status)
	require.Equal(t, int(1), job.SucceededCount)
	require.Equal(t, int(1), job.FailedCount)

	items, err := svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Verify duplicate failed
	var failedFound bool
	for _, it := range items {
		if string(it.Payload) != "" {
			var row services.StudentImportItem
			_ = json.Unmarshal(it.Payload, &row)
			if row.AdmissionNumber == "DUP-001" {
				require.Equal(t, "FAILED", it.Status)
				failedFound = true
			}
		}
	}
	require.True(t, failedFound)

	// Verify new student exists
	var cnt int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM students WHERE school_id=$1 AND admission_number='NEW-001'`, schoolID).Scan(&cnt)
	require.NoError(t, err)
	require.Equal(t, 1, cnt)

	// SSE reflects error state
	sseReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/students/jobs/%s/events", jobID.String()), nil)
	sseResp, err := app.Test(sseReq)
	require.NoError(t, err)
	require.Equal(t, "text/event-stream", sseResp.Header.Get("Content-Type"))
	buf := make([]byte, 512)
	n, _ := sseResp.Body.Read(buf)
	require.Contains(t, string(buf[:n]), "COMPLETED_WITH_ERRORS")
}

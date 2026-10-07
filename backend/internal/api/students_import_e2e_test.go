//go:build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

const testDSN = "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"

// TestStudentsImportE2E_PostStudentsAdd verifies the full flow:
// 1. POST /students/add creates a bulk job with items
// 2. Items are processed and students are inserted into DB
// 3. Job finalizes to COMPLETED
// Stytch is bypassed by setting Fiber locals directly (no session middleware).
func TestStudentsImportE2E_PostStudentsAdd_CreatesJobAndStudents(t *testing.T) {
	ctx := context.Background()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = testDSN
	}

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pool.Ping(ctx))

	logger := zap.NewNop()
	svc := services.NewStudentImportService(pool, logger)
	handler := NewStudentsImportHandler(svc, nil, nil, logger)

	// Resolve existing school/tenant/user to satisfy FK constraints.
	var schoolID, tenantID, userID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT s.id, s.tenant_id, u.id FROM schools s JOIN users u ON u.tenant_id = s.tenant_id LIMIT 1`).Scan(&schoolID, &tenantID, &userID)
	if err != nil {
		t.Fatalf("failed to find existing school/tenant/user for test: %v", err)
	}

	app := fiber.New()
	app.Post("/students/add", func(c fiber.Ctx) error {
		c.Locals("active_school_id", schoolID)
		c.Locals("tenant_id", tenantID)
		c.Locals("user_id", userID)
		return handler.HandleImport(c)
	})

	payload := BulkStudentRequest{
		Students: []StudentImportRow{
			{AdmissionNumber: "ADM-" + uuid.NewString()[:6], FullName: "Alice Example", DateOfBirth: "2006-04-10", Gender: "F"},
			{AdmissionNumber: "ADM-" + uuid.NewString()[:6], FullName: "Bob Example", DateOfBirth: "2005-11-22", Gender: "M"},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/students/add", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	var respBody struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	require.NotEmpty(t, respBody.JobID)
	require.Equal(t, "QUEUED", respBody.Status)

	jobID, err := uuid.Parse(respBody.JobID)
	require.NoError(t, err)

	// Verify job created in DB
	job, err := svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, schoolID, job.SchoolID)
	require.Equal(t, tenantID, job.TenantID)
	require.Equal(t, "STUDENT_IMPORT", job.JobType)
	require.Equal(t, 2, job.TotalRecords)

	items, err := svc.GetItemsByJobID(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	// Simulate worker processing each item
	for _, item := range items {
		var row services.StudentImportItem
		require.NoError(t, json.Unmarshal(item.Payload, &row))

		gender := services.NormalizeGender(row.Gender)
		dob, err := services.ParseDate(row.DateOfBirth)
		require.NoError(t, err)

		metadataBytes, _ := json.Marshal(row.Metadata)
		require.NoError(t, svc.InsertStudent(ctx, schoolID, row.AdmissionNumber, row.FullName, dob, gender, metadataBytes))

		require.NoError(t, svc.UpdateItemStatus(ctx, item.ID, "SUCCEEDED", []byte(`{"admission_number":"`+row.AdmissionNumber+`"}`), "", 1))
		require.NoError(t, svc.IncrementJobCounters(ctx, jobID, 1, 0, 0))
	}

	// Finalize job
	finalized, err := svc.TryFinalizeJob(ctx, jobID)
	require.NoError(t, err)
	require.True(t, finalized)

	job, err = svc.GetJob(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", job.Status)
	require.Equal(t, 2, job.SucceededCount)

	// Verify students actually created
	rows, err := pool.Query(ctx, `SELECT admission_number, full_name, gender FROM students WHERE school_id=$1 AND full_name IN ('Alice Example','Bob Example') ORDER BY full_name`, schoolID)
	require.NoError(t, err)
	defer rows.Close()

	foundAlice := false
	foundBob := false
	for rows.Next() {
		var adm, name, gender string
		require.NoError(t, rows.Scan(&adm, &name, &gender))
		if name == "Alice Example" {
			foundAlice = true
			require.Equal(t, "F", gender)
		}
		if name == "Bob Example" {
			foundBob = true
			require.Equal(t, "M", gender)
		}
	}
	require.NoError(t, rows.Err())
	require.True(t, foundAlice, "Alice student not found")
	require.True(t, foundBob, "Bob student not found")

	// Cleanup test data
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_job_items WHERE job_id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM bulk_jobs WHERE id=$1`, jobID)
	_, _ = pool.Exec(ctx, `DELETE FROM students WHERE school_id=$1 AND full_name IN ('Alice Example','Bob Example')`, schoolID)
}

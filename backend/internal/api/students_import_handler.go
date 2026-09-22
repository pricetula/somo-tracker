package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
)

type StudentsImportHandler struct {
	svc    services.StudentImportService
	asynq  *asynq.Client
	redis  *redis.Client
	logger *zap.Logger
}

func NewStudentsImportHandler(svc services.StudentImportService, asynqClient *asynq.Client, redisClient *redis.Client, logger *zap.Logger) *StudentsImportHandler {
	return &StudentsImportHandler{svc: svc, asynq: asynqClient, redis: redisClient, logger: logger.With(zap.String("handler", "students_import"))}
}

type StudentImportRow struct {
	AdmissionNumber string                 `json:"admission_number"`
	FullName        string                 `json:"full_name"`
	DateOfBirth     string                 `json:"date_of_birth"`
	Gender          string                 `json:"gender"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type BulkStudentRequest struct {
	Students []StudentImportRow `json:"students"`
}

func (h *StudentsImportHandler) HandleImport(c fiber.Ctx) error {
	schoolIDStr := c.Locals("active_school_id")
	tenantIDStr := c.Locals("tenant_id")
	userIDStr := c.Locals("user_id")

	schoolID, err1 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	userID, err3 := uuid.Parse(fmt.Sprintf("%v", userIDStr))
	if err1 != nil || err2 != nil || err3 != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "missing session context", "errors": fiber.Map{}})
	}

	hasAdmin, err := h.svc.UserHasAdminRole(c.Context(), schoolID, userID)
	if err != nil {
		h.logger.Error("admin role check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to verify permissions", "errors": fiber.Map{}})
	}
	if !hasAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "forbidden", "message": "insufficient permissions", "errors": fiber.Map{}})
	}

	var req BulkStudentRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid request body", "errors": fiber.Map{"body": []string{"malformed json"}}})
	}

	if len(req.Students) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "students array is required", "errors": fiber.Map{"students": []string{"required"}}})
	}
	if len(req.Students) > 10000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "exceeds maximum of 10000 rows", "errors": fiber.Map{"students": []string{"max 10000"}}})
	}

	idempotencyKey := c.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "Idempotency-Key header is required", "errors": fiber.Map{"Idempotency-Key": []string{"required"}}})
	}

	if existing, err := h.svc.GetJobByIdempotency(c.Context(), tenantID, idempotencyKey); err == nil && existing != nil {
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"job_id": existing.ID, "status": existing.Status, "total_records": existing.TotalRecords, "message": "existing job returned",
		})
	}

	validationErrors := make(map[string][]string)
	seenAdmission := make(map[string]struct{})
	var dedupedRows []StudentImportRow
	for idx, row := range req.Students {
		adm := strings.TrimSpace(row.AdmissionNumber)
		name := strings.TrimSpace(row.FullName)
		dobRaw := strings.TrimSpace(row.DateOfBirth)
		genderRaw := strings.TrimSpace(row.Gender)

		hasError := false
		if adm == "" {
			validationErrors[fmt.Sprintf("students[%d].admission_number", idx)] = append(validationErrors[fmt.Sprintf("students[%d].admission_number", idx)], "required")
			hasError = true
		}
		if name == "" {
			validationErrors[fmt.Sprintf("students[%d].full_name", idx)] = append(validationErrors[fmt.Sprintf("students[%d].full_name", idx)], "required")
			hasError = true
		}
		if dobRaw == "" {
			validationErrors[fmt.Sprintf("students[%d].date_of_birth", idx)] = append(validationErrors[fmt.Sprintf("students[%d].date_of_birth", idx)], "required")
			hasError = true
		} else {
			if _, err := services.ParseDate(dobRaw); err != nil {
				validationErrors[fmt.Sprintf("students[%d].date_of_birth", idx)] = append(validationErrors[fmt.Sprintf("students[%d].date_of_birth", idx)], "invalid date format")
				hasError = true
			}
		}
		if genderRaw == "" {
			genderRaw = "OTHER"
		}
		// Normalize gender now to ensure consistent storage
		_ = services.NormalizeGender(genderRaw)

		if hasError {
			continue
		}

		admNorm := strings.TrimSpace(adm)
		key := strings.ToLower(admNorm)
		if _, exists := seenAdmission[key]; exists {
			// skip duplicate within file, first wins
			continue
		}
		seenAdmission[key] = struct{}{}
		// Normalize fields for storage
		row.AdmissionNumber = admNorm
		row.FullName = name
		row.DateOfBirth = dobRaw
		row.Gender = services.NormalizeGender(genderRaw)
		dedupedRows = append(dedupedRows, row)
	}

	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "validation_failed", "message": "some student rows are invalid", "errors": validationErrors,
		})
	}
	if len(dedupedRows) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "bad_request", "message": "no valid student rows after deduplication", "errors": fiber.Map{"students": []string{"no valid rows"}},
		})
	}

	items := make([]services.StudentImportItem, len(dedupedRows))
	for i, r := range dedupedRows {
		items[i] = services.StudentImportItem{
			ID:              uuid.New(),
			AdmissionNumber: r.AdmissionNumber,
			FullName:        r.FullName,
			DateOfBirth:     r.DateOfBirth,
			Gender:          r.Gender,
			Metadata:        r.Metadata,
		}
	}

	jobID, err := h.svc.CreateBulkJobWithItems(c.Context(), schoolID, tenantID, userID, idempotencyKey, items)
	if err != nil {
		h.logger.Error("bulk job creation failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to create bulk job", "errors": fiber.Map{}})
	}

	if h.asynq != nil {
		batchSize := 100
		totalBatches := (len(items) + batchSize - 1) / batchSize
		for b := 0; b < totalBatches; b++ {
			start := b * batchSize
			end := start + batchSize
			if end > len(items) {
				end = len(items)
			}
			batchItemIDs := make([]string, 0, end-start)
			for j := start; j < end; j++ {
				batchItemIDs = append(batchItemIDs, items[j].ID.String())
			}
			payload, marshalErr := json.Marshal(map[string]interface{}{
				"job_id":      jobID.String(),
				"batch_index": b,
				"item_ids":    batchItemIDs,
			})
			if marshalErr != nil {
				h.logger.Error("asynq payload marshal failed", zap.Error(marshalErr))
				continue
			}
			_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("student:import:batch", payload), asynq.Queue("student_import"), asynq.TaskID(fmt.Sprintf("%s_%d", jobID.String(), b)))
			if enqueueErr != nil {
				h.logger.Error("asynq enqueue failed", zap.Error(enqueueErr))
				continue
			}
		}
	}

	if h.redis != nil {
		if err := h.redis.Publish(c.Context(), "bulk_progress_"+jobID.String(), `{"status":"QUEUED","total":`+fmt.Sprintf("%d", len(dedupedRows))+`}`).Err(); err != nil {
			h.logger.Warn("redis publish failed", zap.Error(err), zap.String("job_id", jobID.String()))
		}
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id":        jobID.String(),
		"status":        "QUEUED",
		"total_records": len(dedupedRows),
		"message":       "Bulk student import job registered successfully",
	})
}

func (h *StudentsImportHandler) GetJob(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid job_id", "errors": fiber.Map{}})
	}
	tenantIDStr := c.Locals("tenant_id")
	schoolIDStr := c.Locals("active_school_id")
	tenantID, err1 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	schoolID, err2 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
	if err1 != nil || err2 != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "missing session context", "errors": fiber.Map{}})
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID || job.SchoolID != schoolID || job.JobType != "STUDENT_IMPORT" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "job not found", "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusOK).JSON(job)
}

func (h *StudentsImportHandler) Events(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.SendString("event: error\ndata: bad job_id\n\n")
	}
	tenantIDStr := c.Locals("tenant_id")
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	if err2 != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.SendString("event: error\ndata: missing session locals\n\n")
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID {
		c.Status(fiber.StatusNotFound)
		return c.SendString("event: error\ndata: job not found\n\n")
	}

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		initData, _ := json.Marshal(map[string]interface{}{
			"status": job.Status, "succeeded": job.SucceededCount, "failed": job.FailedCount, "deferred": job.DeferredCount, "total": job.TotalRecords,
		})
		fmt.Fprintf(w, "event: progress\ndata: %s\n\n", initData)
		_ = w.Flush()

		channel := "bulk_progress_" + jobIDStr
		sub := h.redis.Subscribe(c.Context(), channel)
		defer func() { _ = sub.Close() }()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-sub.Channel():
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: progress\ndata: %s\n\n", msg.Payload)
				_ = w.Flush()
			case <-ticker.C:
				fmt.Fprintf(w, "event: heartbeat\ndata: \n\n")
				_ = w.Flush()
			case <-c.Context().Done():
				return
			}
		}
	})
	return nil
}

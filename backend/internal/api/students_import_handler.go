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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid school, tenant, or user id",
			"errors":  fiber.Map{},
		})
	}

	var req BulkStudentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid request body",
			"errors":  fiber.Map{},
		})
	}

	if len(req.Students) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "no students provided",
			"errors":  fiber.Map{},
		})
	}

	items := make([]services.StudentImportItem, 0, len(req.Students))
	for _, row := range req.Students {
		if row.AdmissionNumber == "" || row.FullName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "bad_request",
				"message": "admission_number and full_name are required",
				"errors":  fiber.Map{},
			})
		}
		items = append(items, services.StudentImportItem{
			ID:              uuid.New(),
			AdmissionNumber: strings.TrimSpace(row.AdmissionNumber),
			FullName:        strings.TrimSpace(row.FullName),
			DateOfBirth:     row.DateOfBirth,
			Gender:          row.Gender,
			Metadata:        row.Metadata,
		})
	}

	idempotencyKey := fmt.Sprintf("student-import-%s-%d", schoolID.String(), time.Now().UnixMilli())

	jobID, err := h.svc.CreateBulkJobWithItems(c.Context(), schoolID, tenantID, userID, idempotencyKey, items)
	if err != nil {
		h.logger.Error("create bulk job failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "failed to create import job",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id": jobID.String(),
		"status": "QUEUED",
	})
}

func (h *StudentsImportHandler) GetJob(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "bad_request",
			"message": "invalid job_id",
			"errors":  fiber.Map{},
		})
	}

	tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "tenant not found",
			"errors":  fiber.Map{},
		})
	}

	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":    "not_found",
			"message": "job not found",
			"errors":  fiber.Map{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(job)
}

func (h *StudentsImportHandler) Events(c fiber.Ctx) error {
	jobIDStr := c.Query("job_id")
	if jobIDStr == "" {
		return c.Status(fiber.StatusBadRequest).SendString("event: error\ndata: missing job_id\n\n")
	}
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("event: error\ndata: invalid job_id\n\n")
	}

	tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).SendString("event: error\ndata: unauthorized\n\n")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID {
		c.Status(fiber.StatusNotFound)
		return c.SendString("event: error\ndata: job not found\n\n")
	}

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		initData, err := json.Marshal(map[string]interface{}{
			"status": job.Status, "succeeded": job.SucceededCount, "failed": job.FailedCount, "deferred": job.DeferredCount, "total": job.TotalRecords,
		})
		if err != nil {
			return
		}
		if _, err := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", initData); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

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
				if _, err := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", msg.Payload); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := fmt.Fprintf(w, "event: heartbeat\ndata: \n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-c.Context().Done():
				return
			}
		}
	})
	return nil
}

package api

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
	"somotracker/backend/internal/stytch"
)

type AdminInvitationHandler struct {
	svc    services.AdminInvitationService
	stytch *stytch.Client
	asynq  *asynq.Client
	redis  *redis.Client
	logger *zap.Logger
}

func NewAdminInvitationHandler(svc services.AdminInvitationService, cli *stytch.Client, asynqClient *asynq.Client, redisClient *redis.Client, logger *zap.Logger) *AdminInvitationHandler {
	return &AdminInvitationHandler{svc: svc, stytch: cli, asynq: asynqClient, redis: redisClient, logger: logger.With(zap.String("handler", "admin_invitation"))}
}

type InvitationRow struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type BulkInvitationRequest struct {
	Invitations []InvitationRow `json:"invitations"`
}

func (h *AdminInvitationHandler) HandleInvites(c fiber.Ctx) error {
	// Extract locals
	schoolIDStr := c.Locals("school_id")
	tenantIDStr := c.Locals("tenant_id")
	adminUserIDStr := c.Locals("admin_user_id")

	schoolID, err1 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	adminUserID, err3 := uuid.Parse(fmt.Sprintf("%v", adminUserIDStr))
	if err1 != nil || err2 != nil || err3 != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code": "unauthorized", "message": "missing session context", "errors": fiber.Map{},
		})
	}

	var req BulkInvitationRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "bad_request", "message": "invalid request body", "errors": fiber.Map{"body": []string{"malformed json"}},
		})
	}

	if len(req.Invitations) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "bad_request", "message": "invitations array is required", "errors": fiber.Map{"invitations": []string{"required"}},
		})
	}
	if len(req.Invitations) > 10000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "bad_request", "message": "exceeds maximum of 10000 rows", "errors": fiber.Map{"invitations": []string{"max 10000"}},
		})
	}

	// Validation
	validRoles := map[string]bool{"ADMIN": true, "TEACHER": true, "GUARDIAN": true, "FINANCE": true}
	var errors []fiber.Map
	for idx, row := range req.Invitations {
		if strings.TrimSpace(row.Email) == "" {
			errors = append(errors, fiber.Map{"row_index": idx, "field": "email", "message": "required"})
		} else {
			if _, parseErr := mail.ParseAddress(row.Email); parseErr != nil {
				errors = append(errors, fiber.Map{"row_index": idx, "field": "email", "message": "invalid email format"})
			}
		}
		if strings.TrimSpace(row.FullName) == "" {
			errors = append(errors, fiber.Map{"row_index": idx, "field": "full_name", "message": "required"})
		}
		if !validRoles[row.Role] {
			errors = append(errors, fiber.Map{"row_index": idx, "field": "role", "message": "unknown role"})
		}
	}
	if len(errors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "validation_failed", "message": "some invitation rows are invalid", "errors": errors,
		})
	}

	// Idempotency key from header or generated
	idempotencyKey := c.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("inv_%s_%s", tenantID.String(), time.Now().Format(time.RFC3339))
	}

	// Check existing job by idempotency
	if existing, err := h.svc.GetJobByIdempotency(c.Context(), tenantID, idempotencyKey); err == nil && existing != nil {
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"job_id": existing.ID, "status": existing.Status, "total_records": existing.TotalRecords, "message": "existing job returned",
		})
	}

	// Create job
	jobID, err := h.svc.CreateBulkJob(c.Context(), schoolID, tenantID, adminUserID, idempotencyKey, len(req.Invitations))
	if err != nil {
		h.logger.Error("bulk job creation failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "failed to create bulk job", "errors": fiber.Map{},
		})
	}

	// Insert items
	items := make([]map[string]interface{}, len(req.Invitations))
	for i, r := range req.Invitations {
		items[i] = map[string]interface{}{"email": r.Email, "full_name": r.FullName, "role": r.Role}
	}
	if err := h.svc.InsertItems(c.Context(), jobID, items); err != nil {
		h.logger.Error("bulk item insertion failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "failed to insert job items", "errors": fiber.Map{},
		})
	}

	// Enqueue batches of 40
	batchSize := 40
	totalBatches := (len(items) + batchSize - 1) / batchSize
	for b := 0; b < totalBatches; b++ {
		start := b * batchSize
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"job_id": jobID.String(), "batch_index": b, "start_index": start, "end_index": end,
		})
		_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("admin:invitation:batch", payload), asynq.Queue("admin_invitation"), asynq.TaskID(fmt.Sprintf("%s_%d", jobID.String(), b)))
		if enqueueErr != nil {
			h.logger.Error("asynq enqueue failed", zap.Error(enqueueErr))
		}
	}

	// Publish initial progress
	if h.redis != nil {
		h.redis.Publish(c.Context(), "bulk_progress_"+jobID.String(), `{"status":"QUEUED","total":`+fmt.Sprintf("%d", len(req.Invitations))+`}`)
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id": jobID.String(), "status": "QUEUED", "total_records": len(req.Invitations), "message": "Bulk invitation job registered successfully",
	})
}

func (h *AdminInvitationHandler) GetJob(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid job_id", "errors": fiber.Map{}})
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "job not found", "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusOK).JSON(job)
}

func (h *AdminInvitationHandler) RetryFailed(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid job_id", "errors": fiber.Map{}})
	}
	items, err := h.svc.GetFailedOrDeferredItems(c.Context(), jobID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "failed to load retry items", "errors": fiber.Map{}})
	}
	if len(items) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "no failed or deferred items to retry", "count": 0})
	}
	// Re-enqueue as fresh batch tasks for each item group or individually; here enqueue one batch with item IDs.
	itemIDs := make([]string, len(items))
	for i, it := range items {
		itemIDs[i] = it.ID.String()
	}
	payload, _ := json.Marshal(map[string]interface{}{"job_id": jobID.String(), "item_ids": itemIDs, "retry": true})
	_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("admin:invitation:retry", payload), asynq.Queue("admin_invitation"))
	if enqueueErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "retry enqueue failed", "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"job_id": jobID.String(), "message": "retry queued", "count": len(items)})
}

func (h *AdminInvitationHandler) Events(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.SendString("event: error\ndata: bad job_id\n\n")
	}

	// Subscribe to redis channel
	channel := "bulk_progress_" + jobIDStr
	sub := h.redis.Subscribe(c.Context(), channel)
	defer func() { _ = sub.Close() }()

	// Send initial state
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err == nil && job != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"status": job.Status, "succeeded": job.SucceededCount, "failed": job.FailedCount, "deferred": job.DeferredCount, "total": job.TotalRecords,
		})
		_ = c.SendString(fmt.Sprintf("event: progress\ndata: %s\n\n", data))
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-sub.Channel():
			if msg != nil {
				_ = c.SendString(fmt.Sprintf("event: progress\ndata: %s\n\n", msg.Payload))
			}
		case <-ticker.C:
			_ = c.SendString("event: heartbeat\ndata: \n\n")
		case <-c.Context().Done():
			return nil
		}
	}
}

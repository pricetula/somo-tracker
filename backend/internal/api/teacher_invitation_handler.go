package api

import (
	"bufio"
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

type TeacherInvitationHandler struct {
	svc    services.TeacherInvitationServiceInterface
	stytch *stytch.Client
	asynq  *asynq.Client
	redis  *redis.Client
	logger *zap.Logger
}

func NewTeacherInvitationHandler(svc services.TeacherInvitationServiceInterface, cli *stytch.Client, asynqClient *asynq.Client, redisClient *redis.Client, logger *zap.Logger) *TeacherInvitationHandler {
	return &TeacherInvitationHandler{svc: svc, stytch: cli, asynq: asynqClient, redis: redisClient, logger: logger.With(zap.String("handler", "teacher_invitation"))}
}

func (h *TeacherInvitationHandler) HandleInvites(c fiber.Ctx) error {
	// Extract locals
	schoolIDStr := c.Locals("active_school_id")
	tenantIDStr := c.Locals("tenant_id")
	adminUserIDStr := c.Locals("user_id")

	schoolID, err1 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	adminUserID, err3 := uuid.Parse(fmt.Sprintf("%v", adminUserIDStr))
	if err1 != nil || err2 != nil || err3 != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code": "unauthorized", "message": "missing session context", "errors": fiber.Map{},
		})
	}
	// Role guard: only SCHOOL_ADMIN can create bulk invites
	hasAdmin, err := h.svc.UserHasAdminRole(c.Context(), schoolID, adminUserID)
	if err != nil {
		h.logger.Error("admin role check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "failed to verify permissions", "errors": fiber.Map{},
		})
	}
	if !hasAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code": "forbidden", "message": "insufficient permissions", "errors": fiber.Map{},
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

	// Idempotency key is required – check before expensive validation
	idempotencyKey := c.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "bad_request", "message": "Idempotency-Key header is required", "errors": fiber.Map{"Idempotency-Key": []string{"required"}},
		})
	}

	// Check existing job by idempotency – avoid re-processing
	if existing, err := h.svc.GetJobByIdempotency(c.Context(), tenantID, idempotencyKey); err == nil && existing != nil {
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"job_id": existing.ID, "status": existing.Status, "total_records": existing.TotalRecords, "message": "existing job returned",
		})
	}

	validationErrors := make(map[string][]string)
	seen := make(map[string]struct{})
	var dedupedRows []InvitationRow
	for idx, row := range req.Invitations {
		email := strings.TrimSpace(row.Email)
		fullName := strings.TrimSpace(row.FullName)
		hasError := false
		if email == "" {
			validationErrors[fmt.Sprintf("invitations[%d].email", idx)] = append(validationErrors[fmt.Sprintf("invitations[%d].email", idx)], "required")
			hasError = true
		} else if _, parseErr := mail.ParseAddress(email); parseErr != nil {
			validationErrors[fmt.Sprintf("invitations[%d].email", idx)] = append(validationErrors[fmt.Sprintf("invitations[%d].email", idx)], "invalid email format")
			hasError = true
		}
		if fullName == "" {
			validationErrors[fmt.Sprintf("invitations[%d].full_name", idx)] = append(validationErrors[fmt.Sprintf("invitations[%d].full_name", idx)], "required")
			hasError = true
		}
		if hasError {
			continue
		}
		lower := strings.ToLower(email)
		if _, exists := seen[lower]; exists {
			// Skip duplicate silently – first occurrence wins
			continue
		}
		seen[lower] = struct{}{}
		dedupedRows = append(dedupedRows, InvitationRow{Email: email, FullName: fullName})
	}
	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "validation_failed", "message": "some invitation rows are invalid", "errors": validationErrors,
		})
	}
	if len(dedupedRows) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "bad_request", "message": "no valid invitation rows after deduplication", "errors": fiber.Map{"invitations": []string{"no valid rows"}},
		})
	}

	// Build items
	items := make([]services.InvitationItem, len(dedupedRows))
	for i, r := range dedupedRows {
		items[i] = services.InvitationItem{
			ID:       uuid.New(),
			Email:    r.Email,
			FullName: r.FullName,
			Role:     "TEACHER",
		}
	}

	// Create job + items atomically
	jobID, err := h.svc.CreateBulkJobWithItems(c.Context(), schoolID, tenantID, adminUserID, idempotencyKey, items)
	if err != nil {
		h.logger.Error("bulk job creation failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "failed to create bulk job", "errors": fiber.Map{},
		})
	}

	// Enqueue batches of 100
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
				"job_id": jobID.String(), "batch_index": b, "item_ids": batchItemIDs,
			})
			if marshalErr != nil {
				h.logger.Error("asynq payload marshal failed", zap.Error(marshalErr))
				continue
			}
			_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("teacher:invitation:batch", payload), asynq.Queue("teacher_invitation"), asynq.TaskID(fmt.Sprintf("%s_%d", jobID.String(), b)))
			if enqueueErr != nil {
				h.logger.Error("asynq enqueue failed", zap.Error(enqueueErr))
				continue
			}
		}
	}

	// Publish initial progress with graceful degradation
	if h.redis != nil {
		if err := h.redis.Publish(c.Context(), "bulk_progress_"+jobID.String(), `{"status":"QUEUED","total":`+fmt.Sprintf("%d", len(dedupedRows))+`}`).Err(); err != nil {
			h.logger.Warn("redis publish failed, continuing without progress broadcast", zap.Error(err), zap.String("job_id", jobID.String()))
		}
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id": jobID.String(), "status": "QUEUED", "total_records": len(dedupedRows), "message": "Bulk invitation job registered successfully",
	})
}

func (h *TeacherInvitationHandler) GetJob(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid job_id", "errors": fiber.Map{}})
	}
	// Load session context for ownership check
	schoolIDStr := c.Locals("active_school_id")
	tenantIDStr := c.Locals("tenant_id")
	schoolID, err1 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	if err1 != nil || err2 != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "missing session context", "errors": fiber.Map{}})
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "job not found", "errors": fiber.Map{}})
	}
	if job.TenantID != tenantID || job.SchoolID != schoolID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "job not found", "errors": fiber.Map{}})
	}
	return c.Status(fiber.StatusOK).JSON(job)
}

func (h *TeacherInvitationHandler) RetryFailed(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "bad_request", "message": "invalid job_id", "errors": fiber.Map{}})
	}
	// Ownership check
	schoolIDStr := c.Locals("active_school_id")
	tenantIDStr := c.Locals("tenant_id")
	schoolID, err1 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	if err1 != nil || err2 != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "missing session context", "errors": fiber.Map{}})
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID || job.SchoolID != schoolID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "job not found", "errors": fiber.Map{}})
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
	if h.asynq != nil {
		payload, _ := json.Marshal(map[string]interface{}{"job_id": jobID.String(), "item_ids": itemIDs, "retry": true})
		_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("teacher:invitation:retry", payload), asynq.Queue("teacher_invitation"))
		if enqueueErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal_error", "message": "retry enqueue failed", "errors": fiber.Map{}})
		}
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"job_id": jobID.String(), "message": "retry queued", "count": len(items)})
}

func (h *TeacherInvitationHandler) Events(c fiber.Ctx) error {
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
	// Ownership check
	tenantIDStr := c.Locals("tenant_id")
	tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	if err2 != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.SendString("event: error\ndata: missing session locals\n\n")
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID { // school check relaxed for admin flow
		c.Status(fiber.StatusNotFound)
		return c.SendString("event: error\ndata: job not found\n\n")
	}

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		initData, _ := json.Marshal(map[string]interface{}{
			"status": job.Status, "succeeded": job.SucceededCount, "failed": job.FailedCount, "deferred": job.DeferredCount, "total": job.TotalRecords,
		})
		_, _ = fmt.Fprintf(w, "event: progress\ndata: %s\n\n", initData)
		_ = w.Flush()

		// Subscribe to redis channel
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
				_, _ = fmt.Fprintf(w, "event: progress\ndata: %s\n\n", msg.Payload)
				_ = w.Flush()
			case <-ticker.C:
				_, _ = fmt.Fprintf(w, "event: heartbeat\ndata: \n\n")
				_ = w.Flush()
			case <-c.Context().Done():
				return
			}
		}
	})
	return nil
}

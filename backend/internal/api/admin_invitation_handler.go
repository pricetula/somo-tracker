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

type AdminInvitationHandler struct {
	svc    services.AdminInvitationService
	stytch stytch.Inviter
	asynq  *asynq.Client
	redis  *redis.Client
	logger *zap.Logger
}

func NewAdminInvitationHandler(svc services.AdminInvitationService, cli stytch.Inviter, asynqClient *asynq.Client, redisClient *redis.Client, logger *zap.Logger) *AdminInvitationHandler {
	return &AdminInvitationHandler{svc: svc, stytch: cli, asynq: asynqClient, redis: redisClient, logger: logger.With(zap.String("handler", "admin_invitation"))}
}

type InvitationRow struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type BulkInvitationRequest struct {
	Invitations []InvitationRow `json:"invitations"`
}

type BulkInvitationResponse struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	TotalRecords int    `json:"total_records,omitempty"`
	Message      string `json:"message"`
}

type InvitationJob struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	TotalRecords int    `json:"total_records"`
}

type InvitationRetryResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// @Summary Bulk invite admins
// @Tags Admin Invitations
// @Accept json
// @Produce json
// @Param body body BulkInvitationRequest true "Bulk invitation payload"
// @Success 200 {object} BulkInvitationResponse
// @Success 202 {object} BulkInvitationResponse
// @Router /api/admins/invitations [post]
func (h *AdminInvitationHandler) HandleInvites(c fiber.Ctx) error {
	// Extract locals
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return WriteError(c, ErrUnauthorized("tenant not found"))
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid tenant id", nil))
	}
	adminUserIDStr, ok := c.Locals("user_id").(string)
	if !ok || adminUserIDStr == "" {
		return WriteError(c, ErrUnauthorized("user not found"))
	}
	adminUserID, err := uuid.Parse(adminUserIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid user id", nil))
	}

	// Role guard: only SCHOOL_ADMIN can create bulk invites
	hasAdmin, err := h.svc.UserHasAdminRole(c.Context(), schoolID, adminUserID)
	if err != nil {
		h.logger.Error("admin role check failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to verify permissions"))
	}
	if !hasAdmin {
		return WriteError(c, ErrForbidden("insufficient_permissions", "insufficient permissions"))
	}

	var req BulkInvitationRequest
	if err := c.Bind().Body(&req); err != nil {
		return WriteError(c, ErrBadRequest("invalid request body", map[string][]string{"body": {"malformed json"}}))
	}

	if len(req.Invitations) == 0 {
		return WriteError(c, ErrBadRequest("invitations array is required", map[string][]string{"invitations": {"required"}}))
	}
	if len(req.Invitations) > 1000 {
		return WriteError(c, ErrBadRequest("exceeds maximum of 1000 rows", map[string][]string{"invitations": {"max 1000"}}))
	}

	// Idempotency key is required – check before expensive validation
	idempotencyKey, err := ValidateIDempotencyKey(c)
	if err != nil {
		return err
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
	var deduplicatedIndices []int
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
			// Track duplicate but skip
			deduplicatedIndices = append(deduplicatedIndices, idx)
			continue
		}
		seen[lower] = struct{}{}
		dedupedRows = append(dedupedRows, InvitationRow{Email: email, FullName: fullName})
	}
	if len(validationErrors) > 0 {
		return WriteError(c, ErrBadRequest("some invitation rows are invalid", validationErrors))
	}
	if len(dedupedRows) == 0 {
		return WriteError(c, ErrBadRequest("no valid invitation rows after deduplication", map[string][]string{"invitations": {"no valid rows"}}))
	}

	// Build items
	items := make([]services.InvitationItem, len(dedupedRows))
	for i, r := range dedupedRows {
		items[i] = services.InvitationItem{
			ID:       uuid.New(),
			Email:    r.Email,
			FullName: r.FullName,
			Role:     "ADMIN",
		}
	}

	// Create job + items atomically (with service-layer authz)
	jobID, err := h.svc.CreateBulkJobWithItemsWithAuth(c.Context(), schoolID, tenantID, adminUserID, adminUserID, idempotencyKey, items)
	if err != nil {
		h.logger.Error("bulk job creation failed", zap.Error(err))
		return WriteError(c, ErrInternal("failed to create bulk job"))
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
			_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("admin:invitation:batch", payload), asynq.Queue("admin_invitation"), asynq.TaskID(fmt.Sprintf("%s_%d", jobID.String(), b)))
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

	response := fiber.Map{
		"job_id":        jobID.String(),
		"status":        "QUEUED",
		"total_records": len(dedupedRows),
		"message":       "Bulk invitation job registered successfully",
		"errors":        fiber.Map{},
	}
	if len(deduplicatedIndices) > 0 {
		response["deduplicated_count"] = len(deduplicatedIndices)
		response["deduplicated_indices"] = deduplicatedIndices
		response["message"] = fmt.Sprintf("Bulk invitation job registered successfully (%d duplicates removed)", len(deduplicatedIndices))
	}

	return c.Status(fiber.StatusAccepted).JSON(response)
}

// @Summary Get invitation job status
// @Tags Admin Invitations
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} InvitationJob
// @Router /api/admins/invitations/jobs/{job_id} [get]
func (h *AdminInvitationHandler) GetJob(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid job_id", nil))
	}
	// Load session context for ownership check
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return WriteError(c, ErrUnauthorized("tenant not found"))
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid tenant id", nil))
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil {
		return WriteError(c, ErrNotFound("job not found"))
	}
	if job.TenantID != tenantID || job.SchoolID != schoolID {
		return WriteError(c, ErrNotFound("job not found"))
	}
	return c.Status(fiber.StatusOK).JSON(job)
}

// @Summary Retry failed invitation jobs
// @Tags Admin Invitations
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} InvitationRetryResponse
// @Router /api/admins/invitations/jobs/{job_id}/retry-failed [post]
func (h *AdminInvitationHandler) RetryFailed(c fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid job_id", nil))
	}
	// Ownership check
	schoolIDStr, ok := c.Locals("active_school_id").(string)
	if !ok || schoolIDStr == "" {
		return WriteError(c, ErrUnauthorized("active school not found"))
	}
	schoolID, err := uuid.Parse(schoolIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid school id", nil))
	}
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return WriteError(c, ErrUnauthorized("tenant not found"))
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid tenant id", nil))
	}
	adminUserIDStr, ok := c.Locals("user_id").(string)
	if !ok || adminUserIDStr == "" {
		return WriteError(c, ErrUnauthorized("user not found"))
	}
	adminUserID, err := uuid.Parse(adminUserIDStr)
	if err != nil {
		return WriteError(c, ErrBadRequest("invalid user id", nil))
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID || job.SchoolID != schoolID {
		return WriteError(c, ErrNotFound("job not found"))
	}
	items, err := h.svc.GetFailedOrDeferredItemsWithAuth(c.Context(), jobID, schoolID, adminUserID)
	if err != nil {
		return WriteError(c, ErrInternal("failed to load retry items"))
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
		_, enqueueErr := h.asynq.EnqueueContext(c.Context(), asynq.NewTask("admin:invitation:retry", payload), asynq.Queue("admin_invitation"))
		if enqueueErr != nil {
			return WriteError(c, ErrInternal("retry enqueue failed"))
		}
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"job_id": jobID.String(), "message": "retry queued", "count": len(items)})
}

// @Summary Get invitation job events
// @Tags Admin Invitations
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/admins/invitations/jobs/{job_id}/events [get]
func (h *AdminInvitationHandler) Events(c fiber.Ctx) error {
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
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		c.Status(fiber.StatusUnauthorized)
		return c.SendString("event: error\ndata: missing session locals\n\n")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.SendString("event: error\ndata: missing session locals\n\n")
	}
	job, err := h.svc.GetJob(c.Context(), jobID)
	if err != nil || job == nil || job.TenantID != tenantID { // school check relaxed for admin flow
		c.Status(fiber.StatusNotFound)
		return c.SendString("event: error\ndata: job not found\n\n")
	}

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		initData, err := json.Marshal(map[string]interface{}{
			"status": job.Status, "succeeded": job.SucceededCount, "failed": job.FailedCount, "deferred": job.DeferredCount, "total": job.TotalRecords,
		})
		if err != nil {
			h.logger.Warn("failed to marshal init progress data", zap.Error(err), zap.String("job_id", jobID.String()))
			initData = []byte("{}")
		}
		if _, err := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", initData); err != nil {
			h.logger.Warn("failed to write init progress", zap.Error(err), zap.String("job_id", jobID.String()))
		}
		if err := w.Flush(); err != nil {
			h.logger.Warn("failed to flush init progress", zap.Error(err), zap.String("job_id", jobID.String()))
		}

		// Subscribe to redis channel
		channel := "bulk_progress_" + jobIDStr
		sub := h.redis.Subscribe(c.Context(), channel)
		defer func() {
			if err := sub.Close(); err != nil {
				h.logger.Warn("redis sub close failed", zap.Error(err), zap.String("job_id", jobID.String()))
			}
		}()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-sub.Channel():
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", msg.Payload); err != nil {
					h.logger.Warn("failed to write progress", zap.Error(err), zap.String("job_id", jobID.String()))
				}
				if err := w.Flush(); err != nil {
					h.logger.Warn("failed to flush progress", zap.Error(err), zap.String("job_id", jobID.String()))
				}
			case <-ticker.C:
				if _, err := fmt.Fprintf(w, "event: heartbeat\ndata: \n\n"); err != nil {
					h.logger.Warn("failed to write heartbeat", zap.Error(err), zap.String("job_id", jobID.String()))
				}
				if err := w.Flush(); err != nil {
					h.logger.Warn("failed to flush heartbeat", zap.Error(err), zap.String("job_id", jobID.String()))
				}
			case <-c.Context().Done():
				return
			}
		}
	})
	return nil
}

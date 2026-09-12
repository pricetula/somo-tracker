package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"somotracker/backend/internal/services"
	"somotracker/backend/internal/stytch"
)

type AdminInvitationProcessor struct {
	svc       services.AdminInvitationService
	stytchCli *stytch.Client
	logger    *zap.Logger
	redis     RedisPublisher
}

type RedisPublisher interface {
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
}

type InvitationBatchPayload struct {
	JobID      string   `json:"job_id"`
	BatchIndex int      `json:"batch_index"`
	ItemIDs    []string `json:"item_ids"`
}

type InvitationRetryPayload struct {
	JobID   string   `json:"job_id"`
	ItemIDs []string `json:"item_ids"`
	Retry   bool     `json:"retry"`
}

type InvitationPayload struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

func NewAdminInvitationProcessor(svc services.AdminInvitationService, cli *stytch.Client, logger *zap.Logger, pub RedisPublisher) *AdminInvitationProcessor {
	return &AdminInvitationProcessor{
		svc:       svc,
		stytchCli: cli,
		logger:    logger.With(zap.String("worker", "invitation")),
		redis:     pub,
	}
}

// ProcessTask handles "admin:invitation:batch" tasks.
// Processes each item sequentially, updating status and publishing progress.
func (p *AdminInvitationProcessor) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload InvitationBatchPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("bad job_id: %w", err)
	}

	p.logger.Info("processing batch",
		zap.String("job_id", payload.JobID),
		zap.Int("batch_index", payload.BatchIndex),
		zap.Int("item_count", len(payload.ItemIDs)),
	)

	// Mark job as PROCESSING if still QUEUED
	_ = p.svc.UpdateJobStatus(ctx, jobID, "PROCESSING")

	// Process each item in the batch sequentially
	for _, itemIDStr := range payload.ItemIDs {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			p.logger.Warn("bad item id", zap.String("item_id", itemIDStr))
			continue
		}
		p.processSingleItem(ctx, jobID, itemID)
	}

	// After batch, derive and update job status
	p.deriveJobStatus(ctx, jobID)
	return nil
}

// ProcessRetryTask handles "admin:invitation:retry" tasks.
// Re-enqueues only FAILED/DEFERRED items.
func (p *AdminInvitationProcessor) ProcessRetryTask(ctx context.Context, task *asynq.Task) error {
	var payload InvitationRetryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal retry payload: %w", err)
	}
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("bad job_id: %w", err)
	}

	p.logger.Info("processing retry",
		zap.String("job_id", payload.JobID),
		zap.Int("item_count", len(payload.ItemIDs)),
	)

	// Mark job as PROCESSING
	_ = p.svc.UpdateJobStatus(ctx, jobID, "PROCESSING")

	for _, itemIDStr := range payload.ItemIDs {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			p.logger.Warn("bad item id in retry", zap.String("item_id", itemIDStr))
			continue
		}
		p.processSingleItem(ctx, jobID, itemID)
	}

	p.deriveJobStatus(ctx, jobID)
	return nil
}

func (p *AdminInvitationProcessor) processSingleItem(ctx context.Context, jobID, itemID uuid.UUID) {
	// Fetch item
	item, err := p.svc.GetItemByID(ctx, itemID)
	if err != nil {
		p.logger.Error("failed to fetch item", zap.Error(err), zap.String("item_id", itemID.String()))
		return
	}

	// Skip if already terminal
	if item.Status == "SUCCEEDED" || item.Status == "FAILED" {
		return
	}

	// Parse payload
	var itemPayload InvitationPayload
	if err := json.Unmarshal(item.Payload, &itemPayload); err != nil {
		p.logger.Error("failed to unmarshal item payload", zap.Error(err), zap.String("item_id", itemID.String()))
		_ = p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "invalid payload", item.AttemptCount+1)
		p.publishProgress(ctx, jobID)
		return
	}

	// Update status to PROCESSING
	newAttempts := item.AttemptCount + 1
	_ = p.svc.UpdateItemStatus(ctx, itemID, "PROCESSING", nil, "", newAttempts)
	p.publishProgress(ctx, jobID)

	// Call Stytch to invite member
	result, err := p.stytchCli.InviteMember(ctx, itemPayload.Email, itemPayload.FullName, itemPayload.Role, jobID.String())
	if err != nil {
		p.handleStytchError(ctx, jobID, itemID, newAttempts, err, itemPayload)
		p.publishProgress(ctx, jobID)
		return
	}

	// Success - store result
	resultJSON, _ := json.Marshal(map[string]string{
		"stytch_invite_id": result.StytchInviteID,
		"stytch_member_id": result.StytchMemberID,
	})
	_ = p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", resultJSON, "", newAttempts)
	_ = p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0)
	p.logger.Info("invitation succeeded", zap.String("item_id", itemID.String()), zap.String("email", itemPayload.Email))
	p.publishProgress(ctx, jobID)
}

func (p *AdminInvitationProcessor) handleStytchError(ctx context.Context, jobID, itemID uuid.UUID, attempts int, err error, payload InvitationPayload) {
	retry, _, duplicate, reason := p.stytchCli.ClassifyStytchError(err)

	if duplicate {
		// Duplicate email - treat as success (idempotent)
		resultJSON, _ := json.Marshal(map[string]string{
			"stytch_invite_id": "duplicate",
			"stytch_member_id": "",
			"note":             "member already exists",
		})
		_ = p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", resultJSON, "", attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0)
		p.logger.Info("invitation duplicate (treated as success)", zap.String("item_id", itemID.String()), zap.String("email", payload.Email))
		return
	}

	if retry {
		// Transient error - mark DEFERRED, will be retried via Asynq retry
		_ = p.svc.UpdateItemStatus(ctx, itemID, "DEFERRED", nil, reason, attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 0, 1)
		p.logger.Warn("invitation deferred (retryable)", zap.String("item_id", itemID.String()), zap.String("reason", reason), zap.Error(err))
		return
	}

	// Permanent failure
	resultJSON, _ := json.Marshal(map[string]string{"error": reason})
	_ = p.svc.UpdateItemStatus(ctx, itemID, "FAILED", resultJSON, reason, attempts)
	_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0)
	p.logger.Error("invitation failed permanently", zap.String("item_id", itemID.String()), zap.String("email", payload.Email), zap.String("reason", reason))
}

func (p *AdminInvitationProcessor) deriveJobStatus(ctx context.Context, jobID uuid.UUID) {
	items, err := p.svc.GetItemsByJobID(ctx, jobID)
	if err != nil {
		p.logger.Error("failed to get items for status derivation", zap.Error(err), zap.String("job_id", jobID.String()))
		return
	}

	var pending, processing, deferred int
	for _, item := range items {
		switch item.Status {
		case "PENDING":
			pending++
		case "PROCESSING":
			processing++
		case "DEFERRED":
			deferred++
		}
	}

	var newStatus string
	if pending > 0 || processing > 0 {
		newStatus = "PROCESSING"
	} else if deferred > 0 {
		// All items processed but some deferred - job completes with deferred
		newStatus = "COMPLETED_WITH_ERRORS"
	} else {
		// All items terminal (SUCCEEDED or FAILED)
		job, _ := p.svc.GetJob(ctx, jobID)
		if job != nil && job.FailedCount > 0 {
			newStatus = "COMPLETED_WITH_ERRORS"
		} else {
			newStatus = "COMPLETED"
		}
	}

	_ = p.svc.UpdateJobStatus(ctx, jobID, newStatus)
	p.logger.Info("job status derived", zap.String("job_id", jobID.String()), zap.String("status", newStatus), zap.Int("pending", pending), zap.Int("processing", processing), zap.Int("deferred", deferred))
	p.publishProgress(ctx, jobID)
}

func (p *AdminInvitationProcessor) publishProgress(ctx context.Context, jobID uuid.UUID) {
	if p.redis == nil {
		return
	}
	job, err := p.svc.GetJob(ctx, jobID)
	if err != nil || job == nil {
		return
	}
	data, _ := json.Marshal(map[string]interface{}{
		"status":    job.Status,
		"succeeded": job.SucceededCount,
		"failed":    job.FailedCount,
		"deferred":  job.DeferredCount,
		"total":     job.TotalRecords,
	})
	_ = p.redis.Publish(ctx, "bulk_progress_"+jobID.String(), string(data)).Err()
}

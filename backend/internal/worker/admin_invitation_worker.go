package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	// Unmarshal the asynq batch payload: job_id, batch_index, item_ids list.
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
	if len(payload.ItemIDs) == 0 {
		p.logger.Warn("batch payload has zero item_ids — payload contract mismatch", zap.String("job_id", payload.JobID), zap.Int("batch_index", payload.BatchIndex))
	}

	// Mark job as PROCESSING if still QUEUED
	_ = p.svc.UpdateJobStatus(ctx, jobID, "PROCESSING")

	// Process items with limited concurrency to avoid overwhelming Stytch.
	// workerConcurrency = 10: max 10 concurrent Stytch API calls at once.
	const workerConcurrency = 10
	// Semaphore: channel acts as a token bucket. Capacity = max concurrent goroutines.
	sem := make(chan struct{}, workerConcurrency)
	var wg sync.WaitGroup
	for _, itemIDStr := range payload.ItemIDs {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			p.logger.Warn("bad item id", zap.String("item_id", itemIDStr))
			continue
		}
		wg.Add(1)
		// Block if 10 goroutines already running; acquire a semaphore slot.
		sem <- struct{}{}
		go func() {
			// Signal finished when this item completes.
			defer wg.Done()
			// Release the slot so the next queued item can start.
			defer func() { <-sem }()
			p.processSingleItem(ctx, jobID, itemID)
		}()
	}
	// Wait for all goroutines in this batch to finish before deriving job status.
	wg.Wait()

	// After batch: update derived status (COMPLETED / COMPLETED_WITH_ERRORS / PROCESSING).
	p.deriveJobStatus(ctx, jobID)
	p.logger.Info("batch completed",
		zap.String("job_id", payload.JobID),
		zap.Int("batch_index", payload.BatchIndex),
		zap.Int("items_processed", len(payload.ItemIDs)),
	)
	return nil
}

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

	const workerConcurrency = 10
	sem := make(chan struct{}, workerConcurrency)
	var wg sync.WaitGroup
	for _, itemIDStr := range payload.ItemIDs {
		itemIDStr := itemIDStr
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			p.logger.Warn("bad item id in retry", zap.String("item_id", itemIDStr))
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			p.processSingleItem(ctx, jobID, itemID)
		}()
	}
	wg.Wait()

	p.deriveJobStatus(ctx, jobID)
	return nil
}

func (p *AdminInvitationProcessor) processSingleItem(ctx context.Context, jobID, itemID uuid.UUID) {
	// Load item directly, no per-item locking
	item, err := p.svc.GetItemByID(ctx, itemID)
	if err != nil {
		p.logger.Error("failed to get item", zap.Error(err), zap.String("item_id", itemID.String()))
		return
	}
	if item.Status != "PENDING" && item.Status != "DEFERRED" {
		return
	}
	// Mark as processing and bump attempt count
	attempts := item.AttemptCount + 1
	if err := p.svc.UpdateItemStatus(ctx, itemID, "PROCESSING", nil, "", attempts); err != nil {
		p.logger.Error("failed to mark item processing", zap.Error(err), zap.String("item_id", itemID.String()))
		return
	}

	// Parse payload
	var itemPayload InvitationPayload
	if err := json.Unmarshal(item.Payload, &itemPayload); err != nil {
		p.logger.Error("failed to unmarshal item payload", zap.Error(err), zap.String("item_id", itemID.String()))
		if updErr := p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "invalid payload", attempts); updErr != nil {
			p.logger.Error("failed to update item status to FAILED", zap.Error(updErr))
		}
		return
	}

	// Call Stytch to invite member — use job's tenant_id, not job_id
	job, errJob := p.svc.GetJob(ctx, jobID)
	if errJob != nil || job == nil {
		p.logger.Error("failed to fetch job for tenant_id", zap.Error(errJob), zap.String("job_id", jobID.String()))
		p.handleStytchError(ctx, jobID, itemID, attempts, fmt.Errorf("job not found for tenant"), itemPayload)
		return
	}
	orgID, errOrg := p.svc.GetStytchOrgID(ctx, job.TenantID)
	if errOrg != nil {
		p.logger.Error("failed to get stytch org id for tenant", zap.Error(errOrg), zap.String("tenant_id", job.TenantID.String()), zap.String("job_id", jobID.String()))
		p.handleStytchError(ctx, jobID, itemID, attempts, fmt.Errorf("stytch org not found for tenant"), itemPayload)
		return
	}
	result, err := p.stytchCli.InviteMember(ctx, itemPayload.Email, itemPayload.FullName, itemPayload.Role, orgID)
	if err != nil {
		p.handleStytchError(ctx, jobID, itemID, attempts, err, itemPayload)
		return
	}

	// Provision local records for successful invite
	if err := p.svc.ProvisionInvitee(ctx, job.TenantID, job.SchoolID, job.CreatedBy, itemPayload.Email, itemPayload.FullName, itemPayload.Role, result.StytchMemberID); err != nil {
		p.logger.Error("failed to provision invitee records", zap.Error(err), zap.String("item_id", itemID.String()), zap.String("email", itemPayload.Email))
		// Treat provisioning failure as a retryable error
		p.handleStytchError(ctx, jobID, itemID, attempts, fmt.Errorf("provision failed: %w", err), itemPayload)
		return
	}

	// Success - store result
	resultJSON, _ := json.Marshal(map[string]string{
		"stytch_invite_id": result.StytchInviteID,
		"stytch_member_id": result.StytchMemberID,
	})
	if err := p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", resultJSON, "", attempts); err != nil {
		p.logger.Error("failed to update item status to SUCCEEDED", zap.Error(err), zap.String("item_id", itemID.String()))
	}
	if err := p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0); err != nil {
		p.logger.Error("failed to increment job counters for success", zap.Error(err), zap.String("job_id", jobID.String()))
	}
	p.logger.Info("invitation succeeded", zap.String("item_id", itemID.String()), zap.String("email", itemPayload.Email))
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
		if err := p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", resultJSON, "", attempts); err != nil {
			p.logger.Error("failed to update item status to SUCCEEDED for duplicate", zap.Error(err), zap.String("item_id", itemID.String()))
		}
		if err := p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0); err != nil {
			p.logger.Error("failed to increment job counters for duplicate", zap.Error(err), zap.String("job_id", jobID.String()))
		}
		p.logger.Info("invitation duplicate (treated as success)", zap.String("item_id", itemID.String()), zap.String("email", payload.Email))
		return
	}

	if retry {
		// Transient error - mark DEFERRED, will be retried via Asynq retry
		if err := p.svc.UpdateItemStatus(ctx, itemID, "DEFERRED", nil, reason, attempts); err != nil {
			p.logger.Error("failed to update item status to DEFERRED", zap.Error(err), zap.String("item_id", itemID.String()))
		}
		if err := p.svc.IncrementJobCounters(ctx, jobID, 0, 0, 1); err != nil {
			p.logger.Error("failed to increment job counters for deferred", zap.Error(err), zap.String("job_id", jobID.String()))
		}
		p.logger.Warn("invitation deferred (retryable)", zap.String("item_id", itemID.String()), zap.String("reason", reason), zap.Error(err))
		return
	}

	// Permanent failure
	resultJSON, _ := json.Marshal(map[string]string{"error": reason})
	if err := p.svc.UpdateItemStatus(ctx, itemID, "FAILED", resultJSON, reason, attempts); err != nil {
		p.logger.Error("failed to update item status to FAILED", zap.Error(err), zap.String("item_id", itemID.String()))
	}
	if err := p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0); err != nil {
		p.logger.Error("failed to increment job counters for failure", zap.Error(err), zap.String("job_id", jobID.String()))
	}
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
	if err := p.redis.Publish(ctx, "bulk_progress_"+jobID.String(), string(data)).Err(); err != nil {
		p.logger.Warn("redis progress publish failed", zap.Error(err), zap.String("job_id", jobID.String()))
	}
}

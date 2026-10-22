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

// InvitationService defines the minimal interface required by the bulk invitation processor.
type InvitationService interface {
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*services.BulkJobItem, error)
	GetJob(ctx context.Context, jobID uuid.UUID) (*services.BulkJob, error)
	GetStytchOrgID(ctx context.Context, tenantID uuid.UUID) (string, error)
	ProvisionInvitee(ctx context.Context, tenantID, schoolID, invitedBy uuid.UUID, email, fullName, role, stytchMemberID string) error
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status string, result []byte, lastError string, attempts int) error
	IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeeded, failed, deferred int) error
	UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error
	GetItemsByJobID(ctx context.Context, jobID uuid.UUID) ([]services.BulkJobItem, error)
}

// BulkInvitationProcessor is a generic processor for all role-based bulk invitations.
type BulkInvitationProcessor struct {
	svc       InvitationService
	stytchCli stytch.Inviter
	logger    *zap.Logger
	redis     RedisPublisher
}

func NewBulkInvitationProcessor(svc InvitationService, cli stytch.Inviter, logger *zap.Logger, pub RedisPublisher) *BulkInvitationProcessor {
	return &BulkInvitationProcessor{
		svc:       svc,
		stytchCli: cli,
		logger:    logger.With(zap.String("worker", "bulk_invitation")),
		redis:     pub,
	}
}

func (p *BulkInvitationProcessor) ProcessTask(ctx context.Context, task *asynq.Task) error {
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
	if len(payload.ItemIDs) == 0 {
		p.logger.Warn("batch payload has zero item_ids", zap.String("job_id", payload.JobID))
	}

	if err := p.svc.UpdateJobStatus(ctx, jobID, "PROCESSING"); err != nil {
		p.logger.Warn("failed to update job status to PROCESSING", zap.String("job_id", jobID.String()), zap.Error(err))
	}

	const workerConcurrency = 10
	sem := make(chan struct{}, workerConcurrency)
	var wg sync.WaitGroup
	for _, itemIDStr := range payload.ItemIDs {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			p.logger.Warn("bad item id", zap.String("item_id", itemIDStr))
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
	p.logger.Info("batch completed", zap.String("job_id", payload.JobID), zap.Int("items_processed", len(payload.ItemIDs)))
	return nil
}

func (p *BulkInvitationProcessor) ProcessRetryTask(ctx context.Context, task *asynq.Task) error {
	var payload InvitationRetryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal retry payload: %w", err)
	}
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("bad job_id: %w", err)
	}

	p.logger.Info("processing retry", zap.String("job_id", payload.JobID), zap.Int("item_count", len(payload.ItemIDs)))

	if err := p.svc.UpdateJobStatus(ctx, jobID, "PROCESSING"); err != nil {
		p.logger.Warn("failed to update job status to PROCESSING for retry", zap.String("job_id", jobID.String()), zap.Error(err))
	}

	const workerConcurrency = 10
	sem := make(chan struct{}, workerConcurrency)
	var wg sync.WaitGroup
	for _, itemIDStr := range payload.ItemIDs {
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

func (p *BulkInvitationProcessor) processSingleItem(ctx context.Context, jobID, itemID uuid.UUID) {
	item, err := p.svc.GetItemByID(ctx, itemID)
	if err != nil {
		p.logger.Error("failed to get item", zap.Error(err), zap.String("item_id", itemID.String()))
		return
	}
	if item.Status != "PENDING" && item.Status != "DEFERRED" {
		return
	}
	attempts := item.AttemptCount + 1
	if err := p.svc.UpdateItemStatus(ctx, itemID, "PROCESSING", nil, "", attempts); err != nil {
		p.logger.Error("failed to mark item processing", zap.Error(err), zap.String("item_id", itemID.String()))
		return
	}

	var itemPayload InvitationPayload
	if err := json.Unmarshal(item.Payload, &itemPayload); err != nil {
		p.logger.Error("failed to unmarshal item payload", zap.Error(err), zap.String("item_id", itemID.String()))
		_ = p.svc.UpdateItemStatus(ctx, itemID, "FAILED", nil, "invalid payload", attempts)
		return
	}

	job, errJob := p.svc.GetJob(ctx, jobID)
	if errJob != nil || job == nil {
		p.logger.Error("failed to fetch job", zap.Error(errJob), zap.String("job_id", jobID.String()))
		p.handleStytchError(ctx, jobID, itemID, attempts, fmt.Errorf("job not found"), itemPayload)
		return
	}
	orgID, errOrg := p.svc.GetStytchOrgID(ctx, job.TenantID)
	if errOrg != nil {
		p.logger.Error("failed to get stytch org id", zap.Error(errOrg), zap.String("tenant_id", job.TenantID.String()))
		p.handleStytchError(ctx, jobID, itemID, attempts, fmt.Errorf("stytch org not found"), itemPayload)
		return
	}
	result, err := p.stytchCli.InviteMember(ctx, itemPayload.Email, itemPayload.FullName, itemPayload.Role, orgID)
	if err != nil {
		p.handleStytchError(ctx, jobID, itemID, attempts, err, itemPayload)
		return
	}

	if err := p.svc.ProvisionInvitee(ctx, job.TenantID, job.SchoolID, job.CreatedBy, itemPayload.Email, itemPayload.FullName, itemPayload.Role, result.StytchMemberID); err != nil {
		p.logger.Error("provision invitee failed", zap.Error(err), zap.String("item_id", itemID.String()))
		p.handleStytchError(ctx, jobID, itemID, attempts, fmt.Errorf("provision failed: %w", err), itemPayload)
		return
	}

	resultJSON, _ := json.Marshal(map[string]string{
		"stytch_invite_id": result.StytchInviteID,
		"stytch_member_id": result.StytchMemberID,
	})
	if err := p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", resultJSON, "", attempts); err != nil {
		p.logger.Error("failed to update item to SUCCEEDED", zap.Error(err))
	}
	_ = p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0)
	p.logger.Info("invitation succeeded", zap.String("item_id", itemID.String()), zap.String("email", itemPayload.Email))
}

func (p *BulkInvitationProcessor) handleStytchError(ctx context.Context, jobID, itemID uuid.UUID, attempts int, err error, payload InvitationPayload) {
	retry, _, duplicate, reason := p.stytchCli.ClassifyStytchError(err)

	if duplicate {
		resultJSON, _ := json.Marshal(map[string]string{
			"stytch_invite_id": "duplicate",
			"stytch_member_id": "",
			"note":             "member already exists",
		})
		_ = p.svc.UpdateItemStatus(ctx, itemID, "SUCCEEDED", resultJSON, "", attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 1, 0, 0)
		p.logger.Info("invitation duplicate treated as success", zap.String("item_id", itemID.String()), zap.String("email", payload.Email))
		return
	}

	if retry {
		_ = p.svc.UpdateItemStatus(ctx, itemID, "DEFERRED", nil, reason, attempts)
		_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 0, 1)
		p.logger.Warn("invitation deferred", zap.String("item_id", itemID.String()), zap.String("reason", reason))
		return
	}

	resultJSON, _ := json.Marshal(map[string]string{"error": reason})
	_ = p.svc.UpdateItemStatus(ctx, itemID, "FAILED", resultJSON, reason, attempts)
	_ = p.svc.IncrementJobCounters(ctx, jobID, 0, 1, 0)
	p.logger.Error("invitation failed permanently", zap.String("item_id", itemID.String()), zap.String("email", payload.Email), zap.String("reason", reason))
}

func (p *BulkInvitationProcessor) deriveJobStatus(ctx context.Context, jobID uuid.UUID) {
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
		newStatus = "COMPLETED_WITH_ERRORS"
	} else {
		job, _ := p.svc.GetJob(ctx, jobID)
		if job != nil && job.FailedCount > 0 {
			newStatus = "COMPLETED_WITH_ERRORS"
		} else {
			newStatus = "COMPLETED"
		}
	}
	_ = p.svc.UpdateJobStatus(ctx, jobID, newStatus)
	p.logger.Info("job status derived", zap.String("job_id", jobID.String()), zap.String("status", newStatus))
	p.publishProgress(ctx, jobID)
}

func (p *BulkInvitationProcessor) publishProgress(ctx context.Context, jobID uuid.UUID) {
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
